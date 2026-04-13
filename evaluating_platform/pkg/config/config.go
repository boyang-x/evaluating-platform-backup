package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 平台配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	LLM      LLMConfig      `mapstructure:"llm"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Storage  StorageConfig  `mapstructure:"storage"`
	Agent    AgentConfig    `mapstructure:"agent"`
	Auth     AuthConfig     `mapstructure:"auth"`
	Billing  BillingConfig  `mapstructure:"billing"`
	MCP      MCPConfig      `mapstructure:"mcp"`
	Crypto   CryptoConfig   `mapstructure:"crypto"`
}

// CryptoConfig 加密配置
type CryptoConfig struct {
	MasterKey string `mapstructure:"master_key"` // AES-256 密钥（32字节 hex 编码）
	KeyID     string `mapstructure:"key_id"`     // 密钥标识，用于密钥轮换
}

type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"` // debug | release
}

type LLMConfig struct {
	Provider    string  `mapstructure:"provider"`
	BaseURL     string  `mapstructure:"base_url"`
	APIKey      string  `mapstructure:"api_key"`
	Model       string  `mapstructure:"model"`
	MaxTokens   int     `mapstructure:"max_tokens"`
	Temperature float64 `mapstructure:"temperature"`
}

type DatabaseConfig struct {
	DSN         string `mapstructure:"dsn"`
	MaxConns    int    `mapstructure:"max_conns"`
	MinConns    int    `mapstructure:"min_conns"`
	MigrateOnStart bool `mapstructure:"migrate_on_start"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

type StorageConfig struct {
	Endpoint  string `mapstructure:"endpoint"`
	AccessKey string `mapstructure:"access_key"`
	SecretKey string `mapstructure:"secret_key"`
	Bucket    string `mapstructure:"bucket"`
	UseSSL    bool   `mapstructure:"use_ssl"`
}

type AgentConfig struct {
	MaxIterations      int `mapstructure:"max_iterations"`
	TimeoutSeconds     int `mapstructure:"timeout_seconds"`
	ToolTimeoutSeconds int `mapstructure:"tool_timeout_seconds"`
}

type AuthConfig struct {
	JWTSecret        string `mapstructure:"jwt_secret"`
	TokenExpiryHours int    `mapstructure:"token_expiry_hours"`
}

type BillingConfig struct {
	Mode            string  `mapstructure:"mode"` // prepaid | postpaid
	ExpertShareRate float64 `mapstructure:"expert_share_rate"` // 专家分成比例 0-1
}

// MCPConfig MCP Server 配置
type MCPConfig struct {
	Port int `mapstructure:"port"` // MCP SSE Server 监听端口，默认 18080
}

// Load 从文件或环境变量加载配置
func Load(path string) (*Config, error) {
	v := viper.New()

	// 设置默认值
	setDefaults(v)

	// 读取配置文件
	v.SetConfigFile(path)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// 环境变量覆盖（ENV_VAR -> config.key，例如 LLM_API_KEY -> llm.api_key）
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	// 显式绑定 LLM 字段（规避 viper 嵌套 struct Unmarshal 的已知问题）
	_ = v.BindEnv("llm.base_url", "LLM_BASE_URL")
	_ = v.BindEnv("llm.api_key", "LLM_API_KEY")
	_ = v.BindEnv("llm.model", "LLM_MODEL")

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}

	return &cfg, nil
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("server.port", 8080)
	v.SetDefault("server.mode", "debug")

	v.SetDefault("llm.provider", "openai")
	v.SetDefault("llm.base_url", "https://api.openai.com/v1")
	v.SetDefault("llm.model", "gpt-4o")
	v.SetDefault("llm.max_tokens", 4096)
	v.SetDefault("llm.temperature", 0.7)

	v.SetDefault("database.max_conns", 20)
	v.SetDefault("database.min_conns", 2)
	v.SetDefault("database.migrate_on_start", true)

	v.SetDefault("redis.addr", "localhost:6379")
	v.SetDefault("redis.db", 0)

	v.SetDefault("agent.max_iterations", 20)
	v.SetDefault("agent.timeout_seconds", 300)
	v.SetDefault("agent.tool_timeout_seconds", 15)

	v.SetDefault("auth.token_expiry_hours", 24)

	v.SetDefault("billing.mode", "prepaid")
	v.SetDefault("billing.expert_share_rate", 0.3)

	v.SetDefault("mcp.port", 18080)

	v.SetDefault("crypto.master_key", "")
	v.SetDefault("crypto.key_id", "v1")
}
