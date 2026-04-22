package redis

import (
	"GopherAI/config"
	"GopherAI/model"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	redisCli "github.com/redis/go-redis/v9"
)

var Rdb *redisCli.Client

const defaultRedisTimeout = 3 * time.Second
const defaultUserCacheTTL = 10 * time.Minute

type cachedUser struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func getRedisTimeout() time.Duration {
	timeoutMs := config.GetConfig().RedisConfig.RedisTimeoutMs
	if timeoutMs <= 0 {
		return defaultRedisTimeout
	}
	return time.Duration(timeoutMs) * time.Millisecond
}

func withOperationTimeout(parent context.Context) (context.Context, context.CancelFunc) {
	if parent == nil {
		parent = context.Background()
	}
	return context.WithTimeout(parent, getRedisTimeout())
}

func Init() {
	conf := config.GetConfig()
	host := conf.RedisConfig.RedisHost
	port := conf.RedisConfig.RedisPort
	password := conf.RedisConfig.RedisPassword
	db := conf.RedisDb
	addr := host + ":" + strconv.Itoa(port)
	timeout := getRedisTimeout()

	Rdb = redisCli.NewClient(&redisCli.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		Protocol:     2, // 使用 Protocol 2 避免 maint_notifications 警告
		DialTimeout:  timeout,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
	})

}

func GetUserByUsername(ctx context.Context, username string) (*model.User, bool, error) {
	if Rdb == nil {
		return nil, false, nil
	}

	opCtx, cancel := withOperationTimeout(ctx)
	defer cancel()

	cacheValue, err := Rdb.Get(opCtx, GenerateUserCacheKey(username)).Bytes()
	if err != nil {
		if err == redisCli.Nil {
			return nil, false, nil
		}
		return nil, false, err
	}

	var cached cachedUser
	if err := json.Unmarshal(cacheValue, &cached); err != nil {
		return nil, false, err
	}

	return &model.User{
		ID:        cached.ID,
		Name:      cached.Name,
		Email:     cached.Email,
		Username:  cached.Username,
		Password:  cached.Password,
		CreatedAt: cached.CreatedAt,
		UpdatedAt: cached.UpdatedAt,
	}, true, nil
}

func CacheUserByUsername(ctx context.Context, user *model.User) error {
	if Rdb == nil || user == nil {
		return nil
	}

	opCtx, cancel := withOperationTimeout(ctx)
	defer cancel()

	payload, err := json.Marshal(cachedUser{
		ID:        user.ID,
		Name:      user.Name,
		Email:     user.Email,
		Username:  user.Username,
		Password:  user.Password,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	})
	if err != nil {
		return err
	}

	return Rdb.Set(opCtx, GenerateUserCacheKey(user.Username), payload, defaultUserCacheTTL).Err()
}

func DeleteUserCacheByUsername(ctx context.Context, username string) error {
	if Rdb == nil {
		return nil
	}

	opCtx, cancel := withOperationTimeout(ctx)
	defer cancel()

	return Rdb.Del(opCtx, GenerateUserCacheKey(username)).Err()
}

func SetCaptchaForEmail(ctx context.Context, email, captcha string) error {
	opCtx, cancel := withOperationTimeout(ctx)
	defer cancel()

	key := GenerateCaptcha(email)
	expire := 2 * time.Minute
	return Rdb.Set(opCtx, key, captcha, expire).Err()
}

func CheckCaptchaForEmail(ctx context.Context, email, userInput string) (bool, error) {
	opCtx, cancel := withOperationTimeout(ctx)
	defer cancel()

	key := GenerateCaptcha(email)

	storedCaptcha, err := Rdb.Get(opCtx, key).Result()
	if err != nil {
		if err == redisCli.Nil {

			return false, nil
		}

		return false, err
	}

	if strings.EqualFold(storedCaptcha, userInput) {

		// 验证成功后删除 key
		if err := Rdb.Del(opCtx, key).Err(); err != nil {

		} else {

		}
		return true, nil
	}

	return false, nil
}

// InitRedisIndex 初始化 Redis 索引，支持按文件名区分
func InitRedisIndex(ctx context.Context, filename string, dimension int) error {
	indexName := GenerateIndexName(filename)

	// 检查索引是否存在
	_, err := Rdb.Do(ctx, "FT.INFO", indexName).Result()
	if err == nil {
		fmt.Println("索引已存在，跳过创建")
		return nil
	}

	// 如果索引不存在，创建新索引
	if !strings.Contains(err.Error(), "Unknown index name") {
		return fmt.Errorf("检查索引失败: %w", err)
	}

	fmt.Println("正在创建 Redis 索引...")

	prefix := GenerateIndexNamePrefix(filename)

	// 创建索引
	createArgs := []interface{}{
		"FT.CREATE", indexName,
		"ON", "HASH",
		"PREFIX", "1", prefix,
		"SCHEMA",
		"content", "TEXT",
		"metadata", "TEXT",
		"vector", "VECTOR", "FLAT",
		"6",
		"TYPE", "FLOAT32",
		"DIM", dimension,
		"DISTANCE_METRIC", "COSINE",
	}

	if err := Rdb.Do(ctx, createArgs...).Err(); err != nil {
		return fmt.Errorf("创建索引失败: %w", err)
	}

	fmt.Println("索引创建成功！")
	return nil
}

// DeleteRedisIndex 删除 Redis 索引，支持按文件名区分
func DeleteRedisIndex(ctx context.Context, filename string) error {
	indexName := GenerateIndexName(filename)

	// 删除索引
	if err := Rdb.Do(ctx, "FT.DROPINDEX", indexName).Err(); err != nil {
		return fmt.Errorf("删除索引失败: %w", err)
	}

	fmt.Println("索引删除成功！")
	return nil
}
