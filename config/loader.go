package config

import (
	"fmt"
	"strings"

	"github.com/joho/godotenv"
	"github.com/mitchellh/mapstructure"
	"github.com/onizukazaza/anc-portal-be-fake/internal/shared/validator"
	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	_ = godotenv.Load()

	fmt.Println("🐍 viper: loading configuration...")
	v := viper.New()

	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("./config")

	if err := v.ReadInConfig(); err != nil {
		fmt.Println("🐍 viper: no config.yaml found, using env only")
	}

	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	bindEnvs(v)
	setDefaults(v)

	v.SetDefault("externalDbs", map[string]interface{}{})

	var cfg Config

	if err := v.Unmarshal(&cfg, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			mapstructure.StringToTimeDurationHookFunc(),
			mapstructure.StringToSliceHookFunc(","),
		),
	)); err != nil {
		return nil, fmt.Errorf("config unmarshal failed: %w", err)
	}

	externalDBs, err := loadExternalDBs(v)
	if err != nil {
		return nil, err
	}
	cfg.ExternalDBs = externalDBs

	if err := validator.Get().Struct(cfg); err != nil {
		return nil, fmt.Errorf("🐍 config validation error: %w", err)
	}

	fmt.Println("🐍 viper: configuration loaded successfully ✓")

	for name := range cfg.ExternalDBs {
		db := cfg.ExternalDBs[name]
		if err := validator.Get().Struct(db); err != nil {
			return nil, fmt.Errorf("external db [%s] invalid: %w", name, err)
		}
	}

	if cfg.StageStatus == "production" && cfg.Server.JWTSecretKey == "" {
		return nil, fmt.Errorf("JWT secret required in production")
	}

	return &cfg, nil
}

func bindEnvs(v *viper.Viper) {
	_ = v.BindEnv("stageStatus", "STAGE_STATUS")
	_ = v.BindEnv("server.port", "SERVER_PORT")
	_ = v.BindEnv("server.allowOrigins", "SERVER_ALLOW_ORIGINS")
	_ = v.BindEnv("server.bodyLimit", "SERVER_BODY_LIMIT")
	_ = v.BindEnv("server.timeout", "SERVER_TIMEOUT")
	_ = v.BindEnv("server.jwtSecretKey", "SERVER_JWT_SECRET_KEY")
	_ = v.BindEnv("server.jwtExpiry", "SERVER_JWT_EXPIRY")

	_ = v.BindEnv("server.apiKeys.internal", "SERVER_APIKEYS_INTERNAL")
	_ = v.BindEnv("server.apiKeys.partner", "SERVER_APIKEYS_PARTNER")

	_ = v.BindEnv("server.rateLimit.enabled", "SERVER_RATE_LIMIT_ENABLED")
	_ = v.BindEnv("server.rateLimit.max", "SERVER_RATE_LIMIT_MAX")
	_ = v.BindEnv("server.rateLimit.expiration", "SERVER_RATE_LIMIT_EXPIRATION")

	_ = v.BindEnv("database.host", "DB_HOST")
	_ = v.BindEnv("database.port", "DB_PORT")
	_ = v.BindEnv("database.user", "DB_USER")
	_ = v.BindEnv("database.password", "DB_PASSWORD")
	_ = v.BindEnv("database.dbName", "DB_NAME")
	_ = v.BindEnv("database.sslMode", "DB_SSLMODE")
	_ = v.BindEnv("database.schema", "DB_SCHEMA")
	_ = v.BindEnv("database.maxConns", "DB_MAX_CONNS")
	_ = v.BindEnv("database.minConns", "DB_MIN_CONNS")
	_ = v.BindEnv("database.maxConnLifetime", "DB_MAX_CONN_LIFETIME")
	_ = v.BindEnv("database.maxConnIdleTime", "DB_MAX_CONN_IDLE_TIME")
	_ = v.BindEnv("database.connectTimeout", "DB_CONNECT_TIMEOUT")
	_ = v.BindEnv("database.statementTimeout", "DB_STATEMENT_TIMEOUT")

	_ = v.BindEnv("redis.enabled", "REDIS_ENABLED")
	_ = v.BindEnv("redis.host", "REDIS_HOST")
	_ = v.BindEnv("redis.port", "REDIS_PORT")
	_ = v.BindEnv("redis.password", "REDIS_PASSWORD")
	_ = v.BindEnv("redis.db", "REDIS_DB")
	_ = v.BindEnv("redis.keyPrefix", "REDIS_KEY_PREFIX")

	_ = v.BindEnv("localCache.enabled", "LOCAL_CACHE_ENABLED")
	_ = v.BindEnv("localCache.maxSize", "LOCAL_CACHE_MAX_SIZE")
	_ = v.BindEnv("localCache.ttl", "LOCAL_CACHE_TTL")

	_ = v.BindEnv("otel.enabled", "OTEL_ENABLED")
	_ = v.BindEnv("otel.serviceName", "OTEL_SERVICE_NAME")
	_ = v.BindEnv("otel.exporterURL", "OTEL_EXPORTER_URL")
	_ = v.BindEnv("otel.sampleRatio", "OTEL_SAMPLE_RATIO")

	_ = v.BindEnv("swagger.enabled", "SWAGGER_ENABLED")
	_ = v.BindEnv("swagger.host", "SWAGGER_HOST")
	_ = v.BindEnv("swagger.schemes", "SWAGGER_SCHEMES")
	_ = v.BindEnv("swagger.basePath", "SWAGGER_BASE_PATH")

	_ = v.BindEnv("kafka.enabled", "KAFKA_ENABLED")
	_ = v.BindEnv("kafka.brokers", "KAFKA_BROKERS")
	_ = v.BindEnv("kafka.topic", "KAFKA_TOPIC")
	_ = v.BindEnv("kafka.groupID", "KAFKA_GROUP_ID")
	_ = v.BindEnv("kafka.dlqTopic", "KAFKA_DLQ_TOPIC")
	_ = v.BindEnv("kafka.writeTimeout", "KAFKA_WRITE_TIMEOUT")
	_ = v.BindEnv("kafka.maxBytes", "KAFKA_MAX_BYTES")
	_ = v.BindEnv("kafka.maxRetries", "KAFKA_MAX_RETRIES")

	_ = v.BindEnv("webhook.enabled", "WEBHOOK_ENABLED")
	_ = v.BindEnv("webhook.githubSecret", "WEBHOOK_GITHUB_SECRET")
	_ = v.BindEnv("webhook.discordWebhookUrl", "WEBHOOK_DISCORD_WEBHOOK_URL")

	_ = v.BindEnv("mock.enabled", "MOCK_ENABLED")
	_ = v.BindEnv("mock.routesFile", "MOCK_ROUTES_FILE")
}

func setDefaults(v *viper.Viper) {
	v.SetDefault("stageStatus", "local")

	v.SetDefault("server.port", 3000)
	v.SetDefault("server.bodyLimit", 4*1024*1024)
	v.SetDefault("server.timeout", "5s")
	v.SetDefault("server.jwtExpiry", "24h")
	v.SetDefault("server.rateLimit.enabled", false)
	v.SetDefault("server.rateLimit.max", 100)
	v.SetDefault("server.rateLimit.expiration", "1m")

	v.SetDefault("swagger.enabled", true)
	v.SetDefault("swagger.host", "localhost:3000")
	v.SetDefault("swagger.schemes", []string{"http"})
	v.SetDefault("swagger.basePath", "/v1")

	v.SetDefault("database.port", 5432)
	v.SetDefault("database.sslMode", "disable")
	v.SetDefault("database.schema", "public")
	v.SetDefault("database.maxConns", 20)
	v.SetDefault("database.minConns", 5)
	v.SetDefault("database.maxConnLifetime", "1h")
	v.SetDefault("database.maxConnIdleTime", "30m")
	v.SetDefault("database.connectTimeout", "5s")
	v.SetDefault("database.statementTimeout", "5s")

	v.SetDefault("redis.enabled", false)
	v.SetDefault("redis.port", 6379)
	v.SetDefault("redis.keyPrefix", "anc:")

	v.SetDefault("localCache.enabled", false)
	v.SetDefault("localCache.maxSize", 10000)
	v.SetDefault("localCache.ttl", "5m")

	v.SetDefault("otel.enabled", false)
	v.SetDefault("otel.sampleRatio", 1.0)
	v.SetDefault("otel.exporterURL", "localhost:4318")

	v.SetDefault("kafka.enabled", false)
	v.SetDefault("kafka.brokers", []string{"localhost:9092"})
	v.SetDefault("kafka.topic", "anc-topic")
	v.SetDefault("kafka.groupID", "anc-portal-group")
	v.SetDefault("kafka.dlqTopic", "anc-topic-dlq")
	v.SetDefault("kafka.writeTimeout", "10s")
	v.SetDefault("kafka.maxBytes", 10485760)
	v.SetDefault("kafka.maxRetries", 3)

	v.SetDefault("webhook.enabled", false)

	v.SetDefault("mock.enabled", false)
	v.SetDefault("mock.routesFile", "mockdata/routes.json")
}
