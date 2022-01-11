package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type logger struct {
	_logger *zap.SugaredLogger
}

var defaultLogger = &logger{
	_logger: initSugaredLogger(),
}

func initSugaredLogger() *zap.SugaredLogger {
	// First, define our level-handling logic.
	highPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl >= zapcore.ErrorLevel
	})
	lowPriority := zap.LevelEnablerFunc(func(lvl zapcore.Level) bool {
		return lvl < zapcore.ErrorLevel
	})

	// High-priority output should also go to standard error, and low-priority
	// output should also go to standard out.
	stdoutWriteSyncer := zapcore.Lock(os.Stdout)
	stderrWriteSyncer := zapcore.Lock(os.Stderr)

	productionEncoder := zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig())
	core := zapcore.NewTee(
		zapcore.NewCore(productionEncoder, stderrWriteSyncer, highPriority),
		zapcore.NewCore(productionEncoder, stdoutWriteSyncer, lowPriority),
	)

	// "zap.AddCallerSkip(1)" can locate the real caller because we wrap the zap logger
	_zapLogger := zap.New(core, zap.WithCaller(true), zap.AddStacktrace(zapcore.ErrorLevel), zap.AddCallerSkip(1))
	defer func(zapLogger *zap.Logger) {
		_ = zapLogger.Sync() // flushes buffer, if any
	}(_zapLogger)

	return _zapLogger.Sugar()
}

func Debug(args ...interface{}) {
	defaultLogger.Debug(args...)
}

func Info(args ...interface{}) {
	defaultLogger.Info(args...)
}

func Warn(args ...interface{}) {
	defaultLogger.Warn(args...)
}

func Error(args ...interface{}) {
	defaultLogger.Error(args...)
}

func (l *logger) Debug(args ...interface{}) {
	l._logger.Debug(args...)
}

func (l *logger) Info(args ...interface{}) {
	l._logger.Info(args...)
}

func (l *logger) Warn(args ...interface{}) {
	l._logger.Warn(args...)
}

func (l *logger) Error(args ...interface{}) {
	l._logger.Error(args...)
}
