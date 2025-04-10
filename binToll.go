package main

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/adshao/go-binance/v2/futures"
)

// processSymbolInfo 处理交易对信息，根据给定的价格和数量进行格式化处理。
// 参数:
//   - symbol: 交易对的符号，例如 "BTCUSDT"。
//   - p: 价格，浮点数。
//   - q: 数量，浮点数。
//
// 返回值:
//   - price: 格式化后的价格字符串。
//   - quantity: 格式化后的数量字符串。
//   - err: 错误信息，如果处理过程中出现错误则返回相应的错误。
func processSymbolInfo(symbol string, p float64, q float64) (price string, quantity string, err error) {
	var symbolInfo *futures.Symbol
	for _, s := range infoSymbols {
		if s.Symbol == symbol {
			symbolInfo = &s
		}
	}
	if symbolInfo == nil {
		return "", "", errors.New("symbolInfo is nil")
	}
	if q != 0 {
		quantity, err = takeDivisible(q, symbolInfo.Filters[1]["stepSize"].(string))
		if err != nil {
			return "", "", err
		}
	}
	if p != 0 {
		price, err = takeDivisible(p, symbolInfo.Filters[0]["tickSize"].(string))
		if err != nil {
			return "", "", err
		}
	}
	return price, quantity, nil
}

// takeDivisible 调整小数位数并确保可以整除。
// 参数:
//   - inputVal: 待处理的数字，浮点数。
//   - divisor: 被整除的值，文本型。
//
// 返回值:
//   - format: 格式化后的数值字符串。
//   - err: 错误信息，如果处理过程中出现错误则返回相应的错误。
func takeDivisible(inputVal float64, divisor string) (string, error) {
	divisorVal, err := strconv.ParseFloat(divisor, 64)
	if err != nil || divisorVal == 0 {
		return "", fmt.Errorf("无效的 divisor: %v", err)
	}
	decimalPlaces := 0
	if dot := strings.Index(divisor, "."); dot != -1 {
		decimalPlaces = len(divisor) - dot - 1
	}
	quotient := int(inputVal / divisorVal)
	maxDivisible := divisorVal * float64(quotient)
	format := fmt.Sprintf("%%.%df", decimalPlaces)
	return fmt.Sprintf(format, maxDivisible), nil
}
