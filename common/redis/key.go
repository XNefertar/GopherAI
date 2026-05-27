package redis

import (
	"GopherAI/config"
	"fmt"
)

// key:特定邮箱-> 验证码
func GenerateCaptcha(email string) string {
	return fmt.Sprintf(config.DefaultRedisKeyConfig.CaptchaPrefix, email)
}

func GenerateIndexName(kbID string) string {
	indexName := fmt.Sprintf(config.DefaultRedisKeyConfig.IndexName, kbID)
	return indexName
}

func GenerateIndexNamePrefix(kbID string) string {
	prefix := fmt.Sprintf(config.DefaultRedisKeyConfig.IndexNamePrefix, kbID)
	return prefix
}
