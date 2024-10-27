package logger

import (
	"io"
	"log"
	"os"
)

type Log struct {
	logError *log.Logger
	logInfo  *log.Logger
	logDebug *log.Logger
}

func New(logLevel string, prefix string) *Log {
	logError := log.New(io.Discard, "ERROR: "+prefix, log.Ldate|log.Ltime)
	logInfo := log.New(io.Discard, "INFO: "+prefix, log.Ldate|log.Ltime)
	logDebug := log.New(io.Discard, "DEBUG: "+prefix, log.Ldate|log.Ltime)

	switch logLevel {
	case "DEBUG":
		logDebug.SetOutput(os.Stdout)
		fallthrough
	case "INFO":
		logInfo.SetOutput(os.Stdout)
		fallthrough
	case "ERROR":
		logError.SetOutput(os.Stderr)
	}

	return &Log{
		logError: logError,
		logInfo:  logInfo,
		logDebug: logDebug,
	}
}

func (l *Log) Info(format string, args ...interface{}) {
	l.logInfo.Printf(format, args...)
}

func (l *Log) Debug(format string, args ...interface{}) {
	l.logDebug.Printf(format, args...)
}

func (l *Log) Error(format string, args ...interface{}) {
	l.logError.Printf(format, args...)
}
