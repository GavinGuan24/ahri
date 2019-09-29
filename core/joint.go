package core

import (
	"net"
	"reflect"
	"time"
)

type buffer struct {
	buf []byte
	n   int
}

func (b *buffer) String() string {
	return string(b.bytes(0))
}

func (b *buffer) bytes(from int) []byte {
	return b.buf[from:b.n]
}

func (b *buffer) free() {
	if b.buf != nil {
		ByteArrPool.Put(b.buf)
		b.buf = nil
	}
	b.n = 0
}

func readDataFromConn(connId uint64, conn net.Conn) chan *buffer {
	channel := make(chan *buffer)
	go func() {
		defer func() {
			unknownError := recover()
			if unknownError != nil {
				if err, ok := unknownError.(error); ok {
					errMsg := err.Error()
					switch errMsg {
					case IgnoreErrorSendOnClosedChannel, IgnoreErrorInvalidMemoryAddress:
						return
					}
					Log.Errorf("Unknown Error: %s", errMsg)
				} else {
					Log.Errorf("Unknown Error(%v): %v", reflect.TypeOf(unknownError), unknownError)
				}
			}
		}()
		var e error
		var b *buffer
		for {
			b = &buffer{buf: ByteArrPool.Get(), n: 0}
			b.n, e = conn.Read(b.buf)
			if e != nil {
				Log.Debugf("Connect(%d) data read-err: %v", connId, e)
				b.free()
				return
			}
			if b.n == 0 {
				b.free()
				return
			}

			Log.Debugf("Connect(%d) data: %v", connId, b)
			channel <- b
		}
	}()
	return channel
}

func writeDataToConn(conn net.Conn, b *buffer) error {
	defer b.free()
	for i := 0; i < b.n; {
		n, e := conn.Write(b.bytes(i))
		if e != nil {
			return e
		}
		i += n
	}
	return nil
}

func connJoint(connId uint64, tcpConn *net.TCPConn, ahriConn *AhriConn) {
	defer func() {
		unknownError := recover()
		if unknownError != nil {
			if err, ok := unknownError.(error); ok {
				errMsg := err.Error()
				switch errMsg {
				case IgnoreErrorSendOnClosedChannel, IgnoreErrorInvalidMemoryAddress:
					return
				}
				Log.Errorf("Unknown Error: %s", errMsg)
			} else {
				Log.Errorf("Unknown Error(%v): %v", reflect.TypeOf(unknownError), unknownError)
			}
		}
	}()

	tcpConn.SetNoDelay(true)
	dataChan0 := readDataFromConn(connId, tcpConn)
	dataChan1 := readDataFromConn(connId, ahriConn)

	for {
		var e error
		select {
		case <-time.After(AhriTimeoutSec * time.Second):
			goto loopEnd
		case data0 := <-dataChan0:
			e = writeDataToConn(ahriConn, data0)
		case data1 := <-dataChan1:
			e = writeDataToConn(tcpConn, data1)
		}
		if e != nil {
			Log.Warnf("Connect(%d) data write-err: %v", connId, e)
			goto loopEnd
		}
	}

loopEnd:
	tcpConn.Close()
	ahriConn.Close()
	close(dataChan0)
	close(dataChan1)
	Log.Debugf("Connect(%d) Closed.", connId)
}
