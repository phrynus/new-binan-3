package main

import (
	"os"
	"os/signal"
	"syscall"
	"time"
)

var (
	logger *Logger
	err    error = nil
)

func init() {
	logger, err = LoggerNew(LogConfig{
		Filename:      "main.log",
		MaxSize:       10 * 1024 * 1024, // 10MB
		BufferSize:    1000,
		FlushInterval: 5 * time.Second,
		StdoutLevels: map[int]bool{
			0: true,
			1: true,
			2: true,
			3: true,
		},
	})
	if err != nil {
		panic(err)
	}
}
func main() {
	defer logger.Close() // 关闭日志文件
	

	// 监听 OS 信号，优雅退出
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)
	<-sigC
	logger.Error("正在关闭...")
}
