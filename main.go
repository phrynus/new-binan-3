package main

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/adshao/go-binance/v2/futures"
)

var (
	client            *futures.Client
	httpClient        *http.Client
	listenKey         string
	infoSymbolsString []string
	infoSymbols       []futures.Symbol

	logger *Logger
	err    error = nil
)

func init() {

	// 初始化配置
	stdoutLevels := map[int]bool{
		0: false,
		1: false,
		2: true,
		3: true,
	}
	if config.Debug {
		stdoutLevels[0] = true
		stdoutLevels[1] = true
	}
	logger, err = LoggerNew(LogConfig{
		Filename:      "main.log",
		MaxSize:       10 * 1024 * 1024, // 10MB
		BufferSize:    1000,
		FlushInterval: 5 * time.Second,
		StdoutLevels:  stdoutLevels,
		ColorOutput:   true,
	})
	if err != nil {
		panic(err)
	}
	logger.T("\n\n\n---------PHRYNUS:GO---------\n")
	// 设置全局的 HTTP 客户端
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	if config.Proxy != "" {
		proxyURL, err := url.Parse(config.Proxy)
		if err != nil {
			log.Fatal(err)
		}
		transport.Proxy = http.ProxyURL(proxyURL)
		futures.SetWsProxyUrl(config.Proxy)
	}
	httpClient = &http.Client{Transport: transport}
	// 初始化 Binance Futures 客户端
	client = futures.NewClient(config.APIKey, config.SecretKey)
	client.HTTPClient = httpClient
	// 时间偏移
	_, err = client.NewSetServerTimeService().Do(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	// 获取交易信息
	info, err := client.NewExchangeInfoService().Do(context.Background())
	if err != nil {
		log.Fatal(err)
	}
	// 赛选币种
	for _, s := range info.Symbols {
		if s.QuoteAsset == "USDT" && s.ContractType == "PERPETUAL" && s.Status == "TRADING" {
			if !contains(config.Blacks, s.BaseAsset) {
				infoSymbols = append(infoSymbols, s)
				infoSymbolsString = append(infoSymbolsString, s.BaseAsset)
			}
		}
	}
	// logger.Info(infoSymbolsString)
}
func main() {
	defer logger.Close() // 关闭日志文件

	// go func() {
	// 	for {

	// 	}
	// }()

	// 监听 OS 信号，优雅退出
	sigC := make(chan os.Signal, 1)
	signal.Notify(sigC, os.Interrupt, syscall.SIGTERM)
	<-sigC
	logger.Error("正在关闭...")
}
