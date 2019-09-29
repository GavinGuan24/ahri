package main

import (
	"flag"
	"fmt"
	"github.com/GavinGuan24/ahri/core"
	"os"
)

var (
	client       *core.AhriClient
	socks5Server *core.AhriSocks5Server
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(flag.CommandLine.Output(), `Usage: %s <server-info> <client-info> [socks5-cfg]
    server-info: -sip serverIp, -sp serverPort, -k serverPassword
    client-info: -n clientName, -m clientMode
    socks5-cfg:  -s5ip socks5Ip, -s5p socks5Port, -f ahriHostsFile

Parameters:
`, os.Args[0])
		flag.PrintDefaults()
	}
}

func main() {

	var logLevel int

	var serverIp string
	var serverPort string
	var serverPassword string

	var clientName string
	var clientMode int

	var socks5Ip string
	var socks5Port string
	var ahriHostsPath string

	flag.IntVar(&logLevel, "L", int(core.LevelError), "the log level, 0 ~ 3 ==> debug, info, warn, error")

	flag.StringVar(&serverIp, "sip", "", "the IP of an ahri server")
	flag.StringVar(&serverPort, "sp", "", "the port of an ahri server")
	flag.StringVar(&serverPassword, "k", "", "the password of an ahri server")

	flag.StringVar(&clientName, "n", "", "the name of this ahri client")
	flag.IntVar(&clientMode, "m", 0, "the work mode of this ahri client, 0: Take, 1: Give, 2: Trade")

	flag.StringVar(&socks5Ip, "s5ip", "127.0.0.1", "the socks5 IP of this ahri client")
	flag.StringVar(&socks5Port, "s5p", "23456", "the socks5 port of this ahri client")
	flag.StringVar(&ahriHostsPath, "f", "ahri.hosts", "the ahri hosts file of this ahri client")
	flag.Parse()

	if serverIp == "" || serverPort == "" || serverPassword == "" || clientName == "" {
		fmt.Printf("Ahri Client version: %s\nYou can learn more: %s -h\n", core.Version, os.Args[0])
		os.Exit(1)
	}

	switch logLevel {
	case int(core.LevelDebug), int(core.LevelInfo), int(core.LevelWarn), int(core.LevelError):
	default:
		logLevel = int(core.LevelError)
	}

	core.Log = &core.Alog{LowLevel: core.LogLevel(logLevel)}
	core.Debug = core.Log

	client = core.NewAhriClient(serverIp, serverPort, serverPassword, clientName, clientMode)
	socks5Server = core.NewAhriSocks5Server(socks5Ip, socks5Port, ahriHostsPath, client)
	fmt.Printf("Ahri Client (%s) is running.\n", core.Version)
	select {}
}
