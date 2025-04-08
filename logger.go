package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fatih/color"
)

const (
	INFO  = 0
	DEBUG = 1
	WARN  = 2
	ERROR = 3
)

var levelNames = []string{
	"INFO ",
	"DEBUG",
	"WARN ",
	"ERROR",
}

type LogConfig struct {
	Filename       string
	MaxSize        int64 // bytes
	BufferSize     int   // lines
	FlushInterval  time.Duration
	StdoutLevels   map[int]bool
	FileWriterOnly bool
	ColorOutput    bool // Whether to use colored output
}

type RotatingWriter struct {
	logConfig  LogConfig
	file       *os.File
	currentLen int64
	rotateCh   chan struct{}
	mu         sync.RWMutex
}

type Logger struct {
	writer    *RotatingWriter
	logChan   chan *logEntry
	closeCh   chan struct{}
	wg        sync.WaitGroup
	stdoutMap map[int]bool
	colorMap  map[int]*color.Color
}

type logEntry struct {
	level  int
	format string
	args   []interface{}
}

func LoggerNew(logConfig LogConfig) (*Logger, error) {
	rw := &RotatingWriter{
		logConfig: logConfig,
		rotateCh:  make(chan struct{}, 1),
	}

	if !logConfig.FileWriterOnly {
		if err := rw.initializeFile(); err != nil {
			return nil, err
		}
	}

	l := &Logger{
		writer:    rw,
		logChan:   make(chan *logEntry, logConfig.BufferSize),
		closeCh:   make(chan struct{}),
		stdoutMap: make(map[int]bool),
		colorMap: map[int]*color.Color{
			INFO:  color.BgRGB(39, 174, 96).AddRGB(255, 255, 255), // 绿色背景，白色文字
			DEBUG: color.BgRGB(55, 66, 250).AddRGB(255, 255, 255), // 蓝色背景，白色文字
			WARN:  color.BgRGB(255, 128, 0).AddRGB(255, 255, 255), // 黄色背景，白色文字
			ERROR: color.BgRGB(231, 76, 60).AddRGB(255, 255, 255), // 红色背景，白色文字
			4:     color.RGB(255, 255, 255),                       // Title 白色背景，黑色文字
			5:     color.RGB(255, 255, 255),                       // Time 白色背景，蓝色文字
		},
	}

	for level := range logConfig.StdoutLevels {
		l.stdoutMap[level] = true
	}

	l.wg.Add(1)
	go l.processLogs()

	if logConfig.FlushInterval > 0 {
		l.wg.Add(1)
		go l.autoFlush()
	}

	return l, nil
}

func (rw *RotatingWriter) initializeFile() error {
	dir := filepath.Dir(rw.logConfig.Filename)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create log directory failed: %w", err)
	}

	f, err := os.OpenFile(rw.logConfig.Filename, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("open log file failed: %w", err)
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return fmt.Errorf("get file stats failed: %w", err)
	}

	rw.file = f
	rw.currentLen = stat.Size()
	return nil
}

func (rw *RotatingWriter) Write(p []byte) (int, error) {
	rw.mu.RLock()
	defer rw.mu.RUnlock()

	if rw.file == nil {
		return 0, fmt.Errorf("log file not initialized")
	}

	if rw.currentLen+int64(len(p)) > rw.logConfig.MaxSize {
		select {
		case rw.rotateCh <- struct{}{}:
			go rw.safeRotate()
		default:
			// Rotation already in progress
		}
	}

	n, err := rw.file.Write(p)
	rw.currentLen += int64(n)
	return n, err
}

func (rw *RotatingWriter) safeRotate() {
	rw.mu.Lock()
	defer rw.mu.Unlock()

	if rw.file == nil {
		return
	}

	now := time.Now().UnixNano()
	newName := fmt.Sprintf("%s.%d", rw.logConfig.Filename, now)
	if err := os.Rename(rw.logConfig.Filename, newName); err != nil {
		log.Printf("log rotate failed: %v", err)
		return
	}

	if err := rw.initializeFile(); err != nil {
		log.Printf("reopen log file failed: %v", err)
	}
}

func (l *Logger) processLogs() {
	defer l.wg.Done()

	var stdout io.Writer = os.Stderr
	if l.writer.logConfig.FileWriterOnly {
		stdout = io.Discard
	}

	for {
		select {
		case entry := <-l.logChan:
			l.writeLog(entry, stdout)
		case <-l.closeCh:
			// Flush remaining logs
			for {
				select {
				case entry := <-l.logChan:
					l.writeLog(entry, stdout)
				default:
					return
				}
			}
		}
	}
}

func (l *Logger) writeLog(entry *logEntry, stdout io.Writer) {
	msg := l.formatMessage(entry)
	if _, ok := l.stdoutMap[entry.level]; ok {
		if l.writer.logConfig.ColorOutput && stdout == os.Stderr {
			fmt.Fprintln(stdout, msg) // Color is already applied in formatMessage
		} else {
			fmt.Fprintln(stdout, msg)
		}
	}

	if l.writer.logConfig.FileWriterOnly {
		return
	}

	// Write plain text to file (without color codes)
	plainMsg := l.formatPlainMessage(entry)
	if _, err := l.writer.Write([]byte(plainMsg + "\n")); err != nil {
		log.Printf("log write failed: %v", err)
	}

	if entry.level == ERROR {
		os.Exit(1)
	}
}

func (l *Logger) formatPlainMessage(entry *logEntry) string {
	now := time.Now()
	var msg string

	if entry.format == "" {
		msg = fmt.Sprint(entry.args...)
	} else {
		msg = fmt.Sprintf(entry.format, entry.args...)
	}
	if entry.level == -1 {
		return fmt.Sprintf("%s", msg)
	}

	return fmt.Sprintf("[PHRYNUS][%s %s][%s] %s",
		now.Format("2006/01/02"),
		now.Format("15:04:05.000000"),
		levelNames[entry.level],
		msg)
}

func (l *Logger) formatMessage(entry *logEntry) string {
	now := time.Now()
	var msg string

	if entry.format == "" {
		msg = fmt.Sprint(entry.args...)
	} else {
		msg = fmt.Sprintf(entry.format, entry.args...)
	}
	if entry.level == -1 {
		return fmt.Sprintf("%s", msg)
	}

	if l.writer.logConfig.ColorOutput {
		levelPart := fmt.Sprintf("[%s]", levelNames[entry.level])
		coloredLevel := l.colorMap[entry.level].Sprint(levelPart)
		timePart := fmt.Sprintf("[%s %s]", now.Format("2006/01/02"), now.Format("15:04:05.000000"))
		coloredTime := l.colorMap[5].Sprint(timePart)
		return fmt.Sprintf("%s%s%s %s",
			l.colorMap[4].Sprint("[PHRYNUS]"),
			coloredTime,
			coloredLevel,
			msg)
	}

	return fmt.Sprintf("[PHRYNUS][%s %s][%s] %s",
		now.Format("2006/01/02"),
		now.Format("15:04:05.000000"),
		levelNames[entry.level],
		msg)
}

func (l *Logger) autoFlush() {
	defer l.wg.Done()

	ticker := time.NewTicker(l.writer.logConfig.FlushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			l.Flush()
		case <-l.closeCh:
			return
		}
	}
}

// Public Methods
func (l *Logger) Log(level int, args ...interface{}) {
	l.log(level, "", args...)
}

func (l *Logger) Logf(level int, format string, args ...interface{}) {
	l.log(level, format, args...)
}

func (l *Logger) Debug(args ...interface{}) { l.log(DEBUG, "", args...) }
func (l *Logger) Info(args ...interface{})  { l.log(INFO, "", args...) }
func (l *Logger) Warn(args ...interface{})  { l.log(WARN, "", args...) }
func (l *Logger) Error(args ...interface{}) { l.log(ERROR, "", args...) }

func (l *Logger) T(args ...interface{}) { l.log(-1, "", args...) }

func (l *Logger) Debugf(format string, args ...interface{}) { l.log(DEBUG, format, args...) }
func (l *Logger) Infof(format string, args ...interface{})  { l.log(INFO, format, args...) }
func (l *Logger) Warnf(format string, args ...interface{})  { l.log(WARN, format, args...) }
func (l *Logger) Errorf(format string, args ...interface{}) { l.log(ERROR, format, args...) }

func (l *Logger) log(level int, format string, args ...interface{}) {
	entry := &logEntry{
		level:  level,
		format: format,
		args:   args,
	}

	select {
	case l.logChan <- entry:
	default:
		// 缓冲满时直接写入避免阻塞
		l.writeLog(entry, os.Stderr)
	}
}

func (l *Logger) Flush() {
	for len(l.logChan) > 0 {
		time.Sleep(5 * time.Millisecond)
	}
}

func (l *Logger) Close() {
	close(l.closeCh)
	l.wg.Wait()

	if l.writer.file != nil {
		l.writer.file.Close()
	}
}
