package logger

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

var defaultLogger *Logger
var logLevel LogLevel
var logFile *os.File
var mu sync.Mutex

func init() {
	defaultLogger = NewLogger(os.Stdout, "[UBAX-Pilot] ")
	logLevel = INFO
}

// Logger 封装标准日志库，支持分级日志
type Logger struct {
	info  *log.Logger
	warn  *log.Logger
	error *log.Logger
	debug *log.Logger
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

// InitLogger 初始化日志器，设置日志级别和输出文件
func InitLogger(level string, filePath string) error {
	mu.Lock()
	defer mu.Unlock()

	switch strings.ToLower(level) {
	case "debug":
		logLevel = DEBUG
	case "info":
		logLevel = INFO
	case "warn":
		logLevel = WARN
	case "error":
		logLevel = ERROR
	default:
		logLevel = INFO
	}

	var writers []io.Writer
	writers = append(writers, os.Stdout)

	if filePath != "" {
		dir := filepath.Dir(filePath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}

		f, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return err
		}

		if logFile != nil {
			_ = logFile.Close()
		}
		logFile = f
		writers = append(writers, f)
	}

	multiWriter := io.MultiWriter(writers...)
	defaultLogger = NewLogger(multiWriter, "[UBAX-Pilot] ")
	return nil
}

// Info 打印信息级别日志
func Info(v ...interface{}) {
	if logLevel <= INFO {
		defaultLogger.info.Println(v...)
	}
}

// Infof 格式化打印信息级别日志
func Infof(format string, v ...interface{}) {
	if logLevel <= INFO {
		defaultLogger.info.Printf(format, v...)
	}
}

// Warn 打印警告级别日志
func Warn(v ...interface{}) {
	if logLevel <= WARN {
		defaultLogger.warn.Println(v...)
	}
}

// Warnf 格式化打印警告级别日志
func Warnf(format string, v ...interface{}) {
	if logLevel <= WARN {
		defaultLogger.warn.Printf(format, v...)
	}
}

// Error 打印错误级别日志
func Error(v ...interface{}) {
	if logLevel <= ERROR {
		defaultLogger.error.Println(v...)
	}
}

// Errorf 格式化打印错误级别日志
func Errorf(format string, v ...interface{}) {
	if logLevel <= ERROR {
		defaultLogger.error.Printf(format, v...)
	}
}

// Debug 打印调试级别日志
func Debug(v ...interface{}) {
	if logLevel <= DEBUG {
		defaultLogger.debug.Println(v...)
	}
}

// Debugf 格式化打印调试级别日志
func Debugf(format string, v ...interface{}) {
	if logLevel <= DEBUG {
		defaultLogger.debug.Printf(format, v...)
	}
}

// GetLogger 返回默认日志器实例
func GetLogger() *Logger {
	return defaultLogger
}

// Close 关闭日志文件
func Close() {
	mu.Lock()
	defer mu.Unlock()
	if logFile != nil {
		_ = logFile.Close()
		logFile = nil
	}
}
