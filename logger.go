package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"time"
)

const (
	INFO  = "INFO] "
	DEBUG = "DEBUG] "
	WARN  = "WARN] "
	ERROR = "ERROR] "
)

type RotatingFile struct {
	file       *os.File
	maxSize    int64
	currentLen int64
	baseName   string
	mu         sync.Mutex
}

type Logger struct {
	rf          *RotatingFile
	errorLogger *log.Logger
	warnLogger  *log.Logger
	debugLogger *log.Logger
	infoLogger  *log.Logger
	panicLogger *log.Logger
}

func NewRotatingFile(filename string, maxSize int64) (*RotatingFile, error) {
	f, err := os.OpenFile(filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	return &RotatingFile{
		file:       f,
		maxSize:    maxSize,
		currentLen: stat.Size(),
		baseName:   filename,
	}, nil
}

func (rf *RotatingFile) Write(p []byte) (n int, err error) {
	rf.mu.Lock()
	defer rf.mu.Unlock()

	if rf.currentLen+int64(len(p)) >= rf.maxSize {
		if err := rf.rotate(); err != nil {
			return 0, err
		}
	}

	n, err = rf.file.Write(p)
	rf.currentLen += int64(n)
	return n, err
}

func (rf *RotatingFile) rotate() error {
	if rf.file != nil {
		rf.file.Close()
	}

	i := 1
	for {
		newName := fmt.Sprintf("%s.%d", rf.baseName, i)
		if _, err := os.Stat(newName); os.IsNotExist(err) {
			if err := os.Rename(rf.baseName, newName); err != nil {
				return err
			}
			break
		}
		i++
	}

	f, err := os.OpenFile(rf.baseName, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return err
	}

	rf.file = f
	rf.currentLen = 0
	return nil
}

func NewLogger(filename string, maxSize int64) (*Logger, error) {
	rf, err := NewRotatingFile(filename, maxSize)
	if err != nil {
		return nil, err
	}

	multiWriter := io.MultiWriter(rf, os.Stderr)

	return &Logger{
		rf:          rf,
		errorLogger: log.New(multiWriter, "", 0),
		warnLogger:  log.New(multiWriter, "", 0),
		debugLogger: log.New(multiWriter, "", 0),
		infoLogger:  log.New(rf, "", 0),
		panicLogger: log.New(multiWriter, "", 0),
	}, nil
}

func (l *Logger) formatHeader(level string) string {
	now := time.Now()
	return fmt.Sprintf("[PHRYNUS][%s %s][%s",
		now.Format("2006/01/02"),
		now.Format("15:04:05.000000"),
		level)
}

func (l *Logger) log(logger *log.Logger, level string, v ...interface{}) {
	msg := fmt.Sprint(v...)
	logger = log.New(logger.Writer(), l.formatHeader(level), 0)
	logger.Output(3, msg)
}

func (l *Logger) logf(logger *log.Logger, level string, format string, v ...interface{}) {
	msg := fmt.Sprintf(format, v...)
	logger = log.New(logger.Writer(), l.formatHeader(level), 0)
	logger.Output(3, msg)
}

// 常规日志方法
func (l *Logger) Info(v ...interface{})  { l.log(l.infoLogger, INFO, v...) }
func (l *Logger) Debug(v ...interface{}) { l.log(l.debugLogger, DEBUG, v...) }
func (l *Logger) Warn(v ...interface{})  { l.log(l.warnLogger, WARN, v...) }
func (l *Logger) Error(v ...interface{}) {
	s := fmt.Sprint(v...)
	l.log(l.panicLogger, ERROR, s)
	defer func() {
		recover()
	}()
	panic(s)
}

// 格式化日志方法
func (l *Logger) Infof(format string, v ...interface{})  { l.logf(l.infoLogger, INFO, format, v...) }
func (l *Logger) Debugf(format string, v ...interface{}) { l.logf(l.debugLogger, DEBUG, format, v...) }
func (l *Logger) Warnf(format string, v ...interface{})  { l.logf(l.warnLogger, WARN, format, v...) }
func (l *Logger) Errorf(format string, v ...interface{}) {
	s := fmt.Sprintf(format, v...)
	l.logf(l.panicLogger, ERROR, "%s", s)
	defer func() {
		recover()
	}()
	panic(s)
}
