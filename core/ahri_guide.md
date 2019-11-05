# Ahri HTTP(S) 代理实践

本次实践我们的目标是
1. 内网穿透：在家访问公司的内网
2. 防火墙穿透：在家使用 Google（我家不能用 Google）

该教程针对有一些 linux 与 Windows 命令行基础的人，如果你基础太差，请自行学习。

ahri 有两个执行文件 ahri-server 和 ahri-client。
如果你想要让 ta 们运行在 Windows 上的话，请为这两个文件加上后缀 `.exe`。

ahri-server 运行在一个具有公网 IP 的 VPS 中即可。
我们来称这台 VPS 为 S端
- 负责接受 ahri-client 的注册。
- 负责对 ahri-client 的流量进行转发或相应。

ahri-client 运行在一个内网中的电脑上即可。
这里为了实现我们的目标，我们需要两台位于内网中的电脑才能实现内网穿透。
我们来称这两台电脑为 A端，B端
- A端（家里的电脑）：负责为本机提供请求支持（基于 socks5）。
- B端（公司的电脑）：负责为 A端 提供内网穿透服务。


## 部署服务端

为了使用 Google，我会选择一个海外的 VPS。当然香港的也是可以的。
这里实践时使用的是米国的服务器（S端），操作系统是 CentOS 7。
如果你动手能力够强，最好将 S端 的 linux 内核升级到 4.9.0 之后的版本。
应为其内置的 [BBR](https://www.vultr.com/docs/how-to-deploy-google-bbr-on-centos-7) 算法可以显着提高服务器的吞吐量并减少连接延迟。

当然不使用 BBR 也不会有致命性的影响，你可以偷懒。

SSH 登陆到 S端，操作防火墙来开放一个端口用于 ahri-server。这里随机一下，就用 35473 端口吧。

```
# 以 root 身份 ssh 登陆至服务器
ssh -p YourSSHPort root@YourIP

# 开放 35473 端口
firewall-cmd --zone=public --add-port=35473/tcp --permanent && firewall-cmd --reload
``` 

下载并解压 ahri 的 release 包

```
cd
# 下载
wget "https://github.com/GavinGuan24/ahri/releases/download/v0.9.3/ahri_0.9.3_linux_amd64.tgz"
# 解压
tar zxf ahri_0.9.3_linux_arm64.tgz
# 移除无用文件(夹)
rm -rf ahri_0.9.3_linux_arm64.tgz client
```

现在，/root 下只有 server 一个文件夹
我们简单编辑启动脚本中的参数即可准备启动 ahri-server。

```
cd server && ls
```

看见文件夹中有这几个文件
`ahri-server      gen_rsa_keys.sh  start.sh         stop.sh`

让我们用 vim 编辑 start.sh（vim 都不会的话，自行学习）

```
vim start.sh

# 将 `yourIP` 修改为 S端 的公网 IP
# 将 `yourPort` 修改为刚才我们开放的 35473 端口
# 将 `yourPswd` 修改为你想设置的密码。
```

将`启动`与`停止`脚本的权限修改为 755

```
chmod 755 start.sh stop.sh
```

修改上述 3 处之后，便可启动 ahir-server 了。日志默认输出至 a.log 文件

```
./start.sh
# 输出大致如下
Generating RSA private key, 1024 bit long modulus (2 primes)
....................................................+++++
..+++++
e is 65537 (0x010001)
writing RSA key
```

验证 ahri-server 已经正常运行

```
tail a.log
# 输出
Ahri Server (0.9.3) is running.
```

停止 ahri-server，也可以直接调用脚本

```
./stop.sh
```

至此，ahri-server 已经成功地运行在 S端了，可以对外提供服务了。

## 部署客户端

### B端的客户端

首先是 B端，公司电脑一般都是 Windows 10(64位)，我们以此为例。

下载 [Windows 64位](https://github.com/GavinGuan24/ahri/releases/download/v0.9.3/ahri_0.9.3_windows_amd64.tgz) 的压缩包。

Windows 多的是图形化的工具，所以关于 tar.gz（tgz） 的解压请自行解决，GUI 软件建议使用 7-zip。
如果你用的是 Win 10，在 cmd 中即可完成解压
```
# 这里我将文件保存在在桌面，所以先 cd 到 Desktop
cd Desktop
tar -zxf ahri_0.9.3_windows_amd64.tgz
rd /s/q server
del ahri_0.9.3_windows_amd64.tgz
```
解压后，只有 client 文件夹下的文件是我们需要的。
先将 ahri-client 加上后缀以符合 Windows 的规范。

```
move ahri-client ahri-client.exe
```

然后执行如下命令，注意 `^` 是命令行换行标识，这里可以不使用，我是为了一行一个参数看起来比较清楚。
sip 后面替换为 S端 的公网 IP
sp 后面替换为刚才我们开放的 35473 端口
k 后面替换为你刚才为 ahri-server 设置的密码
n 替换为你想称呼这个 ahri-client 的名字(最长两个ASCII字符，可以认为是英文字母与数字)，这里我们写 B
m 模式：0仅使用服务，1仅提供服务，2前两者都是；这里我们的 B端 是公司电脑，仅提供服务即可，填写 1
其他参数默认即可。

然后启动 B端 的 ahri-client

```
start /b ahri-client.exe -sip ip.ip.ip.ip ^
-sp 35473 ^
-k **** ^
-n B ^
-m 1 ^
-f ahri.hosts ^
-L 3 ^
-T 5 ^
>./a.log 2>&1 
```

关闭这个进程

```
tasklist | findstr ahri-client.exe
```
输出内容中第一个数，是该进程的 PID，例如是 3320 ，可以这样关闭它

```
taskkill /F /T /PID 3320
```

如果启动成功，可以看到 a.log 中的日志 `Ahri Client (0.9.3) is running.`
如果在 ahri-server 中注册成功，可以看到相关日志 `Get Client` 之类的。

如果你用的是 linux 或 MacBook， 那你可以直接修改脚本 `start.sh` 来使用 ahri-client。
`stop.sh` 帮助你关闭ta。

### A端的客户端

过程基本同上，这里 n 写 A，m 写 0

s5ip 与 s5p 是启动本地 socks5 的ip与端口，127.0.0.1 就是仅本机可用。如果你是 ipv6，自行修改。

```
start /b ahri-client.exe -sip ip.ip.ip.ip ^
-sp 35473 ^
-k **** ^
-n A ^
-m 0 ^
-s5ip 127.0.0.1 ^
-s5p 23456 ^
-f ahri.hosts ^
-L 3 ^
-T 5 ^
>./a.log 2>&1 
```

### 验证可用

上述过程都没问题后，在 A端 设置 socks5 网络代理即可。

**Mac 用户**
在【系统偏好设置】>【网络】>【高级】>【代理】>【socks代理】，填写 ip 与端口号即可，就是上述 s5ip，s5p 的值。
也可以直接将这些加入脚本，前提是你会用，且知道这些命令怎么用
```
# 开启 WIFI 网卡的 socks 代理。
networksetup -setsocksfirewallproxy Wi-Fi 127.0.0.1 23456 && networksetup -setsocksfirewallproxystate Wi-Fi on
# 关闭 WIFI 网卡的 socks 代理
networksetup -setsocksfirewallproxy Wi-Fi '' '' && networksetup -setsocksfirewallproxystate Wi-Fi off
```

**Win 用户**
出门 baidu，如何设置 socks5 网络代理。需要填写给 Windows 的关键信息仅有上述 s5ip，s5p 的值（ip与端口号）


由于 ahri.hosts 默认写入了 Google，YouTube。

一切没问题的话，A端 这台电脑应该可以使用 Google了。目标二达成。
也可以使用 B端 内网中的资源（ahri.hosts 需要加入 `xxx.com B`这样的行，xxx.com 是 B端 内网中的域名或ip）。目标一达成。


