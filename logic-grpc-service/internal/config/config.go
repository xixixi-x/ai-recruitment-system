package config

import (
	"os"
	"strconv"
)

type Config struct {
	LogicGRPCPort      string
	MySQLDSN           string
	OSSEndpoint        string
	OSSAccessKeyID     string
	OSSAccessKeySecret string
	OSSBucket          string
	OSSSignExpire      int64
	AIAPIKey           string
	AIBaseURL          string
	AIModel            string
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() Config {
	exp, _ := strconv.ParseInt(env("OSS_SIGN_EXPIRE_SECONDS", "600"), 10, 64)
	return Config{
		LogicGRPCPort:      env("LOGIC_GRPC_PORT", "9090"),
		MySQLDSN:           env("MYSQL_DSN", "root:123456@tcp(127.0.0.1:3306)/ai_recruitment?charset=utf8mb4&parseTime=True&loc=Local"),
		OSSEndpoint:        env("OSS_ENDPOINT", ""),
		OSSAccessKeyID:     env("OSS_ACCESS_KEY_ID", ""),
		OSSAccessKeySecret: env("OSS_ACCESS_KEY_SECRET", ""),
		OSSBucket:          env("OSS_BUCKET", ""),
		OSSSignExpire:      exp,
		AIAPIKey:           env("AI_API_KEY", ""),
		AIBaseURL:          env("AI_BASE_URL", "https://api.deepseek.com/v1"),
		AIModel:            env("AI_MODEL", "deepseek-chat"),
	}
}
