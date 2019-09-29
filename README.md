# Ahri

Ahri 是一个内网共享工具, 从某种意义上来说可以认为 ta 是一种 VPN.

Ahri 只有有两个二进制可执行文件, 客户端 ahri-client, 服务端 ahri-server.

##### ahri-client & ahri-server

ahri-client 会主动注册到 ahri-server, 并采用 socks5 协议转发本地 TCP 请求.

请求的目的地有三种类型: 本地, 服务端, 另一个客户端.
也就是说, 服务端与另一个 ahri-client 的部分网络环境可以为你所用.

ahri-client 的工作模式有三种: take(仅享受服务), give(仅提供服务), trade(前两者均可).
所以你可以选择是否为他人提供网络服务.

ahri-client 与 ahri-server 之间采用 RSA, AES-256-CFB 加密算法保证数据的安全性, 并且多路复用一个 TCP 连接交换所有数据.
数据均是加密的, 且帧化的, 这一点与 http 2.0 的处理方式类似.

每一个 ahri-client 均有一个名字(最大长度为 2 的ACSII字符).
每一个 ahri-client 均有一个 ahri.hosts 文件, 可以控制本地请求转发的目的地. 也可控制是否对他人提供某些域名或 IP 的转发服务.
其中 'S', 'L', '-' 为系统保留名
我们假设自己的客户端名为 'A', 另一个客户端名为 'B', 且均注册至服务端, 以下就是 ahri.hosts 文件的示例.

```
# 转发本地请求至服务端
youtube.com S

# 转发本地请求至另一个客户端
mary-live.com B

# 禁止其他客户端访问
tom-live.com -

# 显式表明请求应该在本地处理, 当然所有请求的默认目的地就是 L, 所以可以不写下面这一行.
baidu.com L
```

## Ahri 的使用场景

1. A 公司与 B 公司的人员协作完成一项工作, 但两个公司的内网环境是不互通的, Ahri 可以将一端的请求转发给服务器, 服务器转发给另一端进行代理相应.
2. A 公司的内网环境禁止访问 taobao.com, Ahri 可以将请求转发给服务器进行代理相应.
3. 你在家无法访问公司的内网, Ahri 可以将请求转发给服务器, 服务器转发给你在公司的电脑进行代理相应.

##### 注意: 请严格遵守你所在地区的相关法律法规, 不要将 Ahri 用于违法犯罪行为; 否则后果自负, 毕竟技术无罪.

![示意图](https://github.com/GavinGuan24/ahri/img/a0.jpg)

## Ahri 的用法

我已经对常用的系统完成了源码编译的工作, 你应该可以在 releases 中找到可运行在你系统上的版本. 如果没有, 请自行从源码编译.

```
Usage: ./ahri-client <server-info> <client-info> [socks5-cfg]
    server-info: -sip serverIp, -sp serverPort, -k serverPassword
    client-info: -n clientName, -m clientMode
    socks5-cfg:  -s5ip socks5Ip, -s5p socks5Port, -f ahriHostsFile

Parameters:
  -L int
        the log level, 0 ~ 3 ==> debug, info, warn, error (default 3)
  -f string
        the ahri hosts file of this ahri client (default "ahri.hosts")
  -k string
        the password of an ahri server
  -m int
        the work mode of this ahri client, 0: Take, 1: Give, 2: Trade
  -n string
        the name of this ahri client
  -s5ip string
        the socks5 IP of this ahri client (default "127.0.0.1")
  -s5p string
        the socks5 port of this ahri client (default "23456")
  -sip string
        the IP of an ahri server
  -sp string
        the port of an ahri server

```

```
Usage: ./ahri-server <server-info>
    server-info: -ip serverIp, -p serverPort, -k serverPassword, -a rsaPrivateKey, -b rsaPublicKey

Parameters:
  -L int
        the log level, 0 ~ 3 ==> debug, info, warn, error (default 3)
  -a string
        the private rsa key file of this ahri server (default "rsa_private_key.pem")
  -b string
        the public rsa key file of this ahri server (default "rsa_public_key.pem")
  -ip string
        the IP of an ahri server
  -k string
        the password of an ahri server
  -p string
        the port of an ahri server

```

使用 ahri-client, 你几乎仅配置 ahri.hosts 即可使用.
使用 ahri-server, 你需要 openssl 帮你生成 RSA 公私钥对, 如果你是 *nix 环境, 可以直接调用 `bash gen_rsa_keys.sh`

