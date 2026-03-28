package config

import "time"

type Config struct {
	StageStatus string              `mapstructure:"stageStatus" validate:"required,oneof=local staging production"`
	Server      Server              `mapstructure:"server"`
	Swagger     Swagger             `mapstructure:"swagger"`
	Database    Database            `mapstructure:"database"`
	ExternalDBs map[string]Database `mapstructure:"externalDbs"`
	Redis       Redis               `mapstructure:"redis"`
	LocalCache  LocalCache          `mapstructure:"localCache"`
	OTel        OTel                `mapstructure:"otel"`
	Kafka       Kafka               `mapstructure:"kafka"`
}

type Server struct {
	Port         int           `mapstructure:"port" validate:"required"`
	AllowOrigins []string      `mapstructure:"allowOrigins" validate:"required"`
	BodyLimit    int           `mapstructure:"bodyLimit" validate:"required"`
	Timeout      time.Duration `mapstructure:"timeout" validate:"required"`
	JWTSecretKey string        `mapstructure:"jwtSecretKey" validate:"required"`
	APIKeys      APIKeyConfig  `mapstructure:"apiKeys"`
	RateLimit    RateLimit     `mapstructure:"rateLimit"`
}

type APIKeyConfig struct {
	Internal []string `mapstructure:"internal"`
	Partner  []string `mapstructure:"partner"`
}

type RateLimit struct {
	Enabled    bool          `mapstructure:"enabled"`
	Max        int           `mapstructure:"max"`        // max requests per window (default: 100)
	Expiration time.Duration `mapstructure:"expiration"` // window duration (default: 1m)
}

type Database struct {
	Host     string `mapstructure:"host" validate:"required"`
	Port     int    `mapstructure:"port" validate:"required"`
	User     string `mapstructure:"user" validate:"required"`
	Password string `mapstructure:"password" validate:"required"`
	DBName   string `mapstructure:"dbName" validate:"required"`
	SSLMode  string `mapstructure:"sslMode" validate:"required"`
	Schema   string `mapstructure:"schema" validate:"required"`

	MaxConns         int           `mapstructure:"maxConns" validate:"required,gte=1"`
	MinConns         int           `mapstructure:"minConns" validate:"required,gte=0"`
	MaxConnLifetime  time.Duration `mapstructure:"maxConnLifetime" validate:"required,gt=0"`
	MaxConnIdleTime  time.Duration `mapstructure:"maxConnIdleTime" validate:"required,gt=0"`
	ConnectTimeout   time.Duration `mapstructure:"connectTimeout" validate:"omitempty,gt=0"`
	StatementTimeout time.Duration `mapstructure:"statementTimeout" validate:"omitempty,gt=0"`
}

type Redis struct {
	Enabled   bool   `mapstructure:"enabled"`
	Host      string `mapstructure:"host"`
	Port      int    `mapstructure:"port"`
	Password  string `mapstructure:"password"`
	DB        int    `mapstructure:"db"`
	KeyPrefix string `mapstructure:"keyPrefix"`
}

type OTel struct {
	Enabled     bool    `mapstructure:"enabled"`
	ProjectID   string  `mapstructure:"projectID"`
	ServiceName string  `mapstructure:"serviceName"`
	Release     string  `mapstructure:"release"`
	Env         string  `mapstructure:"env"`
	SampleRatio float64 `mapstructure:"sampleRatio" validate:"gte=0,lte=1"`
	ExporterURL string  `mapstructure:"exporterURL"`
}

type Swagger struct {
	Enabled  bool     `mapstructure:"enabled"`
	Host     string   `mapstructure:"host"`
	Schemes  []string `mapstructure:"schemes"`
	BasePath string   `mapstructure:"basePath"`
}

type LocalCache struct {
	Enabled bool          `mapstructure:"enabled"`
	MaxSize int           `mapstructure:"maxSize"`
	TTL     time.Duration `mapstructure:"ttl"`
}

type Kafka struct {
	Enabled      bool          `mapstructure:"enabled"`
	Brokers      []string      `mapstructure:"brokers" validate:"omitempty,dive,required"`
	Topic        string        `mapstructure:"topic"`
	GroupID      string        `mapstructure:"groupID"`
	DLQTopic     string        `mapstructure:"dlqTopic"`
	WriteTimeout time.Duration `mapstructure:"writeTimeout"`
	MaxBytes     int           `mapstructure:"maxBytes"`
	MaxRetries   int           `mapstructure:"maxRetries"`
}
