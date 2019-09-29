package core

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/binary"
	"encoding/pem"
	"errors"
	"io"
	mrand "math/rand"
	"net"
	"strconv"
	"strings"
)

func ValidIp(ip string) bool {
	targetIP := net.ParseIP(ip)
	return targetIP != nil
}

func ValidPort(port string) bool {
	if intPort, e := strconv.Atoi(port); e == nil {
		return 0 <= intPort && intPort <= 65535
	} else {
		return false
	}
}

// ip + port ==> format like 127.0.0.1:80, [xx:xx::xx:xx::xx]:80
func NetAddrString(ip, port string) string {
	var addr string
	if strings.Contains(ip, ".") {
		addr = ip + ":" + port
	} else {
		addr = "[" + ip + "]:" + port
	}
	return addr
}

//lines: lines of the hosts config file.
//ahriHostsPath: path of the hosts config file.
func ParseAddrCtxMapper(lines []string, ahriHostsPath string) map[string]string {
	newAddrCtxMapper := make(map[string]string, 16)
	for lineIndex, line := range lines {
		line = strings.Trim(line, "\n\r\t \f\v")
		if strings.HasPrefix(line, "#") {
			continue
		}
		//"a.c S" is the shortest mode.
		if len(line) < 5 {
			Log.Debugf("Ignored line in the File(%s: %d).", ahriHostsPath, lineIndex+1)
			continue
		}
		line = strings.ReplaceAll(line, "\t", " ")
		lineParts := strings.Split(line, " ")
		if len(lineParts) < 2 {
			Log.Debugf("Ignored line in the File(%s: %d).", ahriHostsPath, lineIndex+1)
			continue
		}

		host := lineParts[0]
		ahriAddrName := lineParts[len(lineParts)-1]

		if len(ahriAddrName) == 0 {
			Log.Debugf("Ignored line in the File(%s: %d).", ahriHostsPath, lineIndex+1)
			continue
		}
		if AhriAddrNameLocal == ahriAddrName {
			//AhriAddrNameLocal is default value, no need record.
			continue
		}
		byteAddr := make([]byte, 0, 16)
		byteIP := net.ParseIP(host)
		if byteIP == nil {
			if strings.HasPrefix(host, "*.") {
				host = strings.TrimPrefix(host, "*.")
			}
			if strings.HasPrefix(host, ".") {
				host = strings.TrimPrefix(host, ".")
			}
			byteAddr = append(byteAddr, Socks5AddrTypeDomain)
			byteAddr = append(byteAddr, []byte(host)...)
		} else {
			v4IP := byteIP.To4()
			if v4IP != nil {
				byteIP = v4IP
			}
			if len(byteIP) == net.IPv4len {
				byteAddr = append(byteAddr, Socks5AddrTypeIPv4)
				byteAddr = append(byteAddr, byteIP...)
			} else {
				byteAddr = append(byteAddr, Socks5AddrTypeIPv6)
				byteAddr = append(byteAddr, byteIP...)
			}
		}
		newAddrCtxMapper[string(byteAddr)] = ahriAddrName
	}
	return newAddrCtxMapper
}

//return a ahriAddrName.
func MapperAhriAddrName(atyp byte, byteAddr []byte, mapper map[string]string) string {

	byteIP := net.ParseIP(string(byteAddr))
	if byteIP != nil {
		v4IP := byteIP.To4()
		if v4IP != nil {
			atyp = Socks5AddrTypeIPv4
			byteAddr = v4IP
		} else {
			atyp = Socks5AddrTypeIPv6
			byteAddr = byteIP
		}
	}

	key := make([]byte, 0, 16)
	key = append(key, atyp)
	key = append(key, byteAddr...)
	targetName := mapper[string(key)]
	if Socks5AddrTypeIPv4 == atyp || Socks5AddrTypeIPv6 == atyp {
		if targetName == "" {
			return AhriAddrNameLocal
		} else {
			//targetName is AhriAddrNameAhriServer("S") or a name of another Ahri Client.
			return targetName
		}
	} else {
		if targetName != "" {
			return targetName
		}
		//check parent domain
		split := strings.Split(string(byteAddr), ".")
		lenght := len(split)
		if lenght <= 2 {
			return AhriAddrNameLocal
		} else {
			parentDomain := split[lenght-2] + "." + split[lenght-1]
			key2 := make([]byte, 0, 16)
			key2 = append(key2, Socks5AddrTypeDomain)
			key2 = append(key2, []byte(parentDomain)...)
			targetName2 := mapper[string(key2)]
			if targetName2 == "" {
				return AhriAddrNameLocal
			} else {
				return targetName2
			}

		}
	}
}

// =================== AES-256-CFB Encrypt ======================
//quote from this url(https://blog.csdn.net/u012978258/article/details/87868999)
func EncryptAesCfb256(origData []byte, aesCipher cipher.Block) (encrypted []byte) {
	encrypted = ByteArrPool.Get(aes.BlockSize + len(origData))
	iv := encrypted[:aes.BlockSize]
	io.ReadFull(rand.Reader, iv)
	stream := cipher.NewCFBEncrypter(aesCipher, iv)
	stream.XORKeyStream(encrypted[aes.BlockSize:], origData)
	return encrypted
}

// =================== AES-256-CFB Decrypt ======================
//quote from this url(https://blog.csdn.net/u012978258/article/details/87868999)
func DecryptAesCfb256(encrypted []byte, aesCipher cipher.Block) (decrypted []byte) {
	iv := encrypted[:aes.BlockSize]
	encrypted = encrypted[aes.BlockSize:]
	stream := cipher.NewCFBDecrypter(aesCipher, iv)
	stream.XORKeyStream(encrypted, encrypted)
	return encrypted
}

func GenerateAes256Key() [32]byte {
	var key [32]byte
	for i := 0; i < 32; i++ {
		key[i] = byte(mrand.Intn(256))
	}
	return key
}

// =================== RSA Encrypt ======================
//quote from this url(http://www.361way.com/golang-rsa-aes/5828.html)
func EncryptRsa(origData []byte, pubKey []byte) ([]byte, error) {
	block, _ := pem.Decode(pubKey)
	if block == nil {
		return nil, errors.New("public key error")
	}
	pubInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	pub := pubInterface.(*rsa.PublicKey)
	return rsa.EncryptPKCS1v15(rand.Reader, pub, origData)
}

// =================== RSA Encrypt ======================
//quote from this url(http://www.361way.com/golang-rsa-aes/5828.html)
func DecryptRsa(ciphertext []byte, priKey []byte) ([]byte, error) {
	block, _ := pem.Decode(priKey)
	if block == nil {
		return nil, errors.New("private key error")
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	return rsa.DecryptPKCS1v15(rand.Reader, priv, ciphertext)
}

func Uint64ToBytes(n uint64) []byte {
	bs := make([]byte, 8)
	binary.BigEndian.PutUint64(bs, n)
	return bs
}

func BytesToUint64(bs []byte) uint64 {
	return binary.BigEndian.Uint64(bs)
}

func Uint16ToBytes(n uint16) []byte {
	bs := make([]byte, 2)
	binary.BigEndian.PutUint16(bs, n)
	return bs
}

func BytesToUint16(bs []byte) uint16 {
	return binary.BigEndian.Uint16(bs)
}
