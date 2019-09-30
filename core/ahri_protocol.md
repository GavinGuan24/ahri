# Ahri Protocol

下面的两个协议组成了 Ahri Protocol. 
Ahri Protocol 是基于 TCP 协议实现的.

## 1. Ahri Registe Protocol

Ahri Registe Protocol (ARP) 是一个 Ahri Client 与 Ahri Server 之间的通信协议. 用于 Ahri Client 向 Ahri Server 注册自己.

ARP 一共有5个阶段.

### 1.1

client 发起 TCP 连接请求.

### 1.2

server 回传 rsa public key (byte数据格式).

### 1.3 registe request

client 拼接 registe request (byte 数组), 使用 rsa public key 加密后发送给 server.

|伪码函数|作用|
|:--|:--|
|len(x)|获取 x 的长度|
|byteArr(x)|获取 x 的 byte 数组|


需要准备的参数:

|参数名|意义|
|:--|:--|
|serverPassword|服务端的密码|
|name|客户端将注册的名字|
|mode|客户端将支持的运行模式|
|aesKey|客户端将使用的aes密码|

**registe request** (byte数组) 格式为

|内容|长度|
|:--|:--|
|len(serverPassword)|1|
|byteArr(serverPassword)|动态|
|len(name)|1|
|byteArr(name)|动态|
|byteArr(mode)|1|
|len(aesKey)|1|
|byteArr(aesKey)|动态|

### 1.4 registe ack

server 解析 registe request 之后, 返回 **registe ack** 给 client.

**registe ack** (byte数组) 格式为

|内容|长度|
|:--|:--|
|ackCode|1|

ackCode

- 0x00 注册通过
- 0x01 验证密码有误
- 0x02 客户端名称已被注册
- 0x03 不理解的客户端模式
- 0x04 非法的AES密码

### 1.5

上述过程后, 若注册成功, 即可按 Ahri Frame Protocol(AFP) 通信.

## 2. Ahri Frame Protocol

Ahri Frame Protocol (AFP) 用于协调 Ahri Client 与 Ahri Server 之间的通信.

因为这里的模型已经简化为

- Ahri Client: 发起或处理(广义上的)请求.
- Ahri Server: 处理或转发请求. 

他们之间交互数据的传递流程与上述类似. 且对单个 TCP 连接经行多路复用.

所以在此引入这个概念 **Ahri Frame**. 对每个(虚拟的)连接的数据切片后进行传输.



### 2.1 AFP 如何运作

#### 2.1.1 连接虚拟化, 数据帧化

Ahri Client 与 Ahri Server 在 ARP 之后就有了一个可靠的 TCP 连接. 而想做到多路复用(复用这个 TCP 连接), 必须把应用层的连接(以下简称 "应用连接")虚拟化.

以 HTTP 1.1 为例, 一个 HTTP 请求在 TCP 连接准备就绪之后, 会独占这个 TCP 连接, 且响应时间是不确定的.

为了避免这个问题, 我们需要将多个 "应用连接" 的内容切片为一个个的帧, 然后在 Ahri Client 与 Ahri Server 之间的这个已经建立起来的 TCP 连接中传递. 这个想法借鉴于 HTTP 2.

#### 2.1.2 heartbeat

因为 ARP 不涉及 keep alive, 但是我希望这个 TCP 连接可以一直存活, 所以需要引入一个特殊的帧, 即 心跳帧. 用来告知 TCP 连接的对方, 我方还保持着连接. 否则关闭 TCP 连接.

**frame type** 的第一个类型 heartbeat (值为 0x00) 便是心跳帧标识.

#### 2.1.3 "应用连接" 在适当的时候被持有者主动关闭

我希望 "应用连接" 在首次交互后保留一段时间, 若后续无交互的时长达到一个预定值(例如: 3秒), 则由 "应用连接" 的持有者自行关闭该连接. 

#### 2.1.4 Ahri Frame

##### 2.1.4.1 格式

|-|protocol flag|frame type|from|to|conn No|payload len|payload|
|:--:|:--:|:--:|:--:|:--:|:--:|:--:|:--:|
|type / value|0x24|uint8|string|string|uint64|uint16|[ ]byte|
|byte len|1|1|2|2|8|2|variable<=2032|


- protocol flag: AFP header 的标识.
- frame type: 帧类型, 转发时变该值为对应的 proxy 类型
    - 0x00: heartbeat
    - 0x01: direct
    - 0x02: proxy
    - 0x03: dial
    - 0x04: dial ack
    - 0x05: dial proxy
    - 0x06: dial proxy ack
- from: 帧的来源
- to: 帧的终点
- conn ID: 请求的唯一ID, 由发起者生成, 转发时不改变该值
- payload len: 负载长度
- payload: 负载, AFP 约定帧最大为 AfpFrameMaxLen bytes, 头部占用 AfpHeaderLen bytes, 所以这里至多 AfpFrameMaxLen - AfpHeaderLen bytes

##### 2.1.4.2 内容说明

**Ahri Frame** 的头部 (payload 以外的部分) 仅使用 AfpHeaderLen 个字节. 

**protocol flag** 与 **payload len** 保证实现者能够成功的从数据流中分辨出一个个的 Ahri Frame.

|frame type|value|mean|
|:--|:--|:--|
|heartbeat|0x00|心跳帧|
|dial|0x03|client 向 server 发起一个 "应用连接"|
|dial ack|0x04|server 回应 client 发起一个 "应用连接" 的结果|
|direct|0x01|client 与 server 建立一个 "应用连接"后, 双方间的信息帧类型为 direct|
|dial proxy|0x05|转发解析的目标为另一个 client B 时, server 将发起者 (client A) 的请求 (发起一个 "应用连接")  转发给 B 的信息帧类型|
|dial proxy ack|0x06|上一列的情形中, B 回应 A 的信息将要求 server 经行转发时, 该信息帧的类型|
|proxy|0x02|上列的情形中, A 与 B 建立一个 "应用连接"后, 数据在 server 与 B 之间传递的信息帧类型|

图解:

请求由 server 直接处理时:

```sh
A -> S               dial
A <- S('S')          dial ack
------------------------------
A -> S               direct
A <- S('S')          direct
```

请求由另一个 client B 处理时:

```sh
A -> S('B')          dial
     S('B') -> B     dial proxy
     S('B') <- B     dial proxy ack
A <- S('B')          dial ack
--------------------------------
A -> S('B')          direct
     S('B') -> B     proxy
     S('B') <- B     proxy
A <- S('B')          direct
```

**from**, **to** 是 1 到 2 个英文字符组成的名字, 作为 Ahri Client Name. 'S', 'L' 为保留名, 禁止使用.

**conn ID** 是由请求的发起者(一个 client)生成的唯一的ID, 用于标识连接, uint64 保证在使用中不会重复(起码用到你生命的终点🤣)


**payload len** 就是说明后面的 **payload** 长度的.

**payload** 就是 "应用连接" 传输的内容, AFP 约定 AF 最大 AfpFrameMaxLen bytes, 头 AfpHeaderLen bytes, 所以 **payload** 最大 AfpFrameMaxLen - AfpHeaderLen bytes.


##### 2.1.4.3 特殊的帧

###### 心跳

heartbeat: payload 为一个 0x00 字节

###### dial

dial: payload 按 socks5 请求的格式填写这三个信息 ATYP DST.ADDR DST.PORT
dial ack: payload 为一个字节, 0x00 表示应答成功, 0x01 表示应答失败

若一定时间段内无响应, 发起者主动切断连接.

###### dial proxy

此类帧的出现是因为 AhriServer 不是直接的相应者, 所以转发 AhriFrame 至对应的 AhriClient(响应者). 不同点仅在于 AhriFrame 的 header 中的 type 为对应的 proxy 类型.

dial proxy: payload 格式与 dial 完全一致
dial proxy ack: payload 格式与 dial proxy 完全一致


