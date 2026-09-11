package ui

import (
	"os"

	"github.com/charmbracelet/log"
)

// Logger is the global charmbracelet logger instance for dbctl.
var Logger *log.Logger

func init() {
	Logger = log.NewWithOptions(os.Stderr, log.Options{
		ReportTimestamp: true,
		Prefix:          "dbctl",
	})
}

// SetVerbose enables debug-level logging.
func SetVerbose() {
	Logger.SetLevel(log.DebugLevel)
}

// SetQuiet disables all output below error level.
func SetQuiet() {
	Logger.SetLevel(log.ErrorLevel)
}

// Convenience wrappers that delegate to the global logger.

func Debug(msg interface{}, keyvals ...interface{}) {
	Logger.Debug(msg, keyvals...)
}

func Info(msg interface{}, keyvals ...interface{}) {
	Logger.Info(msg, keyvals...)
}

func Warn(msg interface{}, keyvals ...interface{}) {
	Logger.Warn(msg, keyvals...)
}

func Error(msg interface{}, keyvals ...interface{}) {
	Logger.Error(msg, keyvals...)
}

func Fatal(msg interface{}, keyvals ...interface{}) {
	Logger.Fatal(msg, keyvals...)
}
