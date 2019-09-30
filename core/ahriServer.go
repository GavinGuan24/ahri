package core

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"io/ioutil"
	"net"
	"reflect"
	"sync"
	"time"
)

type AhriServer struct {
	password   string
	privateKey []byte
	publicKey  []byte

	quitFlag bool
	quit     chan int

	//key:string, value: *ahriClientPartner
	partners sync.Map

	proxyFrameTaskReceiver chan *proxyFrameTask
}

type proxyFrameTask struct {
	proxyFrame AhriFrame
	aesCipher  cipher.Block
}

func (server *AhriServer) loopStarter(listener net.Listener) {
	server.quitFlag = false
	server.quit = make(chan int)
	server.proxyFrameTaskReceiver = make(chan *proxyFrameTask, AhriServerProxyFrameMaxSize)
	ahriConnChannel := make(chan *net.TCPConn)
	go func() {
		for {
			if server.quitFlag {
				break
			}
			c, e := listener.Accept()
			if e != nil {
				Log.Error(e.Error())
				continue
			}
			conn := c.(*net.TCPConn)
			ahriConnChannel <- conn
		}
	}()
	go func() {
		for {
			if server.quitFlag {
				break
			}
			select {
			case pft := <-server.proxyFrameTaskReceiver:
				server.proxyFrameHandler(pft.proxyFrame, pft.aesCipher)
			}
		}
	}()
	go func() {
		for {
			select {
			case <-server.quit:
				goto loopEnd
			case conn := <-ahriConnChannel:
				go server.connectHandler(conn)
			}

		}
	loopEnd:
		server.quitFlag = true
		listener.Close()
		Log.Warn("Ahri Server Closed.")
	}()

}

func (server *AhriServer) connectHandler(conn *net.TCPConn) {
	e := server.ahriHandShakeAndTryRegisteClient(conn)
	if e != nil {
		conn.Close()
		Log.Warnf("Hand shake Err(%v)", e)
		return
	}
}

func (server *AhriServer) ahriHandShakeAndTryRegisteClient(conn *net.TCPConn) error {
	buf := ByteArrPool.Get()
	defer ByteArrPool.Put(buf)
	//send public key
	pubKeyLen := len(server.publicKey)
	copy(buf[0:pubKeyLen], server.publicKey)

	conn.SetDeadline(time.Now().Add(AhriShakeTimeoutSec * time.Second))
	_, e := conn.Write(buf[0:pubKeyLen])
	if e != nil {
		return e
	}
	//parse registe request
	n, e := conn.Read(buf)
	if e != nil {
		return e
	}
	if decrypted, e := DecryptRsa(buf[0:n], server.privateKey); e != nil {
		return e
	} else {
		n = len(decrypted)
		copy(buf[0:n], decrypted)
	}
	failedMsg := errors.New("ahri hand shake has been failed")
	if n < 7 {
		return failedMsg
	}
	index := 0

	passwordLen := int(buf[index])
	index += 1
	if n < index+passwordLen+1 {
		return failedMsg
	}
	password := string(buf[index : index+passwordLen])
	index += passwordLen

	nameLen := int(buf[index])
	index += 1
	if n < index+nameLen+1 {
		return failedMsg
	}
	clientName := string(buf[index : index+nameLen])
	index += nameLen

	mode := int(buf[index])
	index += 1

	aesKeyLen := int(buf[index])
	index += 1
	if n < index+aesKeyLen {
		return failedMsg
	}
	aesKey := make([]byte, aesKeyLen)
	copy(aesKey, buf[index:])

	if server.password != password {
		conn.Write([]byte{ArpAckWrongPassword})
		return errors.New("wrong password")
	}

	switch mode {
	case AhriClientModeTake, AhriClientModeGive, AhriClientModeTrade:
	default:
		conn.Write([]byte{ArpAckUnknownMode})
		return errors.New("unknown mode")
	}

	if len(aesKey) != 32 {
		conn.Write([]byte{ArpAckIllegalAesPassword})
		return errors.New("illegal AES password")
	}

	partner := server.logonClient(clientName, mode, aesKey, conn)
	if partner == nil {
		conn.Write([]byte{ArpAckClientNameIsAlreadyRegistered})
		return errors.New("client name is already registered")
	}

	conn.Write([]byte{ArpAckOk})
	conn.SetDeadline(NoDeadline)
	time.Sleep(2 * time.Millisecond)
	partner.loopStarter()

	return nil
}

func (server *AhriServer) logonClient(clientName string, mode int, aesKey []byte, conn *net.TCPConn) *ahriClientPartner {
	partner := &ahriClientPartner{}
	if _, loaded := server.partners.LoadOrStore(clientName, partner); loaded {
		return nil
	}
	partner.clientName = clientName
	partner.mode = mode
	partner.aesCipher, _ = aes.NewCipher(aesKey)
	partner.realConn = conn
	partner.proxyFrameTaskSender = server.proxyFrameTaskReceiver
	partner.willClose = server.logoffClient
	Log.NoLevelf("Got client(%s), mode: %d, aesKey: %v", clientName, mode, aesKey)
	return partner
}

func (server *AhriServer) logoffClient(clientName string) {
	server.partners.Delete(clientName)
}

func (server *AhriServer) checkoutClient(clientName string) *ahriClientPartner {
	value, _ := server.partners.Load(clientName)
	if value != nil {
		return value.(*ahriClientPartner)
	}
	return nil
}

func (server *AhriServer) proxyFrameHandler(proxyFrame AhriFrame, aesCipher cipher.Block) {
	var productFrame AhriFrame
	proxyFrameNeedFree, productFrameNeedFree := true, true
	defer func() {
		if proxyFrame != nil && proxyFrameNeedFree {
			proxyFrame.free()
		}
		if productFrame != nil && productFrameNeedFree {
			productFrame.free()
		}
		unknownError := recover()
		if unknownError != nil {
			if err, ok := unknownError.(error); ok {
				Log.Errorf("Unknown Error: %s", err.Error())
			} else {
				Log.Errorf("Unknown Error(%v): %v", reflect.TypeOf(unknownError), unknownError)
			}
		}
	}()

	clientPartner := server.checkoutClient(proxyFrame.to())
	if clientPartner == nil {
		return
	}

	switch proxyFrame.frameType() {
	case AfpFrameTypeDial, AfpFrameTypeDialProxyAck:
		if AfpFrameTypeDial == proxyFrame.frameType() && AhriClientModeTake == clientPartner.mode {
			return
		}
		if AfpFrameTypeDialProxyAck == proxyFrame.frameType() && AhriClientModeGive == clientPartner.mode {
			return
		}
		if AfpFrameTypeDial == proxyFrame.frameType() {
			proxyFrame.setFrameType(AfpFrameTypeDialProxy)
		}
		if AfpFrameTypeDialProxyAck == proxyFrame.frameType() {
			proxyFrame.setFrameType(AfpFrameTypeDialAck)
		}
		clientPartner.proxyFrameReceiver <- proxyFrame
		proxyFrameNeedFree = false
		return
	case AfpFrameTypeDirect, AfpFrameTypeProxy:
		if AfpFrameTypeDirect == proxyFrame.frameType() && AhriClientModeTake == clientPartner.mode {
			return
		}
		if AfpFrameTypeProxy == proxyFrame.frameType() && AhriClientModeGive == clientPartner.mode {
			return
		}
		productFrame = ByteArrPool.Get(len(proxyFrame))
		copy(productFrame[0:AfpHeaderLen], proxyFrame[0:AfpHeaderLen])
		if AfpFrameTypeDirect == proxyFrame.frameType() {
			productFrame.setFrameType(AfpFrameTypeProxy)
		}
		if AfpFrameTypeProxy == proxyFrame.frameType() {
			productFrame.setFrameType(AfpFrameTypeDirect)
		}
		encrypted := EncryptAesCfb256(DecryptAesCfb256(proxyFrame.payload(), aesCipher), clientPartner.aesCipher)
		defer ByteArrPool.Put(encrypted)
		productFrame = productFrame.setPayload(encrypted)

		clientPartner.proxyFrameReceiver <- productFrame
		productFrameNeedFree = false
		return
	}
}

func (server *AhriServer) Stop() {
	server.quit <- 0
}

func NewAhriServer(ip, port, password, rsaPriKeyPath, rsaPubKeyPath string) *AhriServer {
	switch {
	case ValidIp(ip) == false:
		Log.Crashf("Invalid IP(%s) for Ahri Server.", ip)
	case ValidPort(port) == false:
		Log.Crashf("Invalid Port(%s) for Ahri Server.", port)
	case len(password) == 0 || len(password) > 12:
		Log.Crashf("Invalid Password(len == 0 or len > 12).")
	}

	pubKey, e := ioutil.ReadFile(rsaPubKeyPath)
	if e != nil {
		Log.Crashf("Invalid RSA public key file(%s) Err(%v).", rsaPubKeyPath, e)
	}
	priKey, e := ioutil.ReadFile(rsaPriKeyPath)
	if e != nil {
		Log.Crashf("Invalid RSA private key file(%s) Err(%v).", rsaPriKeyPath, e)
	}

	ciphertext, e := EncryptRsa([]byte(password), pubKey)
	if e != nil {
		Log.Crashf("Invalid RSA public key file(%s).", rsaPubKeyPath)
	}
	decrypted, e := DecryptRsa(ciphertext, priKey)
	if e != nil {
		Log.Crashf("Invalid RSA private key file(%s).", rsaPriKeyPath)
	}
	if string(decrypted) != password {
		Log.Crashf("Invalid RSA key files(%s, %s)", rsaPubKeyPath, rsaPriKeyPath)
	}

	server := &AhriServer{}
	if listener, e := net.Listen("tcp", NetAddrString(ip, port)); e != nil {
		Log.Crash(e.Error())
	} else {
		server.password = password
		server.privateKey = priKey
		server.publicKey = pubKey
		server.loopStarter(listener)
	}
	return server
}
