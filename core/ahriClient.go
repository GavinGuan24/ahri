package core

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"net"
	"reflect"
	"strconv"
	"sync"
	"time"
)

type AhriClient struct {
	name      string
	mode      int
	aesKey    [32]byte
	aesCipher cipher.Block

	serverAddr     string
	serverPassword string

	quitOnec sync.Once
	quitFlag bool
	quit     chan int

	heartbeatUnixTimeSec int64
	realConnStop         bool
	realConnLock         sync.RWMutex
	realConnEpoch        uint64
	realConn             *net.TCPConn

	receiverLoopStop  bool
	receiveFrameTruck chan AhriFrame
	//key:string, value: *AhriConn
	conns sync.Map

	mapperAhriAddrName func(atyp byte, byteAddr []byte) string
}

func (client *AhriClient) loopStarter() {
	client.receiverLoopStop = true
	client.receiveFrameTruck = make(chan AhriFrame)
	client.aesKey = GenerateAes256Key()
	client.aesCipher, _ = aes.NewCipher(client.aesKey[:])
	client.quit = make(chan int)
	go func() {
		for {
			select {
			case frame := <-client.receiveFrameTruck:
				client.dispatcher(frame)
			case <-time.After(100*time.Millisecond):
				if client.quitFlag && client.receiverLoopStop {
					goto loopEnd
				}
			}
		}
	loopEnd:
		Log.Warn("Ahri Client receiveFrameTruck loop stopped.")
	}()
	go func() {
		for {
			select {
			case <-client.quit:
				goto loopEnd
			case <-time.After(time.Second):
				go client.keepConn()
			}
		}
	loopEnd:
		client.clearConn(0)
		Log.Warn("Ahri Client Closed.")
	}()
}

func (client *AhriClient) clearConn(epoch uint64) {
	client.realConnLock.Lock()
	defer client.realConnLock.Unlock()
	if client.realConnEpoch == epoch || epoch == 0 {
		client.realConnStop = true
		if client.realConn != nil {
			client.realConn.Close()
			client.realConn = nil
		}
		client.conns.Range(func(key, value interface{}) bool {
			defer func() {
				recover()
			}()
			conn := value.(*AhriConn)
			conn.Close()
			return true
		})
	}
}

func (client *AhriClient) keepConn() {
	client.realConnLock.Lock()
	defer client.realConnLock.Unlock()
	if client.realConn == nil {
		c, e := net.DialTimeout("tcp", client.serverAddr, time.Second)
		if e != nil {
			Log.Warnf("ARP Err (%v)", e)
			return
		}
		conn := c.(*net.TCPConn)
		e = client.registe(conn)
		if e != nil {
			Log.Warnf("ARP Err(%v)", e)
			return
		}
		Log.NoLevel("Registered to the server successfully.")
		client.realConn = conn
		client.heartbeatUnixTimeSec = time.Now().Unix()
		client.realConnEpoch++
		go client.receiverLoop()
	} else {
		//Check if the heartbeat of the server has timed out
		if time.Now().Unix()-client.heartbeatUnixTimeSec > 2 * int64(AhriTimeoutSec) {
			go client.clearConn(client.realConnEpoch)
			return
		}
		//send heartbeat
		heartbeatFrame := AhriFrame(make([]byte, 17))
		heartbeatFrame[0] = AfpFlag
		heartbeatFrame.setFrameType(AfpFrameTypeHeartbeat)
		heartbeatFrame.setFrom(client.name)
		heartbeatFrame.setTo(AhriAddrNameAhriServer)
		heartbeatFrame.setConnId(0)
		heartbeatFrame = heartbeatFrame.setPayload(make([]byte, 1))
		if _, e := client.realConn.Write(heartbeatFrame); e != nil {
			Log.Warnf("Failed to send heartbeat frame, Err(%v)", e)
			go client.clearConn(client.realConnEpoch)
		}
	}
}

func (client *AhriClient) registe(conn *net.TCPConn) error {
	buf := ByteArrPool.Get()
	defer ByteArrPool.Put(buf)

	conn.SetDeadline(time.Now().Add(AhriShakeTimeoutSec * time.Second))
	n, e := conn.Read(buf)
	if e != nil {
		return e
	}

	publicKey := make([]byte, n)
	copy(publicKey, buf[0:n])
	//prepare parameters for the registe request
	index := 0
	myAppend := func(item []byte) {
		itemLen := len(item)
		buf[index] = byte(itemLen)
		copy(buf[index+1:index+1+itemLen], item)
		index += 1 + itemLen
	}
	myAppend([]byte(client.serverPassword))
	myAppend([]byte(client.name))
	buf[index] = byte(client.mode)
	index += 1
	myAppend(client.aesKey[:])

	if ciphertext, e := EncryptRsa(buf[0:index], publicKey); e != nil {
		return errors.New("invalid RSA public key")
	} else {
		_, e = conn.Write(ciphertext)
		if e != nil {
			return e
		}
	}

	//parse registe ack
	n, e = conn.Read(buf)
	if e != nil {
		return e
	}
	if n == 1 {
		switch buf[0] {
		case ArpAckOk:
			conn.SetDeadline(NoDeadline)
			return nil
		case ArpAckWrongPassword:
			return errors.New("wrong server password")
		case ArpAckClientNameIsAlreadyRegistered:
			return errors.New("client name is already registered")
		case ArpAckUnknownMode:
			return errors.New("unknown mode")
		case ArpAckIllegalAesPassword:
			return errors.New("illegal AES password")
		}
	}
	return errors.New("unknown error")
}

func (client *AhriClient) receiverLoop() {
	client.receiverLoopStop = false
	defer func() {
		client.receiverLoopStop = true
	}()
	client.realConnStop = false
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
		if client.realConnStop {
			break
		}
		client.realConnLock.RLock()
		realConn := client.realConn;
		epoch := client.realConnEpoch
		client.realConnLock.RUnlock()

		if realConn == nil {
			continue
		}
		buf = ByteArrPool.Get()
		n, e := realConn.Read(buf)
		if e != nil {
			ByteArrPool.Put(buf)
			Log.Warnf("Failed to read data from the real tcp connection, Err(%v).", e)
			go client.clearConn(epoch)
			break
		}
		if n == 0 {
			ByteArrPool.Put(buf)
			continue
		}

		if (remain == nil && buf[0] != AfpFlag) || (remain != nil && remain[0] != AfpFlag) {
			Log.Warn("Encountered an expected error while parsing the real tcp conn data. There may be a network packet loss situation. The Receiver Loop was interrupted.")
			go client.clearConn(epoch)
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
				// one frame and a part (may another frame or other frames)
				splitIndex := readIndex + AfpHeaderLen + payloadLen
				client.receiveFrameTruck <- NewAhriFrame(buf[readIndex:splitIndex])
				readIndex = splitIndex
				goto SplitFrame
			}
			if payloadLen == payloadMaxLen {
				client.receiveFrameTruck <- NewAhriFrame(buf[readIndex:n])
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
		// The program should not have been executed to this line.
		Log.Error("Encountered an unexpected error while parsing the real tcp conn data. The Receiver Loop was interrupted.")
		go client.clearConn(epoch)
		break
	}
}

func (client *AhriClient) dispatcher(frame AhriFrame) {
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
	if client.quitFlag {
		return
	}
	if AfpFrameTypeHeartbeat == frame.frameType() {
		client.heartbeatUnixTimeSec = time.Now().Unix()
		return
	}

	switch frame.frameType() {
	case AfpFrameTypeDialAck, AfpFrameTypeDirect:
		//find the AhriConn
		connId := make([]byte, 10)
		copy(connId[0:2], frame.from())
		copy(connId[2:], Uint64ToBytes(frame.connId()))
		value, _ := client.conns.Load(string(connId))
		if value == nil {
			return
		}
		conn := value.(*AhriConn)
		select {
		case conn.receiver <- frame:
			needFree = false
		case <-time.After(time.Duration(AhriTimeoutSec) * time.Second):
			go conn.Close()
		}
		return
	case AfpFrameTypeDialProxy:
		oPayload := frame.payload()
		payload := make([]byte, len(oPayload))
		copy(payload, oPayload)
		if AhriClientModeTake == client.mode {
			return
		}
		go client.proxyDialHandler(frame.from(), frame.connId(), payload)
	case AfpFrameTypeProxy:
		if client.name == frame.to() {
			//find the AhriConn
			connId := make([]byte, 10)
			copy(connId[0:2], frame.from())
			copy(connId[2:], Uint64ToBytes(frame.connId()))
			value, _ := client.conns.Load(string(connId))
			if value == nil {
				return
			}
			conn := value.(*AhriConn)
			select {
			case conn.receiver <- frame:
				needFree = false
			case <-time.After(time.Duration(AhriTimeoutSec) * time.Second):
				go conn.Close()
			}
			return
		}
	}
}

func (client *AhriClient) sender(frame AhriFrame) error {
	client.realConnLock.RLock()
	realConn := client.realConn
	client.realConnLock.RUnlock()
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

func (client *AhriClient) logonVirtualConn(ahriConn *AhriConn) {
	client.conns.Store(ahriConn.Id(), ahriConn)
}

func (client *AhriClient) logoffVirtualConn(ahriConn *AhriConn) {
	client.conns.Delete(ahriConn.Id())
}

func (client *AhriClient) proxyDialHandler(from string, connId uint64, payload []byte) {
	defer func() {
		e := recover()
		if e != nil {
			Log.Errorf("%v", e)
		}
	}()
	ackFrame := AhriFrame(make([]byte, AfpHeaderLen+1))
	ackFrame[0] = AfpFlag
	ackFrame.setFrameType(AfpFrameTypeDialProxyAck)
	ackFrame.setFrom(client.name)
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
		_ = client.sender(ackFrame)
		return
	}

	ahriAddrName := client.mapperAhriAddrName(payload[0], dstAddr)
	if AhriAddrNameForbidden == ahriAddrName {
		Log.Warn("Request is forbidden.")
		ackFrame = ackFrame.setPayload([]byte{AfpAckErr})
		_ = client.sender(ackFrame)
		return
	}

	n := len(payload)
	port := int(BytesToUint16(payload[n-2 : n]))
	addr := &AhriAddr{name: client.name, addrType: int(payload[0]), dstAddr: dstAddr, port: port}
	if e := addr.ParseDstAddrIP(); e != nil {
		Log.Warnf("%v", e)
		ackFrame = ackFrame.setPayload([]byte{AfpAckErr})
		_ = client.sender(ackFrame)
		return
	}

	c, e := net.DialTimeout("tcp", NetAddrString(net.IP(addr.dstAddr).String(), strconv.Itoa(addr.port)), time.Second)
	if e != nil {
		Log.Warnf("%v", e)
		ackFrame = ackFrame.setPayload([]byte{AfpAckErr})
		_ = client.sender(ackFrame)
		return
	}
	proxyConn := c.(*net.TCPConn)
	proxyConn.SetLinger(0)
	defer proxyConn.Close()

	ahriConn := NewAhriConnForVirtualization(
		client.name,
		from,
		connId,
		client.aesCipher,
		AfpFrameTypeProxy,
		make(chan AhriFrame),
		client.sender,
		client.logoffVirtualConn)
	client.logonVirtualConn(ahriConn)
	defer ahriConn.Close()

	ackFrame = ackFrame.setPayload([]byte{AfpAckOk})
	e = client.sender(ackFrame)
	if e != nil {
		Log.Errorf("%v", e)
		return
	}

	connJoint(connId, proxyConn, ahriConn)
}

func (client *AhriClient) Dial(addr *AhriAddr, connId uint64) (conn *AhriConn, e error) {
	defer func() {
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
	//mode: local
	if AhriAddrNameLocal == addr.name {
		e = addr.ParseDstAddrIP()
		if e != nil {
			return nil, e
		}
		c, e := net.DialTimeout("tcp", NetAddrString(net.IP(addr.dstAddr).String(), strconv.Itoa(addr.port)), time.Second)
		if e != nil {
			return nil, e
		}
		localConn := c.(*net.TCPConn)
		return &AhriConn{sourceAddr: client.name, targetAddr: AhriAddrNameLocal, id: connId, conn: localConn}, nil
	}

	//mode: A -> S or A -> S -> B
	connReceiver := make(chan AhriFrame)

	ahriConn := NewAhriConnForVirtualization(
		client.name,
		addr.name,
		connId,
		client.aesCipher,
		AfpFrameTypeDirect,
		connReceiver,
		client.sender,
		client.logoffVirtualConn)

	//send frame(dial)
	client.logonVirtualConn(ahriConn)
	//-- 1. make a dial frame payload
	payload := make([]byte, 0, 16)
	payload = append(payload, byte(addr.addrType))
	if Socks5AddrTypeDomain == addr.addrType {
		payload = append(payload, byte(len(addr.dstAddr)))
	}
	payload = append(payload, addr.dstAddr...)
	payload = append(payload, Uint16ToBytes(uint16(addr.port))...)
	//-- 2. make a dial frame
	dialFrame := AhriFrame(make([]byte, AfpHeaderLen, 32))
	dialFrame[0] = AfpFlag
	dialFrame.setFrameType(AfpFrameTypeDial)
	dialFrame.setFrom(client.name)
	dialFrame.setTo(addr.name)
	dialFrame.setConnId(connId)
	dialFrame.setPayloadLen(uint16(len(payload)))
	dialFrame = append(dialFrame, payload...)
	e = client.sender(dialFrame)
	if e != nil {
		ahriConn.Close()
		return nil, errors.New("failed to dial target, network is unavailable")
	}

	//receive frame(dial ack) with timeout
	select {
	case dialAck := <-connReceiver:
		defer dialAck.free()
		//when the real tcp connection is unavailable, 'dialAck' may have been recycled.
		//Several runtime errors will be encountered in the entire program.
		//runtime error: slice bounds out of range
		//runtime error: invalid memory address or nil pointer dereference
		if len(dialAck) == 0 {
			//len(dialAck) == 0, cap(dialAck) == 0
			return nil, errors.New("failed to dial target, network is unavailable")
		}
		if AfpAckOk == dialAck.payload()[0] {
			return ahriConn, nil
		} else {
			ahriConn.Close()
			return nil, errors.New("failed to dial target, connection denied")
		}
	case <-time.After(3 * time.Duration(AhriTimeoutSec) * time.Second):
		ahriConn.Close()
		return nil, errors.New("failed to dial target, connection timeout")
	}
}

func (client *AhriClient) Stop() {
	client.quitOnec.Do(func() {
		client.quit <- 0
		client.quitFlag = true
	})
}

func NewAhriClient(serverIp, serverPort, serverPassword, clientName string, mode int) *AhriClient {

	switch {
	case ValidIp(serverIp) == false:
		Log.Crashf("Invalid Ahri Server IP(%s)", serverIp)
	case ValidPort(serverPort) == false:
		Log.Crashf("Invalid Ahri Server port(%s)", serverPort)
	case len(serverPassword) == 0:
		Log.Crash("Invalid Ahri Server password(len == 0).")
	case AhriAddrNameAhriServer == clientName, AhriAddrNameLocal == clientName, AhriAddrNameForbidden == clientName, AhriAddrNameIntercept == clientName:
		Log.Crash("'|', '-', 'S' and 'L' are system reserved names and cannot be used.")
	case len(clientName) == 0, len(clientName) > 2:
		Log.Crash("The client name should be between 1 and 2 in length.")
	}
	switch mode {
	case AhriClientModeTake, AhriClientModeGive, AhriClientModeTrade:
	default:
		Log.Crashf("Invalid Ahri Client Mode(%d)", mode)
	}

	client := &AhriClient{}
	client.serverAddr = NetAddrString(serverIp, serverPort)
	client.serverPassword = serverPassword
	client.name = clientName
	client.mode = mode
	client.loopStarter()
	return client
}
