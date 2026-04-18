// Copyright (c) 2026 Aptlogica Technologies Private Limited
// Licensed under the Apache License, Version 2.0
// Websites: https://www.aptlogica.com | https://www.serenibase.com
// Support: support@aptlogica.com | support@serenibase.com

package logger

import (
	"io"
	"os"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"path/filepath"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/aptlogica/sereni-base/internal/config"
)

var (
	once sync.Once
	log  zerolog.Logger
)

type FilteredWriter struct {
	w      io.Writer
	levels []string
}

func (fw *FilteredWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	for _, level := range fw.levels {
		if strings.Contains(s, "\"level\":\""+level+"\"") {
			return fw.w.Write(p)
		}
	}
	return len(p), nil
}

func Init(cfg *config.Config) {
	once.Do(func() {
		zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
		zerolog.TimeFieldFormat = time.RFC3339Nano

		level := parseLogLevel(cfg.Log.Level)

		output := setupOutputWriter(cfg)

		gitRevision, goVersion := getBuildInfo()

		log = buildLogger(output, level, gitRevision, goVersion)
	})
}

func parseLogLevel(levelStr string) zerolog.Level {
	level, err := zerolog.ParseLevel(strings.ToLower(levelStr))
	if err != nil {
		level = zerolog.InfoLevel
	}
	return level
}

func setupOutputWriter(cfg *config.Config) io.Writer {
	var output io.Writer = zerolog.ConsoleWriter{
		Out:        os.Stderr,
		TimeFormat: time.RFC3339,
	}

	if cfg.Server.Env != "dev" {
		// Ensure directories exist
		if dir := filepath.Dir(cfg.Log.File); dir != "." {
			_ = os.MkdirAll(dir, 0755)
		}
		if dir := filepath.Dir(cfg.Log.ErrorFile); dir != "." {
			_ = os.MkdirAll(dir, 0755)
		}

		fileLogger := &lumberjack.Logger{
			Filename:   cfg.Log.File,
			MaxSize:    cfg.Log.MaxSize,
			MaxBackups: cfg.Log.MaxBackups,
			MaxAge:     cfg.Log.MaxAge,
			Compress:   cfg.Log.Compress,
		}

		errorFileLogger := &lumberjack.Logger{
			Filename:   cfg.Log.ErrorFile,
			MaxSize:    cfg.Log.MaxSize,
			MaxBackups: cfg.Log.MaxBackups,
			MaxAge:     cfg.Log.MaxAge,
			Compress:   cfg.Log.Compress,
		}

		// Filter for error file
		errorFiltered := &FilteredWriter{
			w:      errorFileLogger,
			levels: []string{"error", "fatal", "panic"},
		}

		output = zerolog.MultiLevelWriter(os.Stderr, fileLogger, errorFiltered)
	}

	return output
}

func getBuildInfo() (string, string) {
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
	return gitRevision, goVersion
}

func buildLogger(output io.Writer, level zerolog.Level, gitRevision, goVersion string) zerolog.Logger {
	return zerolog.New(output).
		Level(level).
		With().
		Timestamp().
		Caller().
		Str("git_revision", gitRevision).
		Str("go_version", goVersion).
		Logger()
}

func Get() *zerolog.Logger {
	return &log
}
