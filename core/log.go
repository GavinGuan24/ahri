package core

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"
)

type LogLevel int

func (level LogLevel) String() string {
	if LevelDebug == level {
		return "Debug"
	}
	if LevelInfo == level {
		return "Info"
	}
	if LevelWarn == level {
		return "Warn"
	}
	if LevelError == level {
		return "Error"
	}
	if LevelCrash == level {
		return "Crash"
	}
	return "*"
}

type Logger interface {
	Deny(LogLevel) bool

	NoLevel(log string)
	NoLevelf(format string, args ...interface{})

	Debug(log string)
	Info(log string)
	Warn(log string)
	Error(log string)
	Crash(log string)

	Debugf(format string, args ...interface{})
	Infof(format string, args ...interface{})
	Warnf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Crashf(format string, args ...interface{})
}

const (
	LevelDebug LogLevel = iota
	LevelInfo
	LevelWarn
	LevelError
	LevelCrash
	NoLevel
)

var stdout = log.New(os.Stderr, "", log.Ldate|log.Lmicroseconds)

type Alog struct {
	LowLevel LogLevel
}

func (a *Alog) NoLevel(log string) {
	a.logout(NoLevel, log)
}

func (a *Alog) NoLevelf(format string, args ...interface{}) {
	a.logout(NoLevel, fmt.Sprintf(format, args...))
}

func (a *Alog) Debug(log string) {
	if a.Deny(LevelDebug) {
		return
	}
	a.logout(LevelDebug, log)
}

func (a *Alog) Info(log string) {
	if a.Deny(LevelInfo) {
		return
	}
	a.logout(LevelInfo, log)
}

func (a *Alog) Warn(log string) {
	if a.Deny(LevelWarn) {
		return
	}
	a.logout(LevelWarn, log)
}

func (a *Alog) Error(log string) {
	if a.Deny(LevelError) {
		return
	}
	a.logout(LevelError, log)
}

func (a *Alog) Crash(log string) {
	a.logout(LevelCrash, log)
}

func (a *Alog) Debugf(format string, args ...interface{}) {
	if a.Deny(LevelDebug) {
		return
	}
	a.logout(LevelDebug, fmt.Sprintf(format, args...))
}

func (a *Alog) Infof(format string, args ...interface{}) {
	if a.Deny(LevelInfo) {
		return
	}
	a.logout(LevelInfo, fmt.Sprintf(format, args...))
}

func (a *Alog) Warnf(format string, args ...interface{}) {
	if a.Deny(LevelWarn) {
		return
	}
	a.logout(LevelWarn, fmt.Sprintf(format, args...))
}

func (a *Alog) Errorf(format string, args ...interface{}) {
	if a.Deny(LevelError) {
		return
	}
	a.logout(LevelError, fmt.Sprintf(format, args...))
}

func (a *Alog) Crashf(format string, args ...interface{}) {
	a.logout(LevelCrash, fmt.Sprintf(format, args...))
}

func (a *Alog) logout(level LogLevel, msg string) {
	isStderr := LevelError == level || LevelCrash == level
	var buf bytes.Buffer
	buf.WriteString(time.Now().Format("2006-01-02 15:04:05.000 "))
	buf.WriteString(level.String())
	buf.WriteString(" - ")
	buf.WriteString(msg)

	if isStderr {
		stackBytes := debug.Stack()
		newlineFlag := 0
		for i := 0; i < len(stackBytes); i++ {
			if stackBytes[i] == '\n' {
				newlineFlag++
			}
			if newlineFlag == 7 {
				stackBytes = stackBytes[i+1:]
			}
		}
		buf.WriteByte('\n')
		buf.Write(stackBytes)
		fmt.Fprintln(os.Stderr, buf.String())
		if LevelCrash == level {
			os.Exit(1)
		}
	} else {
		fmt.Fprintln(os.Stdout, buf.String())
	}
}

func (a *Alog) Deny(level LogLevel) bool {
	return level < a.LowLevel
}
