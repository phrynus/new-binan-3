package main

var (
	logger *Logger
	err    error = nil
)

func init() {
	logger, err = NewLogger("main.log", 10*1024*1024) // 10MB分割
	if err != nil {
		panic(err)
	}
}
func main() {
	for i := 0; i < 1000000; i++ {
		logger.Info("GO")
	}
}
