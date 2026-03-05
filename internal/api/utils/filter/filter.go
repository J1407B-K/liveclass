package filter

import (
	"regexp"
	"strings"
)

// 敏感词列表
var sensitiveWords = []string{"毒品", "赌博", "违法"}

// 过滤相关
func FilterSensitiveWords(content string) string {
	for _, word := range sensitiveWords {
		re := regexp.MustCompile(regexp.QuoteMeta(word))
		content = re.ReplaceAllString(content, strings.Repeat("*", len(word)))
	}
	return content
}

func CleanMessage(content string) string {
	re := regexp.MustCompile(`[^a-zA-Z0-9一-龥\s]`)
	return re.ReplaceAllString(content, "")
}
