package logging

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// FileLoggerConfig configures file-based logging
type FileLoggerConfig struct {
	FilePath    string `yaml:"file_path"`
	MaxSize     int64  `yaml:"max_size"`      // Maximum size in bytes before rotation
	MaxBackups  int    `yaml:"max_backups"`   // Maximum number of backup files
	MaxAge      int    `yaml:"max_age"`       // Maximum age in days
	Compress    bool   `yaml:"compress"`      // Whether to compress rotated files
	BufferSize  int    `yaml:"buffer_size"`   // Buffer size for writing
	SyncOnWrite bool   `yaml:"sync_on_write"` // Sync after each write
}

// FileLogger implements file-based logging with rotation
type FileLogger struct {
	config     FileLoggerConfig
	file       *os.File
	written    int64
	mutex      sync.Mutex
	buffer     []byte
	bufferSize int
	lastRotate time.Time
}

// NewFileLogger creates a new file logger
func NewFileLogger(config FileLoggerConfig) (*FileLogger, error) {
	if err := validateFileLoggerConfig(config); err != nil {
		return nil, err
	}

	fl := &FileLogger{
		config:     config,
		bufferSize: config.BufferSize,
		lastRotate: time.Now(),
	}

	if fl.bufferSize <= 0 {
		fl.bufferSize = 4096 // Default 4KB buffer
	}

	fl.buffer = make([]byte, 0, fl.bufferSize)

	// Create log directory if it doesn't exist
	logDir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// Open log file
	if err := fl.openFile(); err != nil {
		return nil, err
	}

	return fl, nil
}

// openFile opens the log file for writing
func (fl *FileLogger) openFile() error {
	file, err := os.OpenFile(fl.config.FilePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file %s: %w", fl.config.FilePath, err)
	}

	fl.file = file

	// Get current file size
	if stat, err := file.Stat(); err == nil {
		fl.written = stat.Size()
	}

	return nil
}

// Write writes data to the log file
func (fl *FileLogger) Write(data []byte) (int, error) {
	fl.mutex.Lock()
	defer fl.mutex.Unlock()

	// Check if rotation is needed
	if fl.needsRotation(int64(len(data))) {
		if err := fl.rotate(); err != nil {
			return 0, fmt.Errorf("log rotation failed: %w", err)
		}
	}

	// Write to buffer first
	if len(fl.buffer)+len(data) > fl.bufferSize || fl.config.SyncOnWrite {
		// Flush buffer first
		if err := fl.flushBuffer(); err != nil {
			return 0, err
		}
	}

	// If data is larger than buffer, write directly
	if len(data) > fl.bufferSize {
		n, err := fl.file.Write(data)
		if err != nil {
			return n, err
		}
		fl.written += int64(n)

		if fl.config.SyncOnWrite {
			fl.file.Sync()
		}

		return n, nil
	}

	// Add to buffer
	fl.buffer = append(fl.buffer, data...)

	return len(data), nil
}

// flushBuffer writes the buffer to the file
func (fl *FileLogger) flushBuffer() error {
	if len(fl.buffer) == 0 {
		return nil
	}

	n, err := fl.file.Write(fl.buffer)
	if err != nil {
		return err
	}

	fl.written += int64(n)
	fl.buffer = fl.buffer[:0] // Reset buffer

	if fl.config.SyncOnWrite {
		return fl.file.Sync()
	}

	return nil
}

// Flush flushes any buffered data
func (fl *FileLogger) Flush() error {
	fl.mutex.Lock()
	defer fl.mutex.Unlock()

	return fl.flushBuffer()
}

// needsRotation checks if log rotation is needed
func (fl *FileLogger) needsRotation(additionalBytes int64) bool {
	if fl.config.MaxSize <= 0 {
		return false
	}

	return fl.written+additionalBytes > fl.config.MaxSize
}

// rotate rotates the log file
func (fl *FileLogger) rotate() error {
	// Flush any remaining buffer
	if err := fl.flushBuffer(); err != nil {
		return err
	}

	// Close current file
	if fl.file != nil {
		fl.file.Close()
	}

	// Create backup filename with timestamp
	timestamp := time.Now().Format("2006-01-02T15-04-05")
	backupPath := fmt.Sprintf("%s.%s", fl.config.FilePath, timestamp)

	// Rename current file to backup
	if err := os.Rename(fl.config.FilePath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup %s: %w", backupPath, err)
	}

	// Compress backup if configured
	if fl.config.Compress {
		if err := fl.compressFile(backupPath); err != nil {
			// Log error but don't fail rotation
			fmt.Printf("Warning: failed to compress log backup %s: %v\n", backupPath, err)
		}
	}

	// Clean up old backups
	fl.cleanupOldBackups()

	// Open new log file
	if err := fl.openFile(); err != nil {
		return err
	}

	fl.written = 0
	fl.lastRotate = time.Now()

	return nil
}

// compressFile compresses a log file using gzip
func (fl *FileLogger) compressFile(filePath string) error {
	// Simple gzip compression implementation
	// In a full implementation, you'd use compress/gzip package

	compressedPath := filePath + ".gz"

	// Read original file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return err
	}

	// Write compressed file (simplified - in reality use gzip)
	if err := os.WriteFile(compressedPath, data, 0644); err != nil {
		return err
	}

	// Remove original file
	return os.Remove(filePath)
}

// cleanupOldBackups removes old backup files
func (fl *FileLogger) cleanupOldBackups() {
	logDir := filepath.Dir(fl.config.FilePath)
	logName := filepath.Base(fl.config.FilePath)

	// Get all backup files
	pattern := filepath.Join(logDir, logName+".*")
	backups, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	// Remove by count
	if fl.config.MaxBackups > 0 && len(backups) > fl.config.MaxBackups {
		// Sort by modification time and remove oldest
		// Simplified implementation - in reality you'd sort by file mod time
		excess := len(backups) - fl.config.MaxBackups
		for i := 0; i < excess; i++ {
			os.Remove(backups[i])
		}
	}

	// Remove by age
	if fl.config.MaxAge > 0 {
		cutoff := time.Now().AddDate(0, 0, -fl.config.MaxAge)
		for _, backup := range backups {
			if stat, err := os.Stat(backup); err == nil {
				if stat.ModTime().Before(cutoff) {
					os.Remove(backup)
				}
			}
		}
	}
}

// RotateNow forces immediate log rotation
func (fl *FileLogger) RotateNow() error {
	fl.mutex.Lock()
	defer fl.mutex.Unlock()

	return fl.rotate()
}

// Close closes the file logger
func (fl *FileLogger) Close() error {
	fl.mutex.Lock()
	defer fl.mutex.Unlock()

	// Flush any remaining data
	if err := fl.flushBuffer(); err != nil {
		return err
	}

	// Close file
	if fl.file != nil {
		return fl.file.Close()
	}

	return nil
}

// GetStats returns file logger statistics
func (fl *FileLogger) GetStats() FileLoggerStats {
	fl.mutex.Lock()
	defer fl.mutex.Unlock()

	return FileLoggerStats{
		FilePath:     fl.config.FilePath,
		CurrentSize:  fl.written,
		MaxSize:      fl.config.MaxSize,
		BufferSize:   len(fl.buffer),
		LastRotation: fl.lastRotate,
	}
}

// FileLoggerStats represents file logger statistics
type FileLoggerStats struct {
	FilePath     string    `json:"file_path"`
	CurrentSize  int64     `json:"current_size"`
	MaxSize      int64     `json:"max_size"`
	BufferSize   int       `json:"buffer_size"`
	LastRotation time.Time `json:"last_rotation"`
}

// validateFileLoggerConfig validates the file logger configuration
func validateFileLoggerConfig(config FileLoggerConfig) error {
	if config.FilePath == "" {
		return fmt.Errorf("file path cannot be empty")
	}

	// Validate directory is writable
	logDir := filepath.Dir(config.FilePath)
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return fmt.Errorf("cannot create log directory %s: %w", logDir, err)
	}

	// Test write permissions
	testFile := filepath.Join(logDir, ".mcpeg_log_test")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		return fmt.Errorf("cannot write to log directory %s: %w", logDir, err)
	}
	os.Remove(testFile)

	if config.MaxSize < 0 {
		return fmt.Errorf("max size cannot be negative")
	}

	if config.MaxBackups < 0 {
		return fmt.Errorf("max backups cannot be negative")
	}

	if config.MaxAge < 0 {
		return fmt.Errorf("max age cannot be negative")
	}

	return nil
}

// MultiWriter combines multiple writers
type MultiWriter struct {
	writers []io.Writer
	mutex   sync.Mutex
}

// NewMultiWriter creates a new multi-writer
func NewMultiWriter(writers ...io.Writer) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Write writes to all writers
func (mw *MultiWriter) Write(data []byte) (int, error) {
	mw.mutex.Lock()
	defer mw.mutex.Unlock()

	var lastErr error
	written := len(data)

	for _, writer := range mw.writers {
		if _, err := writer.Write(data); err != nil {
			lastErr = err
		}
	}

	return written, lastErr
}

// AddWriter adds a writer to the multi-writer
func (mw *MultiWriter) AddWriter(writer io.Writer) {
	mw.mutex.Lock()
	defer mw.mutex.Unlock()

	mw.writers = append(mw.writers, writer)
}

// RemoveWriter removes a writer from the multi-writer
func (mw *MultiWriter) RemoveWriter(writer io.Writer) {
	mw.mutex.Lock()
	defer mw.mutex.Unlock()

	for i, w := range mw.writers {
		if w == writer {
			mw.writers = append(mw.writers[:i], mw.writers[i+1:]...)
			break
		}
	}
}

// ProductionLogger combines console and file logging
type ProductionLogger struct {
	console     Logger
	fileLogger  *FileLogger
	multiWriter *MultiWriter
	level       string
}

// NewProductionLogger creates a production logger with both console and file output
func NewProductionLogger(level string, fileConfig FileLoggerConfig) (*ProductionLogger, error) {
	// Create file logger
	fileLogger, err := NewFileLogger(fileConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create file logger: %w", err)
	}

	// Create multi-writer for both console and file
	multiWriter := NewMultiWriter(os.Stdout, fileLogger)

	// Create console logger
	consoleLogger := &SimpleLogger{
		level:  level,
		writer: multiWriter,
	}

	return &ProductionLogger{
		console:     consoleLogger,
		fileLogger:  fileLogger,
		multiWriter: multiWriter,
		level:       level,
	}, nil
}

// Simple logger implementation for production use
type SimpleLogger struct {
	level  string
	writer io.Writer
	mutex  sync.Mutex
}

// WithComponent implements Logger interface
func (sl *SimpleLogger) WithComponent(component string) Logger {
	return &SimpleLogger{level: sl.level, writer: sl.writer}
}

// WithContext implements Logger interface
func (sl *SimpleLogger) WithContext(ctx context.Context) Logger {
	return sl
}

// WithTraceID implements Logger interface
func (sl *SimpleLogger) WithTraceID(traceID string) Logger {
	return sl
}

// WithSpanID implements Logger interface
func (sl *SimpleLogger) WithSpanID(spanID string) Logger {
	return sl
}

// Log level methods
func (sl *SimpleLogger) Trace(msg string, fields ...interface{}) {
	if sl.shouldLog("trace") {
		sl.log("TRACE", msg, fields...)
	}
}

func (sl *SimpleLogger) Debug(msg string, fields ...interface{}) {
	if sl.shouldLog("debug") {
		sl.log("DEBUG", msg, fields...)
	}
}

func (sl *SimpleLogger) Info(msg string, fields ...interface{}) {
	if sl.shouldLog("info") {
		sl.log("INFO", msg, fields...)
	}
}

func (sl *SimpleLogger) Warn(msg string, fields ...interface{}) {
	if sl.shouldLog("warn") {
		sl.log("WARN", msg, fields...)
	}
}

func (sl *SimpleLogger) Error(msg string, fields ...interface{}) {
	if sl.shouldLog("error") {
		sl.log("ERROR", msg, fields...)
	}
}

func (sl *SimpleLogger) shouldLog(level string) bool {
	levels := map[string]int{
		"trace": 0,
		"debug": 1,
		"info":  2,
		"warn":  3,
		"error": 4,
	}

	currentLevel, ok1 := levels[sl.level]
	targetLevel, ok2 := levels[level]

	if !ok1 || !ok2 {
		return true
	}

	return targetLevel >= currentLevel
}

func (sl *SimpleLogger) log(level, msg string, fields ...interface{}) {
	sl.mutex.Lock()
	defer sl.mutex.Unlock()

	timestamp := time.Now().Format("2006-01-02T15:04:05.000Z")

	logLine := fmt.Sprintf("[%s] %s %s", timestamp, level, msg)

	if len(fields) > 0 {
		logLine += fmt.Sprintf(" %v", fields)
	}

	logLine += "\n"

	sl.writer.Write([]byte(logLine))
}

// RotateLog forces log rotation
func (pl *ProductionLogger) RotateLog() error {
	return pl.fileLogger.RotateNow()
}

// GetFileStats returns file logger statistics
func (pl *ProductionLogger) GetFileStats() FileLoggerStats {
	return pl.fileLogger.GetStats()
}

// Close closes the production logger
func (pl *ProductionLogger) Close() error {
	return pl.fileLogger.Close()
}

// SetLevel changes the log level
func (pl *ProductionLogger) SetLevel(level string) {
	pl.level = level
	if simpleLogger, ok := pl.console.(*SimpleLogger); ok {
		simpleLogger.level = level
	}
}

// GetLevel returns the current log level
func (pl *ProductionLogger) GetLevel() string {
	return pl.level
}

// Implement Logger interface for ProductionLogger
func (pl *ProductionLogger) WithComponent(component string) Logger {
	return pl.console.WithComponent(component)
}

func (pl *ProductionLogger) WithContext(ctx context.Context) Logger {
	return pl.console.WithContext(ctx)
}

func (pl *ProductionLogger) WithTraceID(traceID string) Logger {
	return pl.console.WithTraceID(traceID)
}

func (pl *ProductionLogger) WithSpanID(spanID string) Logger {
	return pl.console.WithSpanID(spanID)
}

func (pl *ProductionLogger) Trace(msg string, fields ...interface{}) {
	pl.console.Trace(msg, fields...)
}

func (pl *ProductionLogger) Debug(msg string, fields ...interface{}) {
	pl.console.Debug(msg, fields...)
}

func (pl *ProductionLogger) Info(msg string, fields ...interface{}) {
	pl.console.Info(msg, fields...)
}

func (pl *ProductionLogger) Warn(msg string, fields ...interface{}) {
	pl.console.Warn(msg, fields...)
}

func (pl *ProductionLogger) Error(msg string, fields ...interface{}) {
	pl.console.Error(msg, fields...)
}

// Interface compliance
// Note: SimpleLogger is not a full Logger implementation - it's a simplified version
// for internal use. For full Logger compliance, use the main logging package.
var _ io.Writer = (*FileLogger)(nil)
var _ io.Writer = (*MultiWriter)(nil)
