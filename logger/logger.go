package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Logger *zap.Logger

func Init() error {
	// Customize encoder config as needed
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:       "time",
		LevelKey:      "level",
		NameKey:       "logger",
		CallerKey:     "caller",
		MessageKey:    "msg",
		StacktraceKey: "stacktrace",
		LineEnding:    zapcore.DefaultLineEnding,
		EncodeLevel:   zapcore.CapitalColorLevelEncoder, // Colorized for console
		EncodeTime:    zapcore.ISO8601TimeEncoder,       // ISO8601 time format
		EncodeCaller:  zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewConsoleEncoder(encoderConfig), // or zapcore.NewJSONEncoder(encoderConfig)
		zapcore.AddSync(os.Stdout),               // Write to stdout
		zapcore.DebugLevel,                       // Minimum log level
	)

	var err error
	Logger = zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return err
}

func Sync() {
	if Logger != nil {
		_ = Logger.Sync()
	}
}
