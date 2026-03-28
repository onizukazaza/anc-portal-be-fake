package log

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
)

var (
	defaultLogger zerolog.Logger
	once          sync.Once
)

func New(service string) *zerolog.Logger {
	env := strings.ToLower(strings.TrimSpace(os.Getenv("STAGE_STATUS")))
	if env == "" {
		env = "local"
	}

	level := parseLevel(os.Getenv("LOG_LEVEL"))
	zerolog.TimeFieldFormat = time.RFC3339

	var logger zerolog.Logger
	if env == "local" || env == "uat" {
		writer := zerolog.ConsoleWriter{
			Out:        os.Stdout,
			TimeFormat: "15:04:05",
		}

		writer.FormatLevel = func(i any) string {
			s := strings.ToUpper(fmt.Sprintf("%s", i))
			switch s {
			case "DEBUG":
				return "\x1b[36mDBG\x1b[0m"
			case "INFO":
				return "\x1b[32mINF\x1b[0m"
			case "WARN":
				return "\x1b[33mWRN\x1b[0m"
			case "ERROR":
				return "\x1b[31mERR\x1b[0m"
			default:
				return s
			}
		}

		logger = zerolog.New(writer).
			Level(level).
			With().
			Timestamp().
			Str("service", service).
			Str("env", env).
			Caller().
			Logger()
	} else {
		logger = zerolog.New(os.Stdout).
			Level(level).
			With().
			Timestamp().
			Str("service", service).
			Str("env", env).
			Caller().
			Logger()
	}

	return &logger
}

func Set(l *zerolog.Logger) {
	if l == nil {
		return
	}
	defaultLogger = *l
}

func L() *zerolog.Logger {
	once.Do(func() {
		if defaultLogger.GetLevel() == zerolog.NoLevel {
			defaultLogger = *New("anc-portal-be")
		}
	})
	return &defaultLogger
}

func parseLevel(s string) zerolog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "trace":
		return zerolog.TraceLevel
	case "debug":
		return zerolog.DebugLevel
	case "warn", "warning":
		return zerolog.WarnLevel
	case "error":
		return zerolog.ErrorLevel
	case "fatal":
		return zerolog.FatalLevel
	case "panic":
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}
