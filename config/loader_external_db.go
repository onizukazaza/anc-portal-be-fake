package config

import (
	"strings"

	"github.com/spf13/viper"
)

func loadExternalDBs(v *viper.Viper) (map[string]Database, error) {
	raw := v.GetString("external_dbs")
	parts := strings.Split(raw, ",")

	result := make(map[string]Database)

	for _, name := range parts {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}

		prefix := "EXTERNAL_DBS_" + strings.ToUpper(name) + "_"

		db := Database{
			Driver:           v.GetString(prefix + "DRIVER"),
			Host:             v.GetString(prefix + "HOST"),
			Port:             v.GetInt(prefix + "PORT"),
			User:             v.GetString(prefix + "USER"),
			Password:         v.GetString(prefix + "PASSWORD"),
			DBName:           v.GetString(prefix + "DBNAME"),
			SSLMode:          v.GetString(prefix + "SSLMODE"),
			Schema:           v.GetString(prefix + "SCHEMA"),
			MaxConns:         v.GetInt(prefix + "MAX_CONNS"),
			MinConns:         v.GetInt(prefix + "MIN_CONNS"),
			MaxConnLifetime:  v.GetDuration(prefix + "MAX_CONN_LIFETIME"),
			MaxConnIdleTime:  v.GetDuration(prefix + "MAX_CONN_IDLE_TIME"),
			ConnectTimeout:   v.GetDuration(prefix + "CONNECT_TIMEOUT"),
			StatementTimeout: v.GetDuration(prefix + "STATEMENT_TIMEOUT"),
		}

		result[name] = db
	}

	return result, nil
}
