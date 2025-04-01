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
	INFO  = "INFO]  "
	DEBUG = "DEBUG] "
	WARN  = "WARN]  "
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

	// 查找可用的分割文件序号
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

	// 创建多目标Writer
	multiWriter := io.MultiWriter(rf, os.Stderr)

	return &Logger{
		rf:          rf,
		errorLogger: log.New(multiWriter, "", 0),
		warnLogger:  log.New(multiWriter, "", 0),
		debugLogger: log.New(multiWriter, "", 0),
		infoLogger:  log.New(rf, "", 0),
	}, nil
}

func (l *Logger) formatHeader(level string) string {
	now := time.Now()
	return fmt.Sprintf("[PHRYNUS][%s %s][%s",
		now.Format("2006/01/02"),
		now.Format("15:04:05.000"),
		level)
}

func (l *Logger) log(logger *log.Logger, level, msg string) {
	logger.Output(3, l.formatHeader(level)+msg)
}

func (l *Logger) Info(msg string)  { l.log(l.infoLogger, INFO, msg) }
func (l *Logger) Debug(msg string) { l.log(l.debugLogger, DEBUG, msg) }
func (l *Logger) Warn(msg string)  { l.log(l.warnLogger, WARN, msg) }
func (l *Logger) Error(msg string) { l.log(l.errorLogger, ERROR, msg) }
