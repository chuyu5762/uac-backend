package config

import (
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Config 应用配置
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Database DatabaseConfig `mapstructure:"database"`
	Redis    RedisConfig    `mapstructure:"redis"`
	JWT      JWTConfig      `mapstructure:"jwt"`
	Static   StaticConfig   `mapstructure:"static"`
}

// StaticConfig 静态文件配置
type StaticConfig struct {
	// Enabled 是否启用静态文件服务
	Enabled bool `mapstructure:"enabled"`
	// Mode 服务模式：embed（嵌入）或 disk（磁盘）
	Mode string `mapstructure:"mode"`
	// Path 磁盘模式下的文件路径
	Path string `mapstructure:"path"`
}

// 全局配置实例
var globalConfig *Config

// ServerConfig 服务器配置
type ServerConfig struct {
	Addr         string        `mapstructure:"addr"`
	Mode         string        `mapstructure:"mode"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout"`
	WriteTimeout time.Duration `mapstructure:"write_timeout"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Driver   string         `mapstructure:"driver"`
	Postgres PostgresConfig `mapstructure:"postgres"`
	MySQL    MySQLConfig    `mapstructure:"mysql"`
}

// PostgresConfig PostgreSQL 配置
type PostgresConfig struct {
	Host     string `mapstructure:"host"`
	Port     int    `mapstructure:"port"`
	User     string `mapstructure:"user"`
	Password string `mapstructure:"password"`
	DBName   string `mapstructure:"dbname"`
	SSLMode  string `mapstructure:"sslmode"`
}

// MySQLConfig MySQL 配置
type MySQLConfig struct {
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	User      string `mapstructure:"user"`
	Password  string `mapstructure:"password"`
	DBName    string `mapstructure:"dbname"`
	Charset   string `mapstructure:"charset"`
	ParseTime bool   `mapstructure:"parse_time"`
	Loc       string `mapstructure:"loc"`
}

// RedisConfig Redis 配置
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// JWTConfig JWT 配置
type JWTConfig struct {
	PrivateKeyPath string        `mapstructure:"private_key_path"`
	PublicKeyPath  string        `mapstructure:"public_key_path"`
	Issuer         string        `mapstructure:"issuer"`
	AccessExpiry   time.Duration `mapstructure:"access_expiry"`
	RefreshExpiry  time.Duration `mapstructure:"refresh_expiry"`
}

// Load 加载配置
// 支持从配置文件和环境变量加载配置
// 环境变量格式：UAC_SERVER_ADDR, UAC_DATABASE_DRIVER 等
func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")

	// 支持环境变量覆盖
	// 环境变量前缀为 UAC，使用下划线分隔
	// 例如：UAC_SERVER_ADDR 对应 server.addr
	viper.SetEnvPrefix("UAC")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		// 配置文件不存在时使用默认值
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	// 保存全局配置实例
	globalConfig = &cfg

	return &cfg, nil
}

// Get 获取全局配置实例
// 必须先调用 Load() 初始化配置
func Get() *Config {
	return globalConfig
}

// LoadFromFile 从指定路径加载配置文件
func LoadFromFile(path string) (*Config, error) {
	viper.SetConfigFile(path)

	// 支持环境变量覆盖
	viper.SetEnvPrefix("UAC")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, err
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	globalConfig = &cfg
	return &cfg, nil
}

// setDefaults 设置默认值
func setDefaults() {
	// 服务器默认配置
	viper.SetDefault("server.addr", ":8080")
	viper.SetDefault("server.mode", "debug")
	viper.SetDefault("server.read_timeout", "10s")
	viper.SetDefault("server.write_timeout", "10s")

	// 数据库默认配置
	viper.SetDefault("database.driver", "postgres")
	viper.SetDefault("database.postgres.host", "localhost")
	viper.SetDefault("database.postgres.port", 5432)
	viper.SetDefault("database.postgres.user", "postgres")
	viper.SetDefault("database.postgres.password", "")
	viper.SetDefault("database.postgres.dbname", "unified_auth")
	viper.SetDefault("database.postgres.sslmode", "disable")

	// Redis 默认配置
	viper.SetDefault("redis.addr", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("redis.db", 0)

	// JWT 默认配置
	viper.SetDefault("jwt.issuer", "unified-auth-center")
	viper.SetDefault("jwt.access_expiry", "2h")
	viper.SetDefault("jwt.refresh_expiry", "168h")

	// 静态文件默认配置
	viper.SetDefault("static.enabled", true)
	viper.SetDefault("static.mode", "embed")
	viper.SetDefault("static.path", "./web/dist")
}
