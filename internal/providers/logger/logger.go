package logger

import (
	"io"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"

	"serenibase/internal/config"
)

var (
	once sync.Once
	log  zerolog.Logger
)

func Init(cfg *config.Config) {
	once.Do(func() {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		zerolog.TimeFieldFormat = time.RFC3339Nano

		// Parse log level
		level, err := zerolog.ParseLevel(strings.ToLower(cfg.Log.Level))
		if err != nil {
			level = zerolog.InfoLevel
		}

		// Configure output
		var output io.Writer = zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}

		if cfg.Server.Env != "dev" {
			fileLogger := &lumberjack.Logger{
				Filename:   cfg.Log.File,
				MaxSize:    cfg.Log.MaxSize,
				MaxBackups: cfg.Log.MaxBackups,
				MaxAge:     cfg.Log.MaxAge,
				Compress:   cfg.Log.Compress,
			}
			output = zerolog.MultiLevelWriter(os.Stderr, fileLogger)
		}

		// Git revision + Go version
		var gitRevision, goVersion string
		if buildInfo, ok := debug.ReadBuildInfo(); ok {
			goVersion = buildInfo.GoVersion
			for _, v := range buildInfo.Settings {
				if v.Key == "vcs.revision" {
					gitRevision = v.Value
					break
				}
			}
		}

		log = zerolog.New(output).
			Level(level).
			With().
			Timestamp().
			Caller().
			Str("git_revision", gitRevision).
			Str("go_version", goVersion).
			Logger()
	})
}

func Get() zerolog.Logger {
	return log
}
