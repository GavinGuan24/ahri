# Ahri

[Ahri](https://github.com/GavinGuan24/ahri) 是一个好用且便于配置的网络环境共享工具。

Ahri 只想做好三件事。

1. 暴露出尽可能少的配置参数，避免给使用者带来困扰，降低学习使用成本。同时，尽可能的为每一个配置参数给出通用的默认值，将复杂与技术细节留给开发者。
2. 核心功能，一台电脑同时使用多个内网环境。
3. 辅助功能，允许使用公网服务器的网络环境。

请不要将 Ahri 与 [frp](https://github.com/fatedier/frp)，SSR，V2Ray 等项目做比较，因为 Ahri 的使用场景定位就与它们不同。
当然，如果你有优化 Ahri 的意见和建议，我很乐意接受。

- ta 基于 TCP，所以不关心转发的数据包是基于何种应用协议的，也就是说 ahri 肯定支持 http。理论上说 ta 支持所有基于 TCP 的上层协议。
- ta 有自己的应用层协议（Ahri protocol）来经行流量转发。
- ta 在客户端与服务端之间仅建立唯一的一个 TCP 连接进行多路复用，避免不必要的协议握手。
- ta 在处理流量时，做法类似于 http 2.0，所以，所有的数据均是帧化的。
- ta 采用了 RSA, AES-256-CFB 加密算法保证数据的安全性。

你可以将它理解为一个 VPN，但 ta 又不是仅是一个 VPN，因为使用 ta，你可以同时使用数个内网环境（含公网服务器的网络环境）。

## Ahri 的使用场景

你一定遇到过这些令人痛苦的场景。

**场景一**

我是一个程序狗。我的工作内容需要使用到公司内网，但运维不靠谱，VPN 没法使用或者修改了一些配置没有及时使用邮件通知到我，或公司直接不提供 VPN。假设我已经在家了，但 Leader 告诉我有一个 BUG 需要立即处理。
没有 VPN ，难道要我再跑到公司去处理吗？再者，我家带宽一定没有公司的带宽大，Teamviewer 用起来不卡吗？

如果你是个 Java 后端，你的项目需要使用到几个中间件，而它们（包括测试用的 DB）都在公司内网。难道你要在自己电脑上安装几个中间件然后再造一些数据出来？不麻烦吗？

**场景二**

两家 IT 公司进行了合作，但是员工的工作内容需要用到自己公司的内网和对方公司的内网。这时怎么办呢？
员工在家的时候也完全无法使用到双方公司的内网环境。

让两家公司的运维都配置一个 OpenVPN（或者别的 VPN）？
就算有条件，双方使用者配置起来需要各种参数，ta 不麻烦吗？
而且部分配置冲突了需要怎么处理？

**场景三**

我们公司的运维对一些公网域名或 IP 进行了拦截，或者是你所在的网络环境对一些公网域名或 IP 进行了拦截。你又需要使用它们，怎么办？

Ahri 适用于但不局限于上述场景，ta 可以解决这些问题。

**注意: 请严格遵守你所在地区的相关法律法规, 不要将 Ahri 用于违法犯罪行为(尤其是科学上网); 否则后果自负, 毕竟技术无罪, 最坏的情况我会清除源码.**

## Ahri 的工作原理

Ahri 需要解决的问题其实是流量转发。然而并不是所有的流量都需要转发，或多次转发。
所以，就请求的发起来说，流量的目的地有三种类型。

1. 本地：在本机直接 dial TCP 请求。
2. ahri-server： 在 ahri-server 上 dial TCP 请求。
3. 另一个 ahri-client： 在另一个 ahri-client 上 dial TCP 请求。

### ahri-server & ahri-client

Ahri 服务由两个二进制程序来提供，它们是 ahri-server，ahri-client。

- ahri-server 负责响应来自 ahri-client 的请求，或者转发一个 ahri-client 的请求给另一个 ahri-client。

- ahri-client 负责发起请求，或者响应另一个 ahri-client 的请求。ta 有三个模式。
    - take：启动一个 socks5 服务来接受本地的所有 TCP 请求；再按配置好的映射文件（ahri.hosts）决定采用上面的三种流量目的地中的哪一个。
    - give：仅负责响应来自其他 ahri-client 的请求。
    - trade：同时支持上面两种的模式。

到这里，你可能已经猜到，ahri-client 与 ahri-server 的关系是注册与管理。
没错，ahri-client 采用主动注册到 ahri-server 的方式来进行连接。
ahri-client 与 ahri-server 之间采用 RSA, AES-256-CFB 加密算法保证数据的安全性。

ahri-client 注册到 ahri-server 后，它们之间就有了一个 TCP 连接。在这个连接中会传递心跳包，数据包。

至此，我们再回顾一下上面的使用场景。
假设我本机 ahri-client 是 A，
我使用的 ahri-server 是 S，
我公司内网（LAN 1）中有一个 ahri-client 是 B，
别人公司内网（LAN 2）中有一个 ahri-client 是 C。

### 场景解析

**场景一 & 场景二**

这两个场景是最常见的 VPN 使用场景，只不过场景二略复杂一点。

当我在自己家时。
我使用自己公司的内网时，流量走向是
A -> S -> B -> LAN 1， 然后原路返回。
我也可以使用对方公司的内网，流量走向是
A -> S -> C -> LAN 2， 然后原路返回。
我也可以正常访问任何公网资源，A -> Internet。

当我在自己公司，使用自己公司的内网时，流量走向是
A -> LAN 1，然后原路返回。
我也可以使用对方公司的内网，流量走向是
A -> S -> C -> LAN 2， 然后原路返回。
我也可以正常访问任何公网资源，A -> Internet。

你可能会问，这两种情形下，怎么做的流量转发目的地映射？
上面讲过了，这里有一个配置文件 ahri.hosts。后面会说到 ta 的配置方法。

**场景三**

对这个场景的处理其实是我在实现场景一、二的过程中顺带写出来的衍生物。
主要就是你所在的网络环境会对一些公网资源的请求进行拦截。
A -> Internet 这条路不通了。所以换线为 A -> S -> Internet。

希望我对上面的场景的解决描述的足够清楚。

然后再解释上面挖的一个坑，对一个 TCP 连接经行多路复用。

### [Ahri Protocol](https://github.com/GavinGuan24/ahri/blob/master/core/ahri_protocol.md)

基于 TCP ，Ahri 自行定义了一个应用层协议 [Ahri Protocol](https://github.com/GavinGuan24/ahri/blob/master/core/ahri_protocol.md)。
ta 由 Ahri Registe Protocol 与 Ahri Frame Protocol 组成。

详细的请去项目下看。这里仅说，既然协议中出现了 Frame，就表明数据是帧化的。没错，这里的工作方式类似于 HTTP 2.0 。在这样的情形下，仅使用一个 TCP 连接就可以使 ahri-client 与 ahri-server 沟通顺畅。

每一个 ahri-client 均有一个名字（最大长度为 2 的ACSII字符）。

每一个 ahri-client 均有一个 ahri.hosts 文件，可以控制本地请求转发的目的地。也可控制是否对他人提供某些域名或 IP 的转发服务。

**其中 'S', 'L', '-', '|' 为系统保留名, 禁止使用它们对 ahri-client 命名.**


## ahri.hosts 示例

我们假设自己的客户端名为 'A', 另一个客户端名为 'B', 且均注册至服务端, 以下就是 ahri.hosts 文件的示例.

```
# 转发本地请求至服务端
youtube.com S

# 转发本地请求至另一个客户端 B
mary-live.com B

# 当本 ahri-client 为 give 或 trade 模式时, 禁止其他客户端访问本地网络环境中的域名
tom-live.com -

# 遇到该域名, 拦截本地请求, 一般用于广告屏蔽
ad-live.com |

# 显式表明请求应该在本地处理, 当然所有请求的默认目的地就是 L, 所以可以不写下面这一行.
baidu.com L
```

![示意图](https://github.com/GavinGuan24/ahri/blob/master/img/a0.jpg)

## Ahri 的用法

我已经对常用的系统完成了源码编译的工作, 你应该可以在 [releases](https://github.com/GavinGuan24/ahri/releases/tag/v0.9.1) 中找到可运行在你系统上的版本. 如果没有, 请自行从源码编译.

详细参数与解释仅需要在命令行下执行对应的帮助程序

```
客户端
ahri-client -h

服务端
ahri-server -h
```

**方法一**

直接运行 ahri-client / ahri-server, 参数如下.

```
Usage: ./ahri-client <server-info> <client-info> [socks5-cfg] [global-cfg]
    server-info: -sip serverIp, -sp serverPort, -k serverPassword
    client-info: -n clientName, -m clientMode
    socks5-cfg:  -s5ip socks5Ip, -s5p socks5Port, -f ahriHostsFile
    global-cfg: -L logLevel, -T timeoutUnitSec

Parameters:
  -L int
    	the log level, 0 ~ 3 ==> debug, info, warn, error (default 3)
  -T int
    	the timeout of one-way communication time interval between an AhriClient and an AhriServer;
    	Special: AhriClient Dial timeout = 3T, heartbeat timeout = 2T (default 5)
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
Usage: ./ahri-server <server-info> [global-cfg]
    server-info: -ip serverIp, -p serverPort, -k serverPassword, -a rsaPrivateKey, -b rsaPublicKey
    global-cfg: -L logLevel, -T timeoutUnitSec

Parameters:
  -L int
    	the log level, 0 ~ 3 ==> debug, info, warn, error (default 3)
  -T int
    	the timeout of one-way communication time interval between an AhriClient and an AhriServer;
    	Special: AhriClient Dial timeout = 3T, heartbeat timeout = 2T (default 5)
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
使用 ahri-server, 你需要 openssl 帮你生成 RSA 公私钥对, 如果你是 *nix 环境, 可以直接调用 `bash gen_rsa_keys.sh`.

**方法二**

为了降低配置难度，压缩包中已包含 start.sh，stop.sh 。
仅需要修改必要的参数即可在 *nix 环境中使用。
至于 Windows，参考 sh 脚本即可编写 bat 即可。

提示：如果 ahri-client 与 ahri-server 之间的网络环境是 **低带宽** 或 **高延迟**，请适当增大 `-T` 参数的值。

## Q & A

我已测试并正常使用该工具多时，有任何意见与建议或者问题，请 open 一个 [issues](https://github.com/GavinGuan24/ahri/issues)。

### ahri-client 的命名

ahri-client 可以使用最长 2 个 ASCII 字符（你就当做两个英文字母好了）来命名自身。同时 'S'， 'L'， '-'， '|' 均是系统保留名，禁止使用。
为什么最长 2 个字符？Ahri Protocol 制定时决定的。如此，一台服务器已经可以注册 (256^2 - 4) 个客户端。

### 关于其他常用工具的对接 Ahri

对于 ssh， 你可以使用 nc 来对接至 Ahri。

```
现在本地 socks5 监听代理是 socks5://127.0.0.1:23456
自己服务器是server.test.com

修改~/.ssh/config 文件
添加下面配置

host server.test.com
HostName server.test.com
ProxyCommand nc -X 5 -x 127.0.0.1:23456 %h %p
ServerAliveInterval 30
然后通过
ssh user@server.test.com 就可以愉快的使用 Ahri 代理登录了。
```


