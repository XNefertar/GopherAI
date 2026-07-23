package config

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type MainConfig struct {
	Port    int    `toml:"port"`
	AppName string `toml:"appName"`
	Host    string `toml:"host"`
}

type EmailConfig struct {
	Authcode string `toml:"authcode"`
	Email    string `toml:"email" `
}

type RedisConfig struct {
	RedisPort      int    `toml:"port"`
	RedisDb        int    `toml:"db"`
	RedisHost      string `toml:"host"`
	RedisPassword  string `toml:"password"`
	RedisTimeoutMs int    `toml:"timeoutMs"`
}

type MysqlConfig struct {
	MysqlPort         int    `toml:"port"`
	MysqlHost         string `toml:"host"`
	MysqlUser         string `toml:"user"`
	MysqlPassword     string `toml:"password"`
	MysqlDatabaseName string `toml:"databaseName"`
	MysqlCharset      string `toml:"charset"`
}

type JwtConfig struct {
	ExpireDuration int    `toml:"expire_duration"`
	Issuer         string `toml:"issuer"`
	Subject        string `toml:"subject"`
	Key            string `toml:"key"`
}

type Rabbitmq struct {
	RabbitmqPort     int    `toml:"port"`
	RabbitmqHost     string `toml:"host"`
	RabbitmqUsername string `toml:"username"`
	RabbitmqPassword string `toml:"password"`
	RabbitmqVhost    string `toml:"vhost"`
}

type RagModelConfig struct {
	RagEmbeddingModel   string `toml:"embeddingModel"`
	RagEmbeddingBaseURL string `toml:"embeddingBaseUrl"`
	RagEmbeddingAPIType string `toml:"embeddingApiType"`
	RagDocDir           string `toml:"docDir"`
	RagDimension        int    `toml:"dimension"`
	RagChunkSize        int    `toml:"chunkSize"`
	RagChunkOverlap     int    `toml:"chunkOverlap"`
	RagTopK             int    `toml:"topK"`
}

type VoiceServiceConfig struct {
	VoiceServiceApiKey    string `toml:"voiceServiceApiKey"`
	VoiceServiceSecretKey string `toml:"voiceServiceSecretKey"`
}

// SessionCacheConfig 会话内存缓存的容量治理配置（Phase 2 引入）。
type SessionCacheConfig struct {
	MaxSessions    int `toml:"maxSessions"`    // 内存中最多保留的会话数，超过则按 LRU 淘汰
	IdleTimeoutSec int `toml:"idleTimeoutSec"` // 会话空闲超过该秒数则被后台回收
}

// RouterConfig 混合路由器配置（L1 Embedding + L2 LLM + L3 规则）。
// 所有参数均为可选：未配置时使用代码内置默认值，不会影响服务可用性。
// 布尔字段使用指针类型，以便区分"未配置（nil）"和"显式设为 false"。
type RouterConfig struct {
	// --- L1: Embedding 快速匹配 ---
	EmbeddingEnabled   *bool   `toml:"embeddingEnabled"`   // 是否启用 L1（默认 true）
	EmbeddingThreshold float64 `toml:"embeddingThreshold"` // 相似度阈值（默认 0.85）
	EmbeddingMargin    float64 `toml:"embeddingMargin"`    // 意图间最小裕度（默认 0.08）
	EmbeddingTimeoutMs int     `toml:"embeddingTimeoutMs"` // Embedding API 超时毫秒（默认 500）

	// --- L2: LLM 语义分类 ---
	LLMClassifierEnabled   *bool   `toml:"llmClassifierEnabled"`   // 是否启用 L2（默认 true）
	LLMClassifierTimeoutMs int     `toml:"llmClassifierTimeoutMs"` // LLM 分类超时毫秒（默认 800）
	LLMConfidenceThreshold float64 `toml:"llmConfidenceThreshold"`  // 置信度阈值（默认 0.55）
	LLMRewriteEnabled      *bool   `toml:"llmRewriteEnabled"`       // Query 改写开关（默认 true）
}

// boolPtr 便捷工厂：返回一个 bool 指针。
func boolPtr(v bool) *bool { return &v }

// RouterDefaults 路由器的所有默认值，收口在一处便于维护。
var RouterDefaults = RouterConfig{
	EmbeddingEnabled:       boolPtr(true),
	EmbeddingThreshold:     0.85,
	EmbeddingMargin:        0.08,
	EmbeddingTimeoutMs:     500,
	LLMClassifierEnabled:   boolPtr(true),
	LLMClassifierTimeoutMs: 800,
	LLMConfidenceThreshold: 0.55,
	LLMRewriteEnabled:      boolPtr(true),
}

// EmbeddingEnabledOrDefault 返回 L1 是否启用（带默认值）。
func (c RouterConfig) EmbeddingEnabledOrDefault() bool {
	if c.EmbeddingEnabled != nil {
		return *c.EmbeddingEnabled
	}
	return *RouterDefaults.EmbeddingEnabled
}

// EmbeddingThresholdOrDefault 返回 L1 相似度阈值（带默认值）。
func (c RouterConfig) EmbeddingThresholdOrDefault() float64 {
	if c.EmbeddingThreshold > 0 {
		return c.EmbeddingThreshold
	}
	return RouterDefaults.EmbeddingThreshold
}

// EmbeddingMarginOrDefault 返回 L1 意图裕度（带默认值）。
func (c RouterConfig) EmbeddingMarginOrDefault() float64 {
	if c.EmbeddingMargin > 0 {
		return c.EmbeddingMargin
	}
	return RouterDefaults.EmbeddingMargin
}

// EmbeddingTimeoutOrDefaultMs 返回 L1 Embedding 超时毫秒数（带默认值）。
func (c RouterConfig) EmbeddingTimeoutOrDefaultMs() int {
	if c.EmbeddingTimeoutMs > 0 {
		return c.EmbeddingTimeoutMs
	}
	return RouterDefaults.EmbeddingTimeoutMs
}

// LLMClassifierEnabledOrDefault 返回 L2 是否启用（带默认值）。
func (c RouterConfig) LLMClassifierEnabledOrDefault() bool {
	if c.LLMClassifierEnabled != nil {
		return *c.LLMClassifierEnabled
	}
	return *RouterDefaults.LLMClassifierEnabled
}

// LLMClassifierTimeoutOrDefaultMs 返回 L2 LLM 分类超时毫秒数（带默认值）。
func (c RouterConfig) LLMClassifierTimeoutOrDefaultMs() int {
	if c.LLMClassifierTimeoutMs > 0 {
		return c.LLMClassifierTimeoutMs
	}
	return RouterDefaults.LLMClassifierTimeoutMs
}

// LLMConfidenceThresholdOrDefault 返回 L2 置信度阈值（带默认值）。
func (c RouterConfig) LLMConfidenceThresholdOrDefault() float64 {
	if c.LLMConfidenceThreshold > 0 {
		return c.LLMConfidenceThreshold
	}
	return RouterDefaults.LLMConfidenceThreshold
}

// LLMRewriteEnabledOrDefault 返回 Query 改写开关（带默认值）。
func (c RouterConfig) LLMRewriteEnabledOrDefault() bool {
	if c.LLMRewriteEnabled != nil {
		return *c.LLMRewriteEnabled
	}
	return *RouterDefaults.LLMRewriteEnabled
}

type Config struct {
	EmailConfig        `toml:"emailConfig"`
	RedisConfig        `toml:"redisConfig"`
	MysqlConfig        `toml:"mysqlConfig"`
	JwtConfig          `toml:"jwtConfig"`
	MainConfig         `toml:"mainConfig"`
	Rabbitmq           `toml:"rabbitmqConfig"`
	RagModelConfig     `toml:"ragModelConfig"`
	VoiceServiceConfig `toml:"voiceServiceConfig"`
	SessionCache       SessionCacheConfig `toml:"sessionCacheConfig"`
	Router             RouterConfig       `toml:"routerConfig"`
	// Model 来自环境变量，不参与 TOML 反序列化
	Model ModelConfig `toml:"-"`
}

type RedisKeyConfig struct {
	CaptchaPrefix   string
	IndexName       string
	IndexNamePrefix string
}

var DefaultRedisKeyConfig = RedisKeyConfig{
	CaptchaPrefix:   "captcha:%s",
	IndexName:       "rag_docs:%s:idx",
	IndexNamePrefix: "rag_docs:%s:",
}

var config *Config

// configPath 解析配置文件路径，优先级：环境变量 GOPHERAI_CONFIG > config/config.local.toml > config/config.toml。
func configPath() string {
	if p := strings.TrimSpace(os.Getenv("GOPHERAI_CONFIG")); p != "" {
		return p
	}
	candidates := []string{
		filepath.Join("config", "config.local.toml"),
		filepath.Join("config", "config.toml"),
		"config.local.toml",
		"config.toml",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	return candidates[0]
}

// InitConfig 初始化项目配置：先加载 .env，再解析 TOML，最后收口模型配置并做 fail-fast 校验。
func InitConfig() error {
	// 0. 尝试从 .env.local / .env 注入环境变量，让“复制模板即可运行”成立
	loadDotEnv()

	// 1. 解析 TOML 基础设施配置
	path := configPath()
	if _, err := toml.DecodeFile(path, config); err != nil {
		return fmt.Errorf("decode config file %q: %w", path, err)
	}

	// 2. 收口模型相关环境变量
	config.Model = loadModelConfig()

	// 3. fail-fast 校验
	if err := config.Model.Validate(); err != nil {
		return err
	}
	return nil
}

func GetConfig() *Config {
	if config == nil {
		config = new(Config)
		if err := InitConfig(); err != nil {
			log.Fatalf("[config] init failed: %v", err)
		}
	}
	return config
}

// loadDotEnv 加载 .env.local 与 .env（若存在），仅补充尚未存在的环境变量。
// 不引入第三方依赖，缺失文件静默跳过。
func loadDotEnv() {
	for _, name := range []string{".env.local", ".env"} {
		loadDotEnvFile(name)
	}
}

func loadDotEnvFile(name string) {
	f, err := os.Open(name)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		// 兼容 `export KEY=VALUE` 写法
		line = strings.TrimPrefix(line, "export ")
		idx := strings.Index(line, "=")
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		val = strings.Trim(val, `"'`)
		if key == "" {
			continue
		}
		if _, exists := os.LookupEnv(key); !exists {
			_ = os.Setenv(key, val)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintf(os.Stderr, "loadDotEnvFile %s: %v\n", name, err)
	}
}
