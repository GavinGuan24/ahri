package core

import (
	"time"
)

const (
	// the timeout of a socks5 requester shake with an AhriSocks5Server.
	Socks5ShakeTimeoutMillisec = 400
	// the timeout of an AhriClient registers itself with an AhriServer.
	AhriShakeTimeoutSec = 3

	// the maximum number of proxy frames that an AhriClientPartner can save in a special golang channel.
	AhriClientPartnerProxyFrameMaxSize = 10240
	// the maximum number of proxy frames that an AhriServer can save in a special golang channel.
	AhriServerProxyFrameMaxSize = AhriClientPartnerProxyFrameMaxSize * 20

	// Addr Type
	Socks5AddrTypeIPv4   = 0x01
	Socks5AddrTypeDomain = 0x03
	Socks5AddrTypeIPv6   = 0x04

	// Client Mode: 'Take': Serviced, 'Give': Service provider, 'Trader': Both
	AhriClientModeTake  = 0
	AhriClientModeGive  = 1
	AhriClientModeTrade = 2

	// Special Addr Name
	AhriAddrNameLocal      = "L"
	AhriAddrNameAhriServer = "S"
	AhriAddrNameForbidden  = "-"
	AhriAddrNameIntercept  = "|"

	// ARP ACK
	ArpAckOk                            = 0x00
	ArpAckWrongPassword                 = 0x01
	ArpAckClientNameIsAlreadyRegistered = 0x02
	ArpAckUnknownMode                   = 0x03
	ArpAckIllegalAesPassword            = 0x04

	// AFP
	AfpFlag          = 0x24
	AfpHeaderLen     = 16
	AfpFrameMaxLen   = 4096
	AfpPayloadMaxLen = AfpFrameMaxLen - AfpHeaderLen

	// AFP Frame type
	AfpFrameTypeHeartbeat    = 0x00
	AfpFrameTypeDirect       = 0x01
	AfpFrameTypeProxy        = 0x02
	AfpFrameTypeDial         = 0x03
	AfpFrameTypeDialAck      = 0x04
	AfpFrameTypeDialProxy    = 0x05
	AfpFrameTypeDialProxyAck = 0x06

	// AFP ACK
	AfpAckOk  = 0x00
	AfpAckErr = 0x01

	// ignore errors
	IgnoreErrorSendOnClosedChannel  = "send on closed channel"
	IgnoreErrorInvalidMemoryAddress = "runtime error: invalid memory address or nil pointer dereference"
	IgnoreErrorSliceBoundsOutOfRange = "runtime error: slice bounds out of range"
	IgnoreErrorInvalidBufferOverlap= "crypto/cipher: invalid buffer overlap"
)

var (
	Log         Logger
	Debug       Logger
	ByteArrPool = NewByteArrPool(AfpFrameMaxLen)

	// the timeout of one-way communication time interval between an AhriClient and an AhriServer;
	// AhriClient.Dial().timeout = 3 * AhriTimeoutSec
	// heartbeat timeout = 2 * AhriTimeoutSec
	AhriTimeoutSec int

	NoDeadline = time.Time{}

	// Socks5 ACK
	Socks5NoAcceptableMethods           []byte
	Socks5SecretFree                    []byte
	Socks5ConnectionNotAllowedByRuleset []byte
	Socks5UnsupportedCommand            []byte
	Socks5UnsupportedAddressType        []byte
	Socks5Success                       []byte
)

func init() {
	noAcceptableMethodsBytesArr := [2]byte{0x05, 0xFF}
	Socks5NoAcceptableMethods = noAcceptableMethodsBytesArr[:]

	secretFreeBytesArr := [2]byte{0x05, 0x00}
	Socks5SecretFree = secretFreeBytesArr[:]

	connectionNotAllowedByRulesetBytesArr := [22]byte{0x05, 0x02, 0x00, 0x04,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00}
	Socks5ConnectionNotAllowedByRuleset = connectionNotAllowedByRulesetBytesArr[:]

	unsupportedCommandBytesArr := [22]byte{0x05, 0x07, 0x00, 0x04,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00}
	Socks5UnsupportedCommand = unsupportedCommandBytesArr[:]

	unsupportedAddressTypeBytesArr := [22]byte{0x05, 0x08, 0x00, 0x04,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00}
	Socks5UnsupportedAddressType = unsupportedAddressTypeBytesArr[:]

	successBytesArr := [22]byte{0x05, 0x00, 0x00, 0x04,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00,
		0x00, 0x00}
	Socks5Success = successBytesArr[:]
}
