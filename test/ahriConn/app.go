package main

import (
	"bytes"
	"crypto/aes"
	"fmt"
	"github.com/GavinGuan24/ahri/core"
	"time"
)

var bytes0 = []byte(`
One Shot 《一战成名》
林俊杰 🐶🐣🏴‍☠️🙉🤣🙈😁🇨🇳🎉😂👨‍👩‍👧‍👧👪💏🧛‍♂️🐛👜✅👌🚧

Just one shot 就这一次机会
Is all you got to make or break into your fame 让你一战成名
Just one shot 就这一次机会
They will lift you up and take you down the same 他们会把你高高捧起同时又狠狠踩在脚下
Only got one shot 只有这一次机会
Will you give it all you've got Let it take you to the top Or will you bleed it out in vain?
你会奉献出你的一切好让自己站在巅峰或者徒劳地被榨尽血液么？
Only got one shot 只有这一次机会
Tell your heart to never stop 说服自己的内心永不停步
Lock your eyes only on the spot 成为众矢之的时请紧闭你的双眼
If there is no pain there will be no gain 如果没有痛苦便不会收获
O Father please, I ask for Your mercy 哦 主啊 我祈求你的怜悯
They judge me way before they even know me 他们在对我一无所知的情况下就对我进行如此的宣判
O Father please, do you hear me screaming 哦 天上的父 您可否听到我内心的尖啸
These lights won't stop chasing after me 那些刺目的聚光灯不会停止追赶我
I will not fall I will not crawl, I will keep on standing tall
我不会被击倒更不会爬在地上向他们谄媚，我会继续昂首前进
Til I'm strong enough to break this wall直到我足够强壮去打破这道围墙
Oh~ I won't drown don't tie me down哦 我不会沉溺而亡不要给我套上枷锁
Won't you just set me free您能否放我自由？
It takes one shot to it strip away from me只需一次机会就能让我脱掉枷锁
And O my Lord, I can't believe what I saw哦 我的主 我不能相信我所看到的
You got one shot你有一次机会
Just to make or break your fame让你一战成名
Just one shot 就这一次机会
They will lift you up and take you down the same 他们会把你高高捧起同时又狠狠踩在脚下
Only got one shot 只有这一次机会
Will you give it all you've got Let it take you to the top Or will you bleed it out in vain?
你会奉献出你的一切好让自己站在巅峰或者徒劳地被榨尽血液么？
Only got one shot 只有这一次机会
Tell your heart to never stop 说服自己的内心永不停步
Lock your eyes only on the spot 成为众矢之的请紧闭你的双眼
If there is no pain there will be no gain 如果没有痛苦便不会收获
Our father, who art in heaven 我们天上的父,
Hallowed be thy name 愿您的名受显扬
In thy name I pray you, to oh forever it remains 为您的名祷告，从现在到永远。
I know you haven't left me but I'm feeling so alone 我知道你未曾离开我，但我却倍感孤独。
When the darkness comes I shall never have to wait on my own 当黑暗降临我当永远不会独一己之力
I feel so lost, my mind is going out of control我感到十分迷茫，我的心思即将失控
Vengeance is mine, I will repay, says The Lord我若复仇，我将偿还，主这样说道。
I'm persecuted but not forsaken, struck down to the floorBut not destroyed cause Its not like this hasn't
happened before我感到烦扰但并未未被遗忘或者被击倒在地，亦或者被摧毁，因为这又不是从没发生过。
I see these lights and you've warned me be to on the alert我看到了亮光，你曾警告我要随时提防
My heart is tired from soaking it up like rain in the dirt
我的心感到十分劳累，就像雨水落入泥土被迅速吸收一样
Oh~ just one shot 哦 就这一次机会
Oh~ yeah~Yeah~ one shot 是的 一次机会
They will lift you up and take you down the same 他们会把你高高捧起同时又狠狠踩在脚下
Only got one shot 只有这一次机会
Will you give it all you've got Let it take you to the top Or will you bleed it out in vain?
你会奉献出你的一切好让自己站在巅峰或者徒劳地被榨尽血液么？
Only got one shot 只有这一次机会
Tell your heart to never stop 说服自己的内心永不停步
Lock your eyes only on the spot 成为众矢之的时请紧闭你的双眼
If there is no pain there will be no gain 如果没有痛苦便不会收获
`)

func main() {

	key := core.GenerateAes256Key()
	aesCipher, _ := aes.NewCipher(key[:])
	connReceiver := make(chan core.AhriFrame, len(bytes0)/(core.AfpPayloadMaxLen-aes.BlockSize)+1)
	sender := func(frame core.AhriFrame) error {
		webFrame := make([]byte, len(frame))
		copy(webFrame, frame)
		connReceiver <- webFrame
		return nil
	}

	close1 := func(conn *core.AhriConn) {
		fmt.Printf("close the conn")
	}

	conn1 := core.NewAhriConnForVirtualization(
		"fr",
		"to",
		0,
		aesCipher,
		core.AfpFrameTypeDirect,
		connReceiver,
		sender,
		close1)

	for i := 0; i < len(bytes0); {
		n, _ := conn1.Write(bytes0[i:])
		i += n
	}

	go func() {
		time.Sleep(time.Millisecond)
		close(connReceiver)
	}()

	bytes1 := core.ByteArrPool.Get()
	var buf bytes.Buffer
	for {

		n, e := conn1.Read(bytes1)
		if e != nil {
			break
		}
		buf.Write(bytes1[:n])
	}
	fmt.Printf("%s\n", buf.String())

}
