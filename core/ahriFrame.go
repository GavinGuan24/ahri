package core

import (
	"bytes"
)

type AhriFrame []byte

/*
    +---------------+------------+------+----+---------+-------------+----------------+
	| protocol flag | frame type | from | to | conn No | payload len |    payload     |
    +---------------+------------+------+----+---------+-------------+----------------+
	|        1      |     1      |  2   | 2  |    8    |      2      | variable<=2032 |
    +---------------+------------+------+----+---------+-------------+----------------+
	|        0      |     1      |  2   | 4  |    6    |     14      |       16       |
	+---------------+------------+------+----+---------+-------------+----------------+
*/

func (frame AhriFrame) frameType() byte {
	return frame[1]
}

func (frame AhriFrame) setFrameType(frameType byte) {
	frame[1] = frameType
}

func (frame AhriFrame) from() string {
	var buf bytes.Buffer
	for _, v := range frame[2:4] {
		if v == 0 {
			break
		}
		buf.WriteByte(v)
	}
	return buf.String()
}

func (frame AhriFrame) setFrom(from string) {
	source := []byte(from)
	target := frame[2:4]
	for i := range target {
		target[i] = 0
	}
	for i, v := range source {
		if i == 2 {
			break
		}
		target[i] = v
	}
}

func (frame AhriFrame) to() string {
	var buf bytes.Buffer
	for _, v := range frame[4:6] {
		if v == 0 {
			break
		}
		buf.WriteByte(v)
	}
	return buf.String()
}

func (frame AhriFrame) setTo(to string) {
	source := []byte(to)
	target := frame[4:6]
	for i := range target {
		target[i] = 0
	}
	for i, v := range source {
		if i == 2 {
			break
		}
		target[i] = v
	}
}

func (frame AhriFrame) connId() uint64 {
	return BytesToUint64(frame[6:14])
}

func (frame AhriFrame) setConnId(connId uint64) {
	copy(frame[6:14], Uint64ToBytes(connId))
}

func (frame AhriFrame) setPayloadLen(payloadLen uint16) {
	copy(frame[14:16], Uint16ToBytes(payloadLen))
}

func (frame AhriFrame) payloadLen() uint16 {
	return BytesToUint16(frame[14:16])
}

func (frame AhriFrame) payload() []byte {
	return frame[AfpHeaderLen : AfpHeaderLen+BytesToUint16(frame[14:16])]
}

func (frame AhriFrame) setPayload(payload []byte) AhriFrame {
	payloadLen := uint16(len(payload))
	copy(frame[14:16], Uint16ToBytes(payloadLen))
	copy(frame[AfpHeaderLen:AfpHeaderLen+payloadLen], payload)
	return frame[0 : AfpHeaderLen+payloadLen]
}

func (frame AhriFrame) free() {
	if frame == nil || cap(frame) < AfpFrameMaxLen {
		return
	}
	//There must be a type conversion here, otherwise an error will be encountered when call "pool.Get().([]byte)"
	ByteArrPool.Put([]byte(frame))
}

func NewAhriFrame(source []byte) AhriFrame {
	buf := ByteArrPool.Get()
	copy(buf, source)
	return buf[0:len(source)]
}
