package core

import (
	"crypto/cipher"
	"errors"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"
)

type ahriClientPartner struct {
	clientName string
	mode       int
	aesCipher  cipher.Block

	stopOnce sync.Once
	quitFlag bool
	quit     chan int

	heartbeatUnixTimeSec int64
	realConnStop         bool
	realConnLock         sync.RWMutex
	realConn             *net.TCPConn

	receiverLoopStop  bool
	receiveFrameTruck chan AhriFrame
	//key:string, value: *AhriConn
	conns sync.Map

	proxyFrameTaskSender chan *proxyFrameTask
	proxyFrameReceiver   chan AhriFrame

	willClose func(clientName string)
}

func (partner *ahriClientPartner) stop() {
	partner.stopOnce.Do(func() {
		partner.quit <- 0
		partner.quitFlag = true
		partner.willClose(partner.clientName)
		Log.NoLevelf("Removed client(%s), mode: %d", partner.clientName, partner.mode)
	})
}

func (partner *ahriClientPartner) loopStarter() {
	partner.quit = make(chan int)
	partner.receiveFrameTruck = make(chan AhriFrame)
	partner.proxyFrameReceiver = make(chan AhriFrame, AhriClientPartnerProxyFrameMaxSize)
	go func() {
		for {
			select {
			case frame := <-partner.receiveFrameTruck:
				partner.dispatcher(frame)
			case <-time.After(time.Microsecond):
				if partner.quitFlag && partner.receiverLoopStop {
					goto loopEnd
				}
			}
		}
	loopEnd:
		Log.Warn("Ahri Partner receiveFrameTruck loop stopped.")
	}()
	go partner.receiverLoop()
	go func() {
		proxyFrameSender := func(proxyFrame AhriFrame) {
			defer func() {
				proxyFrame.free()
				unknownError := recover()
				if unknownError != nil {
					if err, ok := unknownError.(error); ok {
						errMsg := err.Error()
						Log.Errorf("Unknown Error: %s", errMsg)
					} else {
						Log.Errorf("Unknown Error(%v): %v", reflect.TypeOf(unknownError), unknownError)
					}
				}
			}()
			e := partner.sender(proxyFrame)
			if e != nil {
				switch proxyFrame.frameType() {
				case AfpFrameTypeDial:
					Log.Warnf("Failed to transmit a AhriFrame(type: AfpFrameTypeDial -> AfpFrameTypeDialProxy), %v", e)
				case AfpFrameTypeDialProxyAck:
					Log.Warnf("Failed to transmit a AhriFrame(type: AfpFrameTypeDialProxyAck -> AfpFrameTypeDialAck), %v", e)
				case AfpFrameTypeProxy:
					Log.Warnf("Failed to transmit a AhriFrame(type: AfpFrameTypeDirect -> AfpFrameTypeProxy), %v", e)
				case AfpFrameTypeDirect:
					Log.Warnf("Failed to transmit a AhriFrame(type: AfpFrameTypeProxy -> AfpFrameTypeDirect), %v", e)
				}
			}
		}
		for {
			if partner.quitFlag {
				break
			}
			select {
			case proxyFrame := <-partner.proxyFrameReceiver:
				proxyFrameSender(proxyFrame)
			}
		}
	}()
	go func() {
		for {
			select {
			case <-partner.quit:
				goto loopEnd
			case <-time.After(time.Second):
				go partner.keepConn()
			}
		}
	loopEnd:
		partner.clearConn()
		Log.Warn("Ahri Partner Closed.")
	}()
}

func (partner *ahriClientPartner) clearConn() {
	partner.realConnLock.Lock()
	defer partner.realConnLock.Unlock()
	partner.realConnStop = true
	if partner.realConn != nil {
		partner.realConn.Close()
		partner.realConn = nil
		partner.conns.Range(func(key, value interface{}) bool {
			defer func() {
				recover()
			}()
			conn := value.(*AhriConn)
			conn.Close()
			return true
		})
	}
}

func (partner *ahriClientPartner) keepConn() {
	partner.realConnLock.Lock()
	defer partner.realConnLock.Unlock()
	if partner.realConn == nil {
		go partner.stop()
		return
	}
	//Check if the heartbeat of the client has timed out
	if time.Now().Unix()-partner.heartbeatUnixTimeSec > AhriTimeoutSec {
		go partner.stop()
		return
	}
	//send heartbeat
	heartbeatFrame := AhriFrame(make([]byte, 17))
	heartbeatFrame[0] = AfpFlag
	heartbeatFrame.setFrameType(AfpFrameTypeHeartbeat)
	heartbeatFrame.setFrom(AhriAddrNameAhriServer)
	heartbeatFrame.setTo(partner.clientName)
	heartbeatFrame.setConnId(0)
	heartbeatFrame = heartbeatFrame.setPayload(make([]byte, 1))
	if _, e := partner.realConn.Write(heartbeatFrame); e != nil {
		Log.Warnf("Failed to send heartbeat frame, Err(%v)", e)
		go partner.stop()
	}
}

func (partner *ahriClientPartner) receiverLoop() {
	partner.receiverLoopStop = false
	defer func() {
		partner.receiverLoopStop = true
	}()
	partner.realConnStop = false
	var buf, remain []byte
	defer func() {
		if buf != nil {
			ByteArrPool.Put(buf)
		}
		if remain != nil {
			ByteArrPool.Put(remain)
		}
	}()

	for {
		if partner.realConnStop {
			break
		}
		partner.realConnLock.RLock()
		realConn := partner.realConn;
		partner.realConnLock.RUnlock()

		if realConn == nil {
			break
		}
		buf = ByteArrPool.Get()
		n, e := realConn.Read(buf)
		if e != nil {
			ByteArrPool.Put(buf)
			Log.Warnf("Failed to read data from the real tcp connection, Err(%v).", e)
			go partner.clearConn()
			break
		}
		if n == 0 {
			ByteArrPool.Put(buf)
			continue
		}
		if (remain == nil && buf[0] != AfpFlag) || (remain != nil && remain[0] != AfpFlag) {
			Log.Warn("Encountered an expected error while parsing the real tcp conn data. There may be a network packet loss situation. The Receiver Loop was interrupted.")
			go partner.clearConn()
			break
		}
		if (remain == nil && buf[0] == AfpFlag) || (remain != nil && remain[0] == AfpFlag) {
			if remain != nil {
				remain = append(remain, buf[0:n]...)
				n = len(remain)
				remain, buf = buf, remain
				ByteArrPool.Put(remain)
				remain = nil
			}
			readIndex := 0
		SplitFrame:
			if n-readIndex <= 16 {
				remain = ByteArrPool.Get()
				copy(remain, buf[readIndex:n])
				remain = remain[0 : n-readIndex]
				ByteArrPool.Put(buf)
				continue
			}
			payloadLen := int(BytesToUint16(buf[readIndex+AfpHeaderLen-2 : readIndex+AfpHeaderLen]))
			payloadMaxLen := n - readIndex - AfpHeaderLen
			if payloadLen < payloadMaxLen {
				//one frame and a part (may another frame or other frames)
				splitIndex := readIndex + AfpHeaderLen + payloadLen
				partner.receiveFrameTruck <- NewAhriFrame(buf[readIndex:splitIndex])
				readIndex = splitIndex
				goto SplitFrame
			}
			if payloadLen == payloadMaxLen {
				partner.receiveFrameTruck <- NewAhriFrame(buf[readIndex:n])
				ByteArrPool.Put(buf)
				continue
			}
			if payloadLen > payloadMaxLen {
				remain = ByteArrPool.Get()
				remainBuf := buf[readIndex:n]
				copy(remain, remainBuf)
				remain = remain[0:len(remainBuf)]
				ByteArrPool.Put(buf)
				continue
			}
		}
		//The program should not have been executed to this line.
		Log.Error("Encountered an unexpected error while parsing the real tcp conn data. The Receiver Loop was interrupted.")
		go partner.clearConn()
		break
	}
}

func (partner *ahriClientPartner) dispatcher(frame AhriFrame) {
	needFree := true
	defer func() {
		if frame != nil && needFree {
			frame.free()
		}
		unknownError := recover()
		if unknownError != nil {
			if err, ok := unknownError.(error); ok {
				errMsg := err.Error()
				if IgnoreErrorSendOnClosedChannel == errMsg {
					return
				} else {
					Log.Errorf("Unknown Error: %s", errMsg)
				}
			} else {
				Log.Errorf("Unknown Error(%v): %v", reflect.TypeOf(unknownError), unknownError)
			}
		}
	}()
	if partner.quitFlag {
		return
	}
	if AfpFrameTypeHeartbeat == frame.frameType() {
		partner.heartbeatUnixTimeSec = time.Now().Unix()
		return
	}

	switch frame.frameType() {
	case AfpFrameTypeDial:
		if AhriAddrNameAhriServer == frame.to() {
			oPayload := frame.payload()
			payload := make([]byte, len(oPayload))
			copy(payload, oPayload)
			go partner.serverDialHandler(frame.from(), frame.connId(), payload)
			return
		} else {
			partner.proxyFrameTaskSender <- &proxyFrameTask{proxyFrame: frame, aesCipher: nil}
			needFree = false
			return
		}
	case AfpFrameTypeDialProxyAck:
		partner.proxyFrameTaskSender <- &proxyFrameTask{proxyFrame: frame, aesCipher: nil}
		needFree = false
		return
	case AfpFrameTypeDirect:
		if AhriAddrNameAhriServer == frame.to() {
			//find the AhriConn
			connId := make([]byte, 10)
			copy(connId[0:2], frame.from())
			copy(connId[2:], Uint64ToBytes(frame.connId()))
			value, _ := partner.conns.Load(string(connId))
			if value == nil {
				return
			}
			conn := value.(*AhriConn)
			select {
			case conn.receiver <- frame:
				needFree = false
			case <-time.After(AhriTimeoutSec * time.Second):
				go conn.Close()
			}
			return
		} else {
			partner.proxyFrameTaskSender <- &proxyFrameTask{proxyFrame: frame, aesCipher: partner.aesCipher}
			needFree = false
			return
		}
	case AfpFrameTypeProxy:
		partner.proxyFrameTaskSender <- &proxyFrameTask{proxyFrame: frame, aesCipher: partner.aesCipher}
		needFree = false
		return
	}
}

func (partner *ahriClientPartner) sender(frame AhriFrame) error {
	partner.realConnLock.RLock()
	realConn := partner.realConn
	partner.realConnLock.RUnlock()
	if realConn != nil {
		length := len(frame)
		for i := 0; i < length; {
			n, e := realConn.Write(frame[i:length])
			if e != nil {
				return e
			}
			i += n
		}
		return nil
	}
	return errors.New("the real tcp connection is unavailable")
}

func (partner *ahriClientPartner) logonVirtualConn(ahriConn *AhriConn) {
	partner.conns.Store(ahriConn.Id(), ahriConn)
}

func (partner *ahriClientPartner) logoffVirtualConn(ahriConn *AhriConn) {
	partner.conns.Delete(ahriConn.Id())
}

func (partner *ahriClientPartner) serverDialHandler(from string, connId uint64, payload []byte) {
	defer func() {
		e := recover()
		if e != nil {
			Log.Errorf("%v", e)
		}
	}()
	ackFrame := AhriFrame(make([]byte, AfpHeaderLen+1))
	ackFrame[0] = AfpFlag
	ackFrame.setFrameType(AfpFrameTypeDialAck)
	ackFrame.setFrom(AhriAddrNameAhriServer)
	ackFrame.setTo(from)
	ackFrame.setConnId(connId)

	var dstAddr net.IP
	switch payload[0] {
	case Socks5AddrTypeIPv4:
		dstAddr = payload[1 : 1+net.IPv4len]
	case Socks5AddrTypeIPv6:
		dstAddr = payload[1 : 1+net.IPv6len]
	case Socks5AddrTypeDomain:
		dstAddr = payload[2 : 2+payload[1]]
	default:
		Log.Warnf("Invalid ATYP: %v", payload[0])
		ackFrame = ackFrame.setPayload([]byte{AfpAckErr})
		_ = partner.sender(ackFrame)
		return
	}

	n := len(payload)
	port := int(BytesToUint16(payload[n-2 : n]))
	addr := &AhriAddr{name: AhriAddrNameAhriServer, addrType: int(payload[0]), dstAddr: dstAddr, port: port}
	if e := addr.ParseDstAddrIP(); e != nil {
		Log.Warnf("%v", e)
		ackFrame = ackFrame.setPayload([]byte{AfpAckErr})
		_ = partner.sender(ackFrame)
		return
	}

	c, e := net.DialTimeout("tcp", NetAddrString(net.IP(addr.dstAddr).String(), strconv.Itoa(addr.port)), time.Second)
	if e != nil {
		Log.Warnf("%v", e)
		ackFrame = ackFrame.setPayload([]byte{AfpAckErr})
		_ = partner.sender(ackFrame)
		return
	}
	serverConn := c.(*net.TCPConn)
	serverConn.SetLinger(0)
	defer serverConn.Close()

	ahriConn := NewAhriConnForVirtualization(
		AhriAddrNameAhriServer,
		from,
		connId,
		partner.aesCipher,
		AfpFrameTypeDirect,
		make(chan AhriFrame),
		partner.sender,
		partner.logoffVirtualConn)
	partner.logonVirtualConn(ahriConn)
	defer ahriConn.Close()

	ackFrame = ackFrame.setPayload([]byte{AfpAckOk})
	e = partner.sender(ackFrame)
	if e != nil {
		Log.Errorf("%v", e)
		return
	}

	connJoint(connId, serverConn, ahriConn)
}
