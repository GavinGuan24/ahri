package core

import (
	"crypto/aes"
	"crypto/cipher"
	"errors"
	"net"
	"sync"
	"time"
)

const (
	writeMaxLen = AfpPayloadMaxLen - aes.BlockSize
)

//当 targetAddr == AhriAddrNameLocal, AhriConn 的行为与 *net.TCPConn 基本一致
//当 targetAddr == AhriAddrNameAhriServer 或者目标名, AhriConn 就是一个虚拟连接
type AhriConn struct {
	//协议中约束该值为长度为 1 ~ 2 的文本
	sourceAddr string
	targetAddr string
	id         uint64

	//targetAddr == AhriAddrNameLocal, 该属性有效
	conn *net.TCPConn

	//targetAddr == AhriAddrNameAhriServer 或者目标名, 下列属性有效
	aesCipher    cipher.Block
	alive        bool
	afpFrameType int
	receiver     chan AhriFrame
	sender       func(AhriFrame) error
	willClose    func(*AhriConn)
	closeOnce    sync.Once
}

//仅作为标识, 其内容并不是人类可读的内容
func (conn *AhriConn) Id() string {
	id := make([]byte, 10)
	copy(id[0:2], conn.targetAddr)
	copy(id[2:], Uint64ToBytes(conn.id))
	return string(id)
}

func (conn *AhriConn) Read(b []byte) (n int, err error) {
	if AhriAddrNameLocal == conn.targetAddr {
		return conn.conn.Read(b)
	}
	if !conn.alive {
		return 0, errors.New("cannot read a closed AhriConn")
	}
	frame, ok := <-conn.receiver
	if !ok {
		return 0, errors.New("cannot read a closed AhriConn")
	}
	defer frame.free()
	decrypted := DecryptAesCfb256(frame.payload(), conn.aesCipher)
	copy(b, decrypted)
	return int(frame.payloadLen() - aes.BlockSize), nil
}

func (conn *AhriConn) Write(b []byte) (n int, err error) {
	if AhriAddrNameLocal == conn.targetAddr {
		return conn.conn.Write(b)
	}
	if !conn.alive {
		return 0, errors.New("cannot write a closed AhriConn")
	}
	frame := AhriFrame(ByteArrPool.Get())
	defer frame.free()
	frame[0] = AfpFlag
	frame.setFrameType(byte(conn.afpFrameType))
	frame.setFrom(conn.sourceAddr)
	frame.setTo(conn.targetAddr)
	frame.setConnId(conn.id)
	//set payload
	length := len(b)
	if length > writeMaxLen {
		n = writeMaxLen
	} else {
		n = length
	}
	encrypted := EncryptAesCfb256(b[0:n], conn.aesCipher)
	defer ByteArrPool.Put(encrypted)
	frame = frame.setPayload(encrypted)
	e := conn.sender(frame)
	if e != nil {
		return 0, e
	} else {
		return n, nil
	}
}

func (conn *AhriConn) Close() error {
	if AhriAddrNameLocal == conn.targetAddr {
		return conn.conn.Close()
	}
	conn.closeOnce.Do(func() {
		if conn.alive {
			conn.willClose(conn)
			conn.alive = false
			close(conn.receiver)
		}
	})
	return nil
}

func NewAhriConnForVirtualization(
	from string,
	to string,
	connId uint64,
	aesCipher cipher.Block,
	afpFrameType int,
	connReceiver chan AhriFrame,
	sender func(AhriFrame) error,
	willClose func(*AhriConn)) *AhriConn {

	return &AhriConn{
		sourceAddr:   from,
		targetAddr:   to,
		id:           connId,
		aesCipher:    aesCipher,
		alive:        true,
		afpFrameType: afpFrameType,
		receiver:     connReceiver,
		sender:       sender,
		willClose:    willClose}
}

//---------------------------------------------

func (conn *AhriConn) LocalAddr() net.Addr {
	if AhriAddrNameLocal == conn.targetAddr {
		return conn.conn.LocalAddr()
	}
	panic("Don't invoke me")
}

func (conn *AhriConn) RemoteAddr() net.Addr {
	if AhriAddrNameLocal == conn.targetAddr {
		return conn.conn.RemoteAddr()
	}
	panic("Don't invoke me")
}

func (conn *AhriConn) SetDeadline(t time.Time) error {
	if AhriAddrNameLocal == conn.targetAddr {
		return conn.conn.SetDeadline(t)
	}
	panic("Don't invoke me")
}

func (conn *AhriConn) SetReadDeadline(t time.Time) error {
	if AhriAddrNameLocal == conn.targetAddr {
		return conn.conn.SetReadDeadline(t)
	}
	panic("Don't invoke me")
}

func (conn *AhriConn) SetWriteDeadline(t time.Time) error {
	if AhriAddrNameLocal == conn.targetAddr {
		return conn.conn.SetWriteDeadline(t)
	}
	panic("Don't invoke me")
}

func (conn *AhriConn) SetLinger(sec int) error {
	if AhriAddrNameLocal == conn.targetAddr {
		return conn.conn.SetLinger(sec)
	}
	panic("Don't invoke me")
}
