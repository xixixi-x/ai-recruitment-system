package config

import "os"

type Config struct {
	WebPort       string
	LogicGRPCAddr string
	JWTSecret     string
	CORSOrigins   string
}

func env(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

func Load() Config {
	return Config{
		WebPort:       env("WEB_PORT", "8080"),
		LogicGRPCAddr: env("LOGIC_GRPC_ADDR", "127.0.0.1:9090"),
		JWTSecret:     env("JWT_SECRET", "change-this-secret"),
		CORSOrigins:   env("CORS_ALLOW_ORIGINS", "http://localhost:5173,http://localhost:5174"),
	}
}
