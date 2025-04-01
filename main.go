package main

var (
	logger *Logger
	err    error = nil
)

func init() {
	logger, err = NewLogger("main.log", 1*1024*1024) // 10MB分割
	if err != nil {
		panic(err)
	}
}
func main() {
	// 使用示例
	logger.Info("This is an info message")
	logger.Debug("Debugging information")
	logger.Warn("Warning message")
	for i := 0; i < 1000000; i++ {
		logger.Debug("message")

	}
}
