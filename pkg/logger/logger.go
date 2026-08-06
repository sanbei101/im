package logger

import (
	"context"
	"io"
	"os"

	"github.com/jackc/pgx/v5/tracelog"
	"github.com/phuslu/log"
)

func InitLogger() {
	isProd := os.Getenv("PRODUCTION") == "true"

	var writer log.Writer

	if isProd {
		writer = &log.AsyncWriter{
			ChannelSize:   4096,
			DiscardOnFull: false,
			Writer:        &log.IOWriter{Writer: os.Stderr},
		}
	} else {
		writer = &log.ConsoleWriter{
			ColorOutput:    true,
			QuoteString:    true,
			EndWithMessage: true,
		}
	}

	log.DefaultLogger = log.Logger{
		Level:  log.InfoLevel,
		Caller: 0,
		Writer: writer,
	}
}

func Close() {
	if closer, ok := log.DefaultLogger.Writer.(io.Closer); ok {
		closer.Close()
	}
}

func NewPgxLogger() tracelog.Logger {
	return tracelog.LoggerFunc(func(_ context.Context, level tracelog.LogLevel, msg string, data map[string]any) {
		var entry *log.Entry
		switch level {
		case tracelog.LogLevelTrace:
			entry = log.Trace()
		case tracelog.LogLevelDebug:
			entry = log.Debug()
		case tracelog.LogLevelInfo:
			entry = log.Info()
		case tracelog.LogLevelWarn:
			entry = log.Warn()
		case tracelog.LogLevelError:
			entry = log.Error()
		default:
			entry = log.Error().Str("PGX_LOG_LEVEL", level.String())
		}
		for k, v := range data {
			entry.Any(k, v)
		}
		entry.Msg(msg)
	})
}
