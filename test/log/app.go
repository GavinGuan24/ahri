package log

import (
	"fmt"
	"github.com/GavinGuan24/ahri/core"
	"time"
)

var Log core.Logger = &core.Alog{core.LevelError}

func main() {
	fmt.Println("========= 开始日志测试 =========")
	Log.NoLevelf("写点 * 信息 %09d", 0x24)
	Log.Debug("写点debug信息")
	Log.Debugf("写点debug信息 %09d", 0xFF)
	time.Sleep(time.Millisecond * 10)
	Log.Info("写点info信息")
	time.Sleep(time.Millisecond * 10)
	Log.Warn("写点warn信息")
	time.Sleep(time.Millisecond * 10)
	Log.Error("写点error信息")
	time.Sleep(time.Millisecond * 10)
	Log.Crash("写点crash信息")
	time.Sleep(time.Millisecond * 10)
}
