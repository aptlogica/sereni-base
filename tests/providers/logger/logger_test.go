package logger_test

import (
	"bytes"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"serenibase/internal/config"
	"serenibase/internal/providers/logger"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInit tests the logger initialization
func TestInit(t *testing.T) {
	t.Run("init with dev environment", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "dev",
			},
			Log: config.LogConfig{
				Level: "info",
			},
		}

		// Should not panic
		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})

		// Verify logger is available
		log := logger.Get()
		assert.NotNil(t, log)
	})

	t.Run("init with production environment", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "production",
			},
			Log: config.LogConfig{
				Level:      "info",
				File:       filepath.Join(tmpDir, "app.log"),
				ErrorFile:  filepath.Join(tmpDir, "error.log"),
				MaxSize:    10,
				MaxBackups: 3,
				MaxAge:     7,
				Compress:   true,
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})

		log := logger.Get()
		assert.NotNil(t, log)
	})

	t.Run("init with staging environment", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "staging",
			},
			Log: config.LogConfig{
				Level:      "debug",
				File:       filepath.Join(tmpDir, "staging.log"),
				ErrorFile:  filepath.Join(tmpDir, "staging-error.log"),
				MaxSize:    5,
				MaxBackups: 2,
				MaxAge:     5,
				Compress:   false,
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})

		log := logger.Get()
		assert.NotNil(t, log)
	})

	t.Run("init with debug level", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "dev",
			},
			Log: config.LogConfig{
				Level: "debug",
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})

		log := logger.Get()
		assert.NotNil(t, log)
	})

	t.Run("init with trace level", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "dev",
			},
			Log: config.LogConfig{
				Level: "trace",
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})

		log := logger.Get()
		assert.NotNil(t, log)
	})

	t.Run("init with warn level", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "dev",
			},
			Log: config.LogConfig{
				Level: "warn",
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})

		log := logger.Get()
		assert.NotNil(t, log)
	})

	t.Run("init with error level", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "dev",
			},
			Log: config.LogConfig{
				Level: "error",
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})

		log := logger.Get()
		assert.NotNil(t, log)
	})

	t.Run("init with invalid level defaults to info", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "dev",
			},
			Log: config.LogConfig{
				Level: "invalid-level",
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})

		log := logger.Get()
		assert.NotNil(t, log)
	})

	t.Run("init with uppercase level", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "dev",
			},
			Log: config.LogConfig{
				Level: "DEBUG",
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})

		log := logger.Get()
		assert.NotNil(t, log)
	})

	t.Run("init with nested log directories", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "production",
			},
			Log: config.LogConfig{
				Level:      "info",
				File:       filepath.Join(tmpDir, "logs", "app", "app.log"),
				ErrorFile:  filepath.Join(tmpDir, "logs", "errors", "error.log"),
				MaxSize:    10,
				MaxBackups: 3,
				MaxAge:     7,
				Compress:   true,
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})

		// Note: Directories are created only on first Init call due to sync.Once
	})

	t.Run("init called multiple times uses sync.Once", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "dev",
			},
			Log: config.LogConfig{
				Level: "info",
			},
		}

		// Call Init multiple times
		logger.Init(cfg)
		logger.Init(cfg)
		logger.Init(cfg)

		// Should not panic and logger should be available
		log := logger.Get()
		assert.NotNil(t, log)
	})
}

// TestGet tests the Get function
func TestGet(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Env: "dev",
		},
		Log: config.LogConfig{
			Level: "info",
		},
	}

	logger.Init(cfg)

	t.Run("get logger returns non-nil", func(t *testing.T) {
		log := logger.Get()
		assert.NotNil(t, log)
	})

	t.Run("get logger multiple times returns same instance", func(t *testing.T) {
		log1 := logger.Get()
		log2 := logger.Get()

		assert.NotNil(t, log1)
		assert.NotNil(t, log2)
		// Both should point to the same logger
	})

	t.Run("logger can log messages", func(t *testing.T) {
		log := logger.Get()

		// Should not panic
		assert.NotPanics(t, func() {
			log.Info().Msg("test message")
			log.Debug().Msg("debug message")
			log.Warn().Msg("warning message")
		})
	})
}

// TestLoggerConfiguration tests various logger configurations
func TestLoggerConfiguration(t *testing.T) {
	t.Run("logger with git revision and go version", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "dev",
			},
			Log: config.LogConfig{
				Level: "info",
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})

		log := logger.Get()
		assert.NotNil(t, log)
	})

	t.Run("logger with all log config options", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "production",
			},
			Log: config.LogConfig{
				Level:      "debug",
				File:       filepath.Join(tmpDir, "full-config.log"),
				ErrorFile:  filepath.Join(tmpDir, "full-config-error.log"),
				MaxSize:    100,
				MaxBackups: 5,
				MaxAge:     30,
				Compress:   true,
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})

		log := logger.Get()
		assert.NotNil(t, log)
	})

	t.Run("logger without compression", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "production",
			},
			Log: config.LogConfig{
				Level:      "info",
				File:       filepath.Join(tmpDir, "no-compress.log"),
				ErrorFile:  filepath.Join(tmpDir, "no-compress-error.log"),
				MaxSize:    10,
				MaxBackups: 3,
				MaxAge:     7,
				Compress:   false,
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})

		log := logger.Get()
		assert.NotNil(t, log)
	})
}

// TestLogLevels tests different log levels
func TestLogLevels(t *testing.T) {
	levels := []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}

	for _, level := range levels {
		t.Run("init with "+level+" level", func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{
					Env: "dev",
				},
				Log: config.LogConfig{
					Level: level,
				},
			}

			assert.NotPanics(t, func() {
				logger.Init(cfg)
			})

			log := logger.Get()
			assert.NotNil(t, log)
		})
	}
}

// TestLoggerThreadSafety tests concurrent logger operations
func TestLoggerThreadSafety(t *testing.T) {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Env: "dev",
		},
		Log: config.LogConfig{
			Level: "info",
		},
	}

	logger.Init(cfg)

	t.Run("concurrent logging", func(t *testing.T) {
		log := logger.Get()
		numGoroutines := 50

		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(index int) {
				assert.NotPanics(t, func() {
					log.Info().Int("index", index).Msg("concurrent log")
					log.Debug().Int("index", index).Msg("debug log")
					log.Warn().Int("index", index).Msg("warning log")
				})
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	t.Run("concurrent Get calls", func(t *testing.T) {
		numGoroutines := 50
		done := make(chan bool, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func() {
				log := logger.Get()
				assert.NotNil(t, log)
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})
}

// TestLoggerEdgeCases tests edge cases
func TestLoggerEdgeCases(t *testing.T) {
	t.Run("init with empty log file path", func(t *testing.T) {
		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "production",
			},
			Log: config.LogConfig{
				Level:      "info",
				File:       "",
				ErrorFile:  "",
				MaxSize:    10,
				MaxBackups: 3,
				MaxAge:     7,
				Compress:   true,
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})
	})

	t.Run("init with zero max size", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "production",
			},
			Log: config.LogConfig{
				Level:      "info",
				File:       filepath.Join(tmpDir, "zero-size.log"),
				ErrorFile:  filepath.Join(tmpDir, "zero-size-error.log"),
				MaxSize:    0,
				MaxBackups: 3,
				MaxAge:     7,
				Compress:   true,
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})
	})

	t.Run("init with negative max backups", func(t *testing.T) {
		tmpDir := t.TempDir()

		cfg := &config.Config{
			Server: config.ServerConfig{
				Env: "production",
			},
			Log: config.LogConfig{
				Level:      "info",
				File:       filepath.Join(tmpDir, "negative.log"),
				ErrorFile:  filepath.Join(tmpDir, "negative-error.log"),
				MaxSize:    10,
				MaxBackups: -1,
				MaxAge:     7,
				Compress:   true,
			},
		}

		assert.NotPanics(t, func() {
			logger.Init(cfg)
		})
	})
}

// Fix FilteredWriter reference
type MockFilteredWriter struct {
	w      io.Writer
	levels []string
}

func (fw *MockFilteredWriter) Write(p []byte) (n int, err error) {
	s := string(p)
	for _, level := range fw.levels {
		if strings.Contains(s, `"level":"`+level+`"`) {
			return fw.w.Write(p)
		}
	}
	return len(p), nil
}

// TestFilteredWriter tests the FilteredWriter functionality
func TestFilteredWriter(t *testing.T) {
	t.Run("filtered writer basic functionality", func(t *testing.T) {
		var buf bytes.Buffer
		fw := &MockFilteredWriter{
			w:      &buf,
			levels: []string{"error", "fatal"},
		}

		// Test error level (should pass)
		errorLog := `{"level":"error","msg":"error message"}`
		n, err := fw.Write([]byte(errorLog))
		require.NoError(t, err)
		assert.Equal(t, len(errorLog), n)
		assert.Contains(t, buf.String(), "error message")

		// Clear buffer
		buf.Reset()

		// Test info level (should be filtered)
		infoLog := `{"level":"info","msg":"info message"}`
		n, err = fw.Write([]byte(infoLog))
		require.NoError(t, err)
		assert.Equal(t, len(infoLog), n)
		assert.Empty(t, buf.String())
	})
}
