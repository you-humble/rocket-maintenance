package logger

import (
	"go.uber.org/zap"
)

const (
	LevelDebug = zap.DebugLevel
	LevelInfo  = zap.InfoLevel
	LevelWarn  = zap.WarnLevel
	LevelError = zap.ErrorLevel
	LevelFatal = zap.FatalLevel
)

var (
	String   = zap.String
	Int      = zap.Int
	Duration = zap.Duration
	Bool     = zap.Bool
	ErrorF   = zap.Error
	Any      = zap.Any
)

type (
	Field = zap.Field
)
