package main

// contains 判断目标字符串是否存在于字符串切片中
// 参数:
//   - elements: 字符串切片，用于存储待检查的字符串元素
//   - target: 目标字符串，需要判断是否存在于 elements 中
// 返回值:
//   - bool: 如果 target 存在于 elements 中，返回 true；否则返回 false
func contains(elements []string, target string) bool {
	for _, elem := range elements {
		if elem == target {
			return true
		}
	}
	return false
}
