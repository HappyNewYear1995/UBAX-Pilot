package logger

import (
	"io"
	"log"
	"os"
)

// Logger 封装标准日志库，支持分级日志
type Logger struct {
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
	debug *log.Logger
}

var defaultLogger *Logger

func init() {
	defaultLogger = NewLogger(os.Stdout, "[UBAX-Pilot] ")
}

// NewLogger 创建一个新的 Logger 实例，写入指定的 io.Writer
func NewLogger(w io.Writer, prefix string) *Logger {
	return &Logger{
		info:  log.New(w, prefix+"INFO:  ", log.Ldate|log.Ltime|log.Lshortfile),
		warn:  log.New(w, prefix+"WARN:  ", log.Ldate|log.Ltime|log.Lshortfile),
		error: log.New(w, prefix+"ERROR: ", log.Ldate|log.Ltime|log.Lshortfile),
		debug: log.New(w, prefix+"DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
}

// Info 打印信息级别日志
func Info(v ...interface{}) {
	defaultLogger.info.Println(v...)
}

// Infof 格式化打印信息级别日志
func Infof(format string, v ...interface{}) {
	defaultLogger.info.Printf(format, v...)
}

// Warn 打印警告级别日志
func Warn(v ...interface{}) {
	defaultLogger.warn.Println(v...)
}

// Warnf 格式化打印警告级别日志
func Warnf(format string, v ...interface{}) {
	defaultLogger.warn.Printf(format, v...)
}

// Error 打印错误级别日志
func Error(v ...interface{}) {
	defaultLogger.error.Println(v...)
}

// Errorf 格式化打印错误级别日志
func Errorf(format string, v ...interface{}) {
	defaultLogger.error.Printf(format, v...)
}

// Debug 打印调试级别日志
func Debug(v ...interface{}) {
	defaultLogger.debug.Println(v...)
}

// Debugf 格式化打印调试级别日志
func Debugf(format string, v ...interface{}) {
	defaultLogger.debug.Printf(format, v...)
}

// GetLogger 返回默认日志器实例
func GetLogger() *Logger {
	return defaultLogger
}
