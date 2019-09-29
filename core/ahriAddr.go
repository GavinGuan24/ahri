package core

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"strconv"
)

type AhriAddr struct {
	name     string
	addrType int
	dstAddr  []byte
	port     int
}

func (ahriAddr *AhriAddr) String() string {
	var buf bytes.Buffer
	buf.WriteString("AhriAddr(name: ")
	buf.WriteString(ahriAddr.name)
	buf.WriteString(", atyp: ")
	buf.WriteString(strconv.Itoa(ahriAddr.addrType))
	buf.WriteString(", dstAddr: ")
	if Socks5AddrTypeDomain == ahriAddr.addrType {
		buf.WriteString(string(ahriAddr.dstAddr))
	} else {
		buf.WriteString(net.IP(ahriAddr.dstAddr).String())
	}
	buf.WriteString(", port: ")
	buf.WriteString(strconv.Itoa(ahriAddr.port))
	buf.WriteByte(')')
	return buf.String()
}

func (ahriAddr *AhriAddr) ParseDstAddrIP() error {
	switch ahriAddr.addrType {
	case Socks5AddrTypeIPv4:
		Log.Debug("This is already an ipv4 address.")
		return nil
	case Socks5AddrTypeIPv6:
		Log.Debug("This is already an ipv6 address.")
		return nil
	case Socks5AddrTypeDomain:
		addr, e := net.ResolveIPAddr("ip", string(ahriAddr.dstAddr))
		if e != nil {
			return e
		}
		v4Ip := addr.IP.To4()
		if v4Ip != nil {
			ahriAddr.dstAddr, ahriAddr.addrType = v4Ip, Socks5AddrTypeIPv4
		} else {
			ahriAddr.dstAddr, ahriAddr.addrType = addr.IP, Socks5AddrTypeIPv6
		}
		return nil
	default:
		return errors.New(fmt.Sprintf("Unsupported ATYP (value: %x) of Socks5 Protocol.", ahriAddr.addrType))
	}
}
