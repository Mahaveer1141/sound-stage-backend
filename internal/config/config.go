package config

import (
	"sound-stage-backend/internal/pkg/env"
	"time"

	"github.com/gin-gonic/gin"
)

type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Logger   LoggerConfig
	Mailer   MailerConfig
	JWT      JWTConfig
	Redis    RedisConfig
	Worker   WorkerConfig
	Turn     TurnConfig
}

type ServerConfig struct {
	Port         string
	Mode         string
	Environment  string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

type DatabaseConfig struct {
	DSN string
}

type LoggerConfig struct {
	Level string
}

type MailerConfig struct {
	MailgunDomain    string
	MailgunAPIKey    string
	MailgunFromEmail string
	MailgunFromName  string
}

type JWTConfig struct {
	AccessTokenSecret  string
	RefreshTokenSecret string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string
}

type RedisConfig struct {
	Addr     string
	Password string
	Username string
	DB       int
}

type WorkerConfig struct {
	Concurrency  int
	DefaultQueue int
}

type TurnConfig struct {
	Username   string
	Credential string
}

func Load() *Config {
	appEnv := env.GetEnv("APP_ENV", "development")
	var mode string
	if appEnv == "production" {
		mode = gin.ReleaseMode
	} else {
		mode = gin.DebugMode
	}

	return &Config{
		Server: ServerConfig{
			Port:         env.GetEnv("SERVER_PORT", "8000"),
			Mode:         mode,
			Environment:  appEnv,
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			IdleTimeout:  time.Minute,
		},
		Database: DatabaseConfig{
			DSN: env.GetEnv("GOOSE_DBSTRING", "host=localhost port=5432 user=postgres password=postgres dbname=sound_stage sslmode=disable"),
		},
		Logger: LoggerConfig{
			Level: env.GetEnv("LOGGER_LEVEL", "debug"),
		},
		Mailer: MailerConfig{
			MailgunDomain:    env.GetEnv("MAILGUN_DOMAIN", ""),
			MailgunAPIKey:    env.GetEnv("MAILGUN_API_KEY", ""),
			MailgunFromEmail: env.GetEnv("MAILGUN_FROM_EMAIL", ""),
			MailgunFromName:  env.GetEnv("MAILGUN_FROM_NAME", "Sound Stage"),
		},
		JWT: JWTConfig{
			AccessTokenSecret:  env.GetEnv("JWT_ACCESS_TOKEN_SECRET", "access_token_secret"),
			RefreshTokenSecret: env.GetEnv("JWT_REFRESH_TOKEN_SECRET", "refresh_token_secret"),
			AccessTokenExpiry:  2 * time.Hour,
			RefreshTokenExpiry: 7 * 24 * time.Hour, // 7 days
			Issuer:             env.GetEnv("JWT_ISSUER", "sound-stage"),
		},
		Redis: RedisConfig{
			Addr:     env.GetEnv("REDIS_ADDR", "localhost:6379"),
			Username: env.GetEnv("REDIS_USERNAME", ""),
			Password: env.GetEnv("REDIS_PASSWORD", ""),
			DB:       env.GetEnvInt("REDIS_DB", 0),
		},
		Worker: WorkerConfig{
			Concurrency:  env.GetEnvInt("WORKER_CONCURRENCY", 5),
			DefaultQueue: env.GetEnvInt("WORKER_DEFAULT_QUEUE", 1),
		},
		Turn: TurnConfig{
			Username:   env.GetEnv("TURN_USERNAME", ""),
			Credential: env.GetEnv("TURN_CREDENTIAL", ""),
		},
	}
}
