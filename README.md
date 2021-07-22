## 前言

Ahri 这个实验性的项目我已经停止维护了。原因是，现在我没有同时访问多内网的需求。而且 Ahri 的**科学上网**的能力，在当时我也是顺带实现的。维护一个帧化协议确实有点累。
嗯，现在我对于**科学上网**的需求确实增加了。因 Ahri 是基于单 TCP 链的帧化协议，所以效果确实不太好。
这主要是因为国内访问国外是有审查的，有些线路的延迟也真的是高。导致 Ahri 在有些时段经常重连，没有提供有效的**科学上网**流量。

**其实我已经独立完成了另一个专门用于科学上网的工具，我称呼ta为 SpreadX**。关于ta的能力，我只能说非常不错。我用了很久，还开发了 macOS 的客户端。
我国（中国🇨🇳）国民素质逐步提高，我愿意相信能看到本文的人都是有较高素质的国民，不会做破坏国家安全的事情。

**所以，我已将 SpreadX 直接 release 到 Ahri 的 release 页面**。
但是，为了这个工具能存活下去，**我不会提供源码**。

sha-256 cb65b0f58cc113cd8d897a4eb8fa532d8ccc2a56b7c7c0bbddb9df558fe2155b

如果有人下载了它，并担心使用时有安全问题，请自行搭建测试环境验证，或者反编译解析。
本人承诺，该工具源码仅包含该工具自身的能力，不含有任何恶意代码。

时间（2021-07-22 18:07:00）

# Ahri

[Ahri](https://github.com/GavinGuan24/ahri) 是一个高效且易用的网络环境共享工具。

你可以将它理解为一个 VPN，但 ta 又不是一个传统意义上的 VPN。
你只需要有一个拥有公网 IP 的 VPS 便可以使用 Ahri 同时访问多个内网环境，并且支持使用公网服务器的网络环境。

Ahri 只想做好三件事。

1. 核心：一台电脑同时使用多个内网环境，且不干预其他正常的网络访问。
2. 辅助：允许使用公网服务器的网络环境。
3. 易用：暴露出尽可能少的配置参数，避免给使用者带来困扰，降低学习使用成本。同时，尽可能的为每一个配置参数给出通用的默认值，将复杂与技术细节留给开发者。

请不要将 Ahri 与 [frp](https://github.com/fatedier/frp)，SSR，V2Ray 等项目做比较，因为 Ahri 对使用场景的定位与它们不同。

当然，如果你有优化 Ahri 的意见和建议，我很乐意接受。

- [Ahri 的特性](#feature)
- [编写 Ahri 的初衷](#original_intention)
- [Ahri 的使用场景](#situation)
- [Ahri 的工作原理](#principle)
    - [ahri-server & ahri-client](#server_client)
    - [Ahri Protocal](#ahri_protocal)
    - [ahri.hosts 示例](#ahri_hosts_examples)
    - [场景解析](#analysis)
- [Ahri 的用法](#usage)
- [Ahri http(s)代理实践](#practice)
- [Q & A](#Q_A)

## <a id="feature">Ahri 的特性</a>

|特性|详情|
|:--:|:--|
|通用性|Ahri 基于 TCP，所以不关心转发的数据包是基于何种应用层协议的，也就是说 Ahri 一定支持 http。理论上讲，ta 支持所有基于 TCP 的上层协议。|
|帧化数据协议|Ahri 拥有自己的应用层协议（Ahri protocol）来经行流量转发。Ahri 对流量的处理方式类似于 http 2.0，所有的数据均是 **帧化** 的。|
|安全|Ahri 采用了 **RSA**, **AES-256-CFB** 加密算法保证数据的安全性。|
|高效|Ahri 由 golang 编写，本身非常底层，执行效率极高。同时，Ahri 采用 **多路复用** 模式，在 client 与 server 之间仅建立一个 TCP 连接，避免不必要的协议握手。|
|低内存, 低 CPU 使用率|采用 **sync.pool** 降低 GC 压力与内存占用量。* ahri-client 活跃时，MEM < 7.0MB，%CPU ~= 10%，%CPU max < 15.0% 。* ahri-server 活跃时，MEM < 15.0MB，%CPU ~ 1.5%，%CPU max < 2.1%。|
|跨平台|Ahri 由 golang 编写，轻松跨平台；|

*最简模型下的内存测试环境与情景：

- 在中国大陆连续访问数个 YouTube 1080p 视频播放页（即，当视频开始播放时，马上点击下一个视频地址，使机器保持数个持续的大流量的网络访问）。
- ahri-client 运行在 MacBook Pro <Retina, 13-inch, Early 2015> 乞丐版上。
- ahri-server 运行在 Vultr 的 LA 节点的 VPS 上 <CentOS 7 x64 5.1.14-1.el7.elrepo.x86_64, CPU 1 vCore, RAM 1GB,> 。

## <a id="original_intention">编写 Ahri 的初衷</a>

在工作中，因商务合作需要使用几家合作公司的内网环境，导致我需要频繁切换 VPN 配置。
而回到家中，如果偶遇突发情况又无法使用自己公司的内网（因为运维不给配）。
TeamViewer 个人版虽然免费，但时不时会出现服务器宕机的情况。
女友公司的运维配置的 VPN 不稳定。

一系列的恼人事件发生后，我决定编写 Ahri 来改善我的工作与生活质量。同时，希望 Ahri 可以帮助那些遇到类似情况的人。

**注意: 请严格遵守你所在地区的相关法律法规, 不要将 Ahri 用于违法犯罪行为(尤其是科学上网); 否则后果自负, 毕竟技术无罪, 最坏的情况我会清除源码.**

 
## <a id="situation">Ahri 的使用场景</a>

Ahri 适用于但不局限于以下场景，ta 可以解决这些问题。

**场景一**

你的工作内容需要使用到公司内网，但由于外因，没有 VPN 或根本无法使用。

TeamViewer 无法连接到公司内网的电脑或使用时及其卡顿。

作为开发，需要使用到几个中间件，而它们都在公司内网。

**场景二**

因商务合作需要使用几家合作公司的内网环境，频繁切换 VPN 配置非常麻烦。

**场景三**

你所在的网络环境对一些公网域名或 IP 进行了拦截，导致你无法使用它们。

## <a id="principle">Ahri 的工作原理</a>

Ahri 需要解决的问题其实是流量转发。然而并不是所有的流量都需要转发，或多次转发。
所以，就请求的发起来说，流量的目的地有三种类型。

1. 本地：在本机直接拨号（dial）TCP 请求。
2. ahri-server： 使用 ahri-server 作为代理，在 ahri-server 上 dial TCP 请求。
3. 另一个 ahri-client： 使用另一个 ahri-client 作为代理，在另一个 ahri-client 上 dial TCP 请求。

### <a id="server_client">ahri-server & ahri-client</a>

Ahri 服务由两个二进制程序来提供，它们是 ahri-server，ahri-client。

- ahri-server 负责响应来自 ahri-client 的请求，或者转发一个 ahri-client 的请求给另一个 ahri-client。

- ahri-client 负责发起请求，或者响应另一个 ahri-client 的请求。ta 有三个模式。
    - take：启动一个 socks5 服务来接受本地的所有 TCP 请求；再按配置好的映射文件（ahri.hosts）决定采用上面的三种流量目的地中的哪一个。
    - give：仅负责响应来自其他 ahri-client 的请求。
    - trade：同时支持上面两种的模式。

ahri-client 采用主动注册到 ahri-server 的方式来进行连接，而 ahri-server 会对数个 ahri-client 进行管理。
ahri-client 注册到 ahri-server 后，它们之间会有一个内容加密的 TCP 连接。在这个连接中会传递心跳包与各类数据包。

然后再解释上面挖的一个坑，对一个 TCP 连接经行多路复用。

### <a id="ahri_protocal">Ahri Protocal</a>

基于 TCP ，Ahri 自行定义了一个应用层协议 [Ahri Protocol](https://github.com/GavinGuan24/ahri/blob/master/core/ahri_protocol.md)。
ta 由 Ahri Registe Protocol（ARP）与 Ahri Frame Protocol（AFP）组成。
ARP 定义了 ahri-client 应该怎样注册到 ahri-server。
AFP 定义了 ahri-client 与 ahri-server 应该怎样交互数据包。
详细的内容请查看[Ahri Protocol](https://github.com/GavinGuan24/ahri/blob/master/core/ahri_protocol.md)。

在这里我仅想说明，协议中出现了 Frame，表明 Ahri 将数据做了帧化，对数据包的管理方式类似于 HTTP 2.0。在这样的前提下，仅使用一个 TCP 连接便可以使 ahri-client 与 ahri-server 沟通顺畅。

为了让 ahri-server 管理数个 ahri-client，ARP 规定每一个 ahri-client 均有一个名字（最大长度为 2 的ACSII字符）。
每一个 ahri-client 均有一个 ahri.hosts 文件，可以控制本地请求转发的目的地；也可控制是否对他人提供某些域名或 IP 的转发服务。
**其中 'S', 'L', '-', '|' 为系统保留名, 禁止使用它们对 ahri-client 命名.**

#### <a id="ahri_hosts_examples">ahri.hosts 示例</a>

我们假设自己的客户端名为 'A', 另一个客户端名为 'B', 且均注册至服务端 S, 以下就是 ahri.hosts 文件的示例.

```
# 转发本地请求至服务端
youtube.com S

# 转发本地请求至另一个客户端 B
mary-live.com B

# 当前 ahri-client 为 give 或 trade 模式时, 禁止其他客户端访问本地网络环境中的这个域名
tom-live.com -

# 当前 ahri-client 遇到该域名时, 拦截本地请求, 一般用于广告屏蔽
ad-live.com |

# 显式表明请求应该在本地处理, 当然所有请求的默认目的地就是 L, 所以可以不写下面这一行.
baidu.com L
```

### <a id="analysis">场景解析</a>

我们回顾一下上面的使用场景。
假设我本机 ahri-client 是 A，
我使用的 ahri-server 是 S，
我公司内网（LAN 1）中有一个 ahri-client 是 B，
别人公司内网（LAN 2）中有一个 ahri-client 是 C。

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

**场景三**

对这个场景的处理其实是我在实现场景一、二的过程中顺带写出来的衍生物。
主要就是你所在的网络环境会对一些公网资源的请求进行拦截。
A -> Internet 这条路不通了。所以换线为 A -> S -> Internet。

希望我对上面的场景的解决描述的足够清楚。


![示意图](https://github.com/GavinGuan24/ahri/blob/master/img/a0.jpg)

## <a id="usage">Ahri 的用法</a>

我已经对常用的系统完成了源码编译的工作, 你应该可以在 [releases](https://github.com/GavinGuan24/ahri/releases/tag/v0.9.3) 中找到可运行在你系统上的版本. 如果没有, 请自行从源码编译(go1.12.1+).

详细参数与解释仅需要在命令行下执行对应的帮助程序（**因为 windows 的限制，需要将 ahri-client 与 ahri-server 先重命名为 ahri-client.exe 与 ahri-server.exe**）

```
客户端
ahri-client -h

服务端
ahri-server -h

# windows 下

ahri-client.exe -h
ahri-server.exe -h
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

## <a id="practice">Ahri http(s)代理实践</a>

Ahri 其实很简单，但对于不熟悉终端的童鞋还是太难了，那么跟着这个[【实践教程】](https://github.com/GavinGuan24/ahri/blob/master/core/ahri_guide.md)，你可以学习到基础用法。


## <a id="Q_A">Q & A</a>

我已测试并正常使用该工具多时，有任何意见与建议或者问题，请 open 一个 [issues](https://github.com/GavinGuan24/ahri/issues)。

### ahri-client 的命名

ahri-client 可以使用最长 2 个 ASCII 字符（你就当做两个英文字母好了）来命名自身。同时 'S'， 'L'， '-'， '|' 均是系统保留名，禁止使用。
为什么最长 2 个字符？由 ARP 决定的。如此，一台服务器已经可以选 (256^2 - 4) 个客户端名，足够使用。

### windows 的命令行中无法运行 Ahri 的二进制文件

因为 Ahri 的交叉编译脚本中，默认使用 ahri-client 与 ahri-server 这两个名字命名二进制文件，而 windows 要求可执行文件以 `exe` 作为后缀名。
所以，将 ta 们重命名加上 `.exe` 即可。

### 关于其他常用工具的对接 Ahri

一般来说，至少在 *nix 环境下绝大多数工具的对接都可以使用 nc 来解决。

下面是 ssh 使用 nc 来对接 Ahri 的例子。

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


