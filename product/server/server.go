package main

import (
	"flag"
	"fmt"
	"github.com/GavinGuan24/ahri/core"
	"os"
)

var server *core.AhriServer

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `Usage: %s <server-info> [global-cfg]
    server-info: -ip serverIp, -p serverPort, -k serverPassword, -a rsaPrivateKey, -b rsaPublicKey
    global-cfg: -L logLevel, -T timeoutUnitSec

Parameters:
`, os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {

	var logLevel int
	var timeoutUnit int

	var serverIp string
	var serverPort string
	var serverPassword string

	var privateKey string
	var publicKey string

	flag.IntVar(&logLevel, "L", int(core.LevelError), "the log level, 0 ~ 3 ==> debug, info, warn, error")
	flag.IntVar(&timeoutUnit, "T", 5, "the timeout of one-way communication time interval between an AhriClient and an AhriServer;\r\nSpecial: AhriClient Dial timeout = 3T, heartbeat timeout = 2T")

	flag.StringVar(&serverIp, "ip", "", "the IP of an ahri server")
	flag.StringVar(&serverPort, "p", "", "the port of an ahri server")
	flag.StringVar(&serverPassword, "k", "", "the password of an ahri server")

	flag.StringVar(&privateKey, "a", "rsa_private_key.pem", "the private rsa key file of this ahri server")
	flag.StringVar(&publicKey, "b", "rsa_public_key.pem", "the public rsa key file of this ahri server")
	flag.Parse()

	if serverIp == "" || serverPort == "" || serverPassword == "" {
		fmt.Printf("Ahri Server version: %s\nYou can learn more: %s -h\n", core.Version, os.Args[0])
		os.Exit(1)
	}

	switch logLevel {
	case int(core.LevelDebug), int(core.LevelInfo), int(core.LevelWarn), int(core.LevelError):
	default:
		logLevel = int(core.LevelError)
	}
	core.Log = &core.Alog{LowLevel: core.LogLevel(logLevel)}
	core.Debug = core.Log

	if timeoutUnit > 0 {
		core.AhriTimeoutSec = timeoutUnit
	} else {
		core.AhriTimeoutSec = 5
	}

	server = core.NewAhriServer(serverIp, serverPort, serverPassword, privateKey, publicKey)
	fmt.Printf("Ahri Server (%s) is running.\n", core.Version)
	select {}
}
