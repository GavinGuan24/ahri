package core

import (
	"io/ioutil"
	"net"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"
)

type AhriSocks5Server struct {
	ahriClient *AhriClient

	quitFlag bool
	quit     chan int

	connIdGen uint64

	addrCtxMapper        map[string]string
	addrCtxMapperModTime time.Time
	addrCtxMapperLock    sync.Mutex
}

func (ahriSocks5Server *AhriSocks5Server) loopStarter(listener net.Listener, ahriHostsPath string) {
	ahriSocks5Server.quitFlag = false
	ahriSocks5Server.quit = make(chan int)
	ahriSocks5ConnChannel := make(chan *net.TCPConn)
	if listener != nil {
		go func() {
			for {
				if ahriSocks5Server.quitFlag {
					break
				}
				c, e := listener.Accept()
				if e != nil {
					Log.Error(e.Error())
					continue
				}
				conn := c.(*net.TCPConn)
				ahriSocks5ConnChannel <- conn
			}
		}()
	}

	go func() {
		for {
			select {
			case <-ahriSocks5Server.quit:
				ahriSocks5Server.quitFlag = true
				if listener != nil {
					listener.Close()
				}
				goto loopEnd
			case conn := <-ahriSocks5ConnChannel:
				ahriSocks5Server.connIdGen++
				go ahriSocks5Server.ahriSocks5ConnectHandler(conn, ahriSocks5Server.connIdGen)
			case <-time.After(time.Second):
				go func() {
					fileInfo, e := os.Stat(ahriHostsPath)
					if e != nil {
						ahriSocks5Server.resetAddrCtxMapper()
						return
					}
					if fileInfo.ModTime().After(ahriSocks5Server.addrCtxMapperModTime) || ahriSocks5Server.addrCtxMapperModTime.IsZero() {
						ahriSocks5Server.loadAddrCtxMapper(ahriHostsPath, fileInfo.ModTime())
					}
				}()
			}
		}
	loopEnd:
		Log.Warn("Ahri Socks5 Server Closed.")
	}()
}

func (ahriSocks5Server *AhriSocks5Server) resetAddrCtxMapper() {
	ahriSocks5Server.addrCtxMapperLock.Lock()
	defer ahriSocks5Server.addrCtxMapperLock.Unlock()
	ahriSocks5Server.addrCtxMapperModTime = time.Time{}
	size := len(ahriSocks5Server.addrCtxMapper)
	if size == 0 {
		return
	}
	ahriSocks5Server.addrCtxMapper = make(map[string]string)
}

func (ahriSocks5Server *AhriSocks5Server) loadAddrCtxMapper(ahriHostsPath string, modTime time.Time) {
	ahriSocks5Server.addrCtxMapperLock.Lock()
	defer ahriSocks5Server.addrCtxMapperLock.Unlock()
	bytes, e := ioutil.ReadFile(ahriHostsPath)
	if e != nil {
		ahriSocks5Server.addrCtxMapper = make(map[string]string)
		ahriSocks5Server.addrCtxMapperModTime = time.Time{}
		return
	}
	ahriSocks5Server.addrCtxMapper = ParseAddrCtxMapper(strings.Split(string(bytes), "\n"), ahriHostsPath)
	ahriSocks5Server.addrCtxMapperModTime = modTime
}

func (ahriSocks5Server *AhriSocks5Server) mapperAhriAddrName(atyp byte, byteAddr []byte) string {
	ahriSocks5Server.addrCtxMapperLock.Lock()
	defer ahriSocks5Server.addrCtxMapperLock.Unlock()
	return MapperAhriAddrName(atyp, byteAddr, ahriSocks5Server.addrCtxMapper)
}

func (ahriSocks5Server *AhriSocks5Server) socks5HandShakeAndMapper(conn *net.TCPConn, connId uint64) (*AhriAddr) {
	buf := ByteArrPool.Get()
	defer ByteArrPool.Put(buf)

	// ========== Socks5 auth ==========
	conn.SetDeadline(time.Now().Add(Socks5ShakeTimeoutMillisec * time.Millisecond))
	n, e := conn.Read(buf)
	if e != nil {
		Log.Infof("%v", e)
		return nil
	}

	protocolCheck := false
	if n >= 3 && buf[0] == 0x05 && buf[1] > 0 {
		for i := 0; i < int(buf[1]); i++ {
			if buf[2+i] == 0x00 {
				protocolCheck = true
			}
		}
	}
	if !protocolCheck {
		conn.Write(Socks5NoAcceptableMethods)
		return nil
	}
	conn.Write(Socks5SecretFree)

	// ========== Socks5 request auth ==========
	n, e = conn.Read(buf)
	if e != nil {
		Log.Infof("%v", e)
		return nil
	}
	//shit request
	if !(n >= 10 && buf[0] == 0x05 && buf[2] == 0x00) {
		conn.Write(Socks5ConnectionNotAllowedByRuleset)
		return nil
	}
	//CMD
	if buf[1] != 0x01 {
		Log.Debugf("Unsupported Command (value: %x).", buf[1])
		conn.Write(Socks5UnsupportedCommand)
		return nil
	}
	//ATYP & DST.ADDR
	var dstIP net.IP
	port := 0
	switch buf[3] {
	case Socks5AddrTypeIPv4:
		dstIP = buf[4 : 4+net.IPv4len]
	case Socks5AddrTypeDomain:
		dstIP = buf[5 : 5+buf[4]]
	case Socks5AddrTypeIPv6:
		dstIP = buf[4 : 4+net.IPv6len]
	default:
		Log.Debugf("Unsupported ATYP (value: %x) of Socks5 Protocol.", buf[3])
		conn.Write(Socks5UnsupportedAddressType)
		return nil
	}

	ahriAddrName := ahriSocks5Server.mapperAhriAddrName(buf[3], dstIP)
	if AhriAddrNameForbidden == ahriAddrName {
		ahriAddrName = AhriAddrNameLocal
	}
	if AhriAddrNameIntercept == ahriAddrName {
		conn.Write(Socks5ConnectionNotAllowedByRuleset)
		return nil
	}

	// ========== Socks5 hand shake finished ==========
	conn.Write(Socks5Success)
	conn.SetDeadline(NoDeadline)

	port = int(BytesToUint16(buf[n-2 : n]))

	dstAddr := make([]byte, len(dstIP))
	copy(dstAddr, dstIP)

	ahriAddr := &AhriAddr{name: ahriAddrName, addrType: int(buf[3]), dstAddr: dstAddr, port: port}
	Log.NoLevelf("Connect(%d) Target %v.", connId, ahriAddr)
	return ahriAddr
}

func (ahriSocks5Server *AhriSocks5Server) ahriSocks5ConnectHandler(conn *net.TCPConn, connId uint64) {
	defer func() {
		unknownError := recover()
		if unknownError != nil {
			if err, ok := unknownError.(error); ok {
				errMsg := err.Error()
				switch errMsg {
				case
					IgnoreErrorSendOnClosedChannel,
					IgnoreErrorInvalidMemoryAddress,
					IgnoreErrorSliceBoundsOutOfRange:
					return
				}
				Log.Errorf("Unknown Error: %s", errMsg)
			} else if errStr, ok := unknownError.(string); ok {
				switch errStr {
				case IgnoreErrorInvalidBufferOverlap:
					return
				}
				Log.Errorf("Unknown Error: %s", errStr)
			} else {
				Log.Errorf("Unknown Error(%v): %v", reflect.TypeOf(unknownError), unknownError)
			}
		}
	}()
	conn.SetLinger(0)
	var ahriAddr *AhriAddr
	if ahriAddr = ahriSocks5Server.socks5HandShakeAndMapper(conn, connId); ahriAddr == nil {
		return
	}

	//Connect to Ahri client.
	ahriConn, e := ahriSocks5Server.ahriClient.Dial(ahriAddr, connId)
	if e != nil {
		Log.Warnf("Connect(%d) dial target(%v) err: %v", connId, ahriAddr, e)
		return
	}
	defer ahriConn.Close()
	if AhriAddrNameLocal == ahriAddr.name {
		ahriConn.SetLinger(0)
	}

	connJoint(connId, conn, ahriConn)
}

func (ahriSocks5Server *AhriSocks5Server) Stop() {
	ahriSocks5Server.quit <- 0
}

func NewAhriSocks5Server(socks5IP, socks5Port, ahriHostsPath string, ahriClient *AhriClient) *AhriSocks5Server {
	if ahriClient.mode != AhriClientModeGive {
		switch {
		case ValidIp(socks5IP) == false:
			Log.Crashf("Invalid IP(%s) for Ahri Socks5 Server.", socks5IP)
		case ValidPort(socks5Port) == false:
			Log.Crashf("Invalid Port(%s) for Ahri Socks5 Server.", socks5Port)
		}
	}

	if ahriHostsFileInfo, e := os.Stat(ahriHostsPath); e != nil {
		Log.Crashf("Invalid Ahri Hosts File(%s), Err(%v).", ahriHostsPath, e)
	} else {
		if ahriHostsFileInfo.IsDir() {
			Log.Crashf("Your Ahri Hosts File(%s) is a Folder.", ahriHostsPath)
		}
	}

	ahriSocks5Server := &AhriSocks5Server{}
	var listener net.Listener
	var e error
	if ahriClient.mode != AhriClientModeGive {
		listener, e = net.Listen("tcp", NetAddrString(socks5IP, socks5Port))
		if e != nil {
			Log.Crash(e.Error())
		}
	}
	ahriSocks5Server.ahriClient = ahriClient
	ahriClient.mapperAhriAddrName = ahriSocks5Server.mapperAhriAddrName
	ahriSocks5Server.loopStarter(listener, ahriHostsPath)
	return ahriSocks5Server
}
