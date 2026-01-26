package config

import (
	"os"

	"github.com/cloudwego/hertz/pkg/common/hlog"
	"gopkg.in/yaml.v3"
)

func Init(filepath string) {
	content, err := os.ReadFile(filepath)
	if err != nil {
		panic(err)
	}

	if err := yaml.Unmarshal(content, &globalConfig); err != nil {
		panic(err)
	}

	hlog.Debugf("config debug: %+v", globalConfig)
}

func GetMySQLConf() MySQLConf {
	return globalConfig.MySQL
}

func GetRedisConf() RedisConf {
	return globalConfig.Redis
}

func GetJWTConfig() JWTConf {
	return globalConfig.JWT
}

func GetCORSConf() CORSConf {
	return globalConfig.CORS
}

func GetSessionConf() SessionConf {
	return globalConfig.Session
}

func GetRateLimitConf() []RateLimitConf {
	return globalConfig.RateLimit
}

func GetLoggerConf() LoggerConf {
	return globalConfig.Logger
}

func GetLoginProtectionConf() LoginProtectionConf {
	return globalConfig.LoginProtection
}

func GetRegisterProtectionConf() RegisterProtectionConf {
	return globalConfig.RegisterProtection
}

var globalConfig ServiceConf

type ServiceConf struct {
	MySQL              MySQLConf              `yaml:"mysql"`
	Redis              RedisConf              `yaml:"redis"`
	JWT                JWTConf                `yaml:"jwt"`
	CORS               CORSConf               `yaml:"cors"`
	Session            SessionConf            `yaml:"session"`
	RateLimit          []RateLimitConf        `yaml:"rate_limit"`
	Logger             LoggerConf             `yaml:"logger"`
	LoginProtection    LoginProtectionConf    `yaml:"login_protection"`
	RegisterProtection RegisterProtectionConf `yaml:"register_protection"`
}

type LoginProtectionConf struct {
	WindowSeconds     int `yaml:"window_seconds"`
	Limit             int `yaml:"limit"`
	BlockMinDuration  int `yaml:"block_min_duration"`
	BlockHourDuration int `yaml:"block_hour_duration"`
	LevelDuration     int `yaml:"level_duration"`
}

type RegisterProtectionConf struct {
	BlockMinutes int `yaml:"block_minutes"`
}

type MySQLConf struct {
	DBName   string `yaml:"db_name"`
	IP       string `yaml:"ip"`
	Port     int    `yaml:"port"`
	Username string `yaml:"username"`
	Password string `yaml:"password"`
}

type RedisConf struct {
	IP       string `yaml:"ip"`
	Port     int    `yaml:"port"`
	Password string `yaml:"password"`
	DB       int    `yaml:"db"`
}

type JWTConf struct {
	Issuer string `yaml:"issuer"`

	AccessTokenSecret  string `yaml:"access_token_secret"`
	RefreshTokenSecret string `yaml:"refresh_token_secret"`

	AccessExpiration  int `yaml:"access_expiration"`
	RefreshExpiration int `yaml:"refresh_expiration"`
}

type CORSConf struct {
	AllowOrigins     []string `yaml:"allow_origins"`
	AllowMethods     []string `yaml:"allow_methods"`
	AllowHeaders     []string `yaml:"allow_headers"`
	AllowCredentials bool     `yaml:"allow_credentials"`
	MaxAge           int      `yaml:"max_age"`
}

type SessionConf struct {
	StorePrefix string `yaml:"store_prefix"`
	Name        string `yaml:"name"`
	Path        string `yaml:"path"`
	Domain      string `yaml:"domain"`
	MaxAge      int    `yaml:"max_age"`
	Secure      bool   `yaml:"secure"`
	HTTPOnly    bool   `yaml:"http_only"`
	SameSite    string `yaml:"same_site"`
}

type RateLimitConf struct {
	Path          string `yaml:"path"`
	WindowSeconds int    `yaml:"window_seconds"`
	Limit         int64  `yaml:"limit"`
	HasSession    bool   `yaml:"has_session"`
}

type LoggerConf struct {
	Level      string `yaml:"level"`
	Dir        string `yaml:"dir"`
	FileName   string `yaml:"file_name"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
}
