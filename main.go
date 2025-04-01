package main

import (
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
	// 常规日志
	logger.Info("System initialized")
	logger.Debug("Cache hits:", 2450, " misses:", 12)
	logger.Warnf("Connection attempt %d failed", 3)
	logger.Error("Database connection lost")

}
