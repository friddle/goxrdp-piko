package glog

import (
	"fmt"
	"log"
	"os"
	"sync"
)

var (
	logger *log.Logger
	level  LEVEL
	mu     sync.Mutex
)

type LEVEL int

const (
	TRACE LEVEL = iota
	DEBUG
	INFO
	WARN
	ERROR
	NONE
)

// InitStdout 初始化glog输出到stdout
func InitStdout(l LEVEL) {
	SetLevel(l)

	// 创建一个配置好的logger，输出到stdout
	logger := log.New(os.Stdout, "", log.LstdFlags|log.Lshortfile)
	SetLogger(logger)

	// 输出初始化信息
	Info("glog已配置到stdout")
}

// InitFile 初始化glog输出到文件
func InitFile(l LEVEL, filename string) error {
	file, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("无法打开日志文件: %v", err)
	}

	SetLevel(l)

	// 创建一个配置好的logger，输出到文件
	logger := log.New(file, "", log.LstdFlags|log.Lshortfile)
	SetLogger(logger)

	// 输出初始化信息
	Info("glog已配置到文件:", filename)
	return nil
}

func SetLogger(l *log.Logger) {
	l.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)
	logger = l
}

func SetLevel(l LEVEL) {
	level = l
}

func checkLogger() {
	if logger == nil && level != NONE {
		panic("logger not inited")
	}
}
func Trace(v ...interface{}) {
	checkLogger()
	if level <= TRACE {
		mu.Lock()
		defer mu.Unlock()
		logger.SetPrefix("[TRACE]")
		logger.Output(2, fmt.Sprintln(v...))
	}
}
func Tracef(f string, v ...interface{}) {
	checkLogger()
	if level <= TRACE {
		mu.Lock()
		defer mu.Unlock()
		logger.SetPrefix("[TRACE]")
		logger.Output(2, fmt.Sprintln(fmt.Sprintf(f, v...)))
	}
}
func Debug(v ...interface{}) {
	checkLogger()
	if level <= DEBUG {
		mu.Lock()
		defer mu.Unlock()
		logger.SetPrefix("[DEBUG]")
		logger.Output(2, fmt.Sprintln(v...))
	}
}
func Debugf(f string, v ...interface{}) {
	checkLogger()
	if level <= DEBUG {
		mu.Lock()
		defer mu.Unlock()
		logger.SetPrefix("[DEBUG]")
		logger.Output(2, fmt.Sprintln(fmt.Sprintf(f, v...)))
	}
}
func Info(v ...interface{}) {
	checkLogger()
	if level <= INFO {
		mu.Lock()
		defer mu.Unlock()
		logger.SetPrefix("[INFO]")
		logger.Output(2, fmt.Sprintln(v...))
	}
}
func Infof(f string, v ...interface{}) {
	checkLogger()
	if level <= INFO {
		mu.Lock()
		defer mu.Unlock()
		logger.SetPrefix("[INFO]")
		logger.Output(2, fmt.Sprintln(fmt.Sprintf(f, v...)))
	}
}
func Warn(v ...interface{}) {
	checkLogger()
	if level <= WARN {
		mu.Lock()
		defer mu.Unlock()
		logger.SetPrefix("[WARN]")
		logger.Output(2, fmt.Sprintln(v...))
	}
}
func Warnf(f string, v ...interface{}) {
	checkLogger()
	if level <= WARN {
		mu.Lock()
		defer mu.Unlock()
		logger.SetPrefix("[WARN]")
		logger.Output(2, fmt.Sprintln(fmt.Sprintf(f, v...)))
	}
}
func Error(v ...interface{}) {
	checkLogger()
	if level <= ERROR {
		mu.Lock()
		defer mu.Unlock()
		logger.SetPrefix("[ERROR]")
		logger.Output(2, fmt.Sprintln(v...))
	}
}
func Errorf(f string, v ...interface{}) {
	checkLogger()
	if level <= ERROR {
		mu.Lock()
		defer mu.Unlock()
		logger.SetPrefix("[ERROR]")
		logger.Output(2, fmt.Sprintln(fmt.Sprintf(f, v...)))
	}
}
