package logger

import (
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var zapLogger *zap.Logger

// LoggerConfig 日志配置
type LoggerConfig struct {
	Level      string // 日志级别: debug, info, warn, error
	Output     string // 输出方式: stdout, file, both
	Path       string // 日志文件路径
	MaxSize    int    // 单个文件最大MB
	MaxBackups int    // 最多保留备份数
	MaxAge     int    // 保留天数
	Compress   bool   // 是否压缩旧日志
}

// Init 初始化日志（简化版本，只输出到stdout）
func Init(level string) {
	InitWithConfig(&LoggerConfig{
		Level:  level,
		Output: "stdout",
	})
}

// InitWithConfig 初始化日志（完整配置）
func InitWithConfig(cfg *LoggerConfig) {
	if cfg == nil {
		cfg = &LoggerConfig{}
	}

	// 设置默认值
	if cfg.Level == "" {
		cfg.Level = "info"
	}
	if cfg.Output == "" {
		cfg.Output = "stdout"
	}
	if cfg.Path == "" {
		cfg.Path = "./logs/fastdp-orbit.log"
	}
	if cfg.MaxSize == 0 {
		cfg.MaxSize = 100
	}
	if cfg.MaxBackups == 0 {
		cfg.MaxBackups = 30
	}
	if cfg.MaxAge == 0 {
		cfg.MaxAge = 30
	}

	// 解析日志级别
	var zapLevel zapcore.Level
	if err := zapLevel.UnmarshalText([]byte(cfg.Level)); err != nil {
		zapLevel = zapcore.InfoLevel
	}

	// 创建输出同步器
	var writeSyncer zapcore.WriteSyncer
	switch cfg.Output {
	case "file":
		writeSyncer = zapcore.AddSync(createFileWriter(cfg))
	case "both":
		fileSyncer := zapcore.AddSync(createFileWriter(cfg))
		consoleSyncer := zapcore.AddSync(os.Stdout)
		writeSyncer = zapcore.NewMultiWriteSyncer(fileSyncer, consoleSyncer)
	default: // stdout
		writeSyncer = zapcore.AddSync(os.Stdout)
	}

	// 编码器配置
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder, // 大写级别: INFO, ERROR
		EncodeTime:     customTimeEncoder,           // 自定义时间格式
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	// 创建编码器
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	// 创建核心
	core := zapcore.NewCore(
		encoder,
		writeSyncer,
		zap.NewAtomicLevelAt(zapLevel),
	)

	// 创建logger
	zapLogger = zap.New(core,
		zap.AddCaller(),
		zap.AddStacktrace(zap.ErrorLevel), // 错误级别添加堆栈
	)
}

// createFileWriter 创建文件写入器（支持日志切割）
func createFileWriter(cfg *LoggerConfig) *lumberjack.Logger {
	return &lumberjack.Logger{
		Filename:   cfg.Path,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	}
}

// customTimeEncoder 自定义时间格式
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// Sync 同步日志缓冲区
func Sync() {
	if zapLogger != nil {
		_ = zapLogger.Sync()
	}
}

// Info 记录Info级别日志
func Info(msg string, fields ...zap.Field) {
	zapLogger.Info(msg, fields...)
}

// Error 记录Error级别日志
func Error(msg string, fields ...zap.Field) {
	zapLogger.Error(msg, fields...)
}

// Debug 记录Debug级别日志
func Debug(msg string, fields ...zap.Field) {
	zapLogger.Debug(msg, fields...)
}

// Warn 记录Warn级别日志
func Warn(msg string, fields ...zap.Field) {
	zapLogger.Warn(msg, fields...)
}

// Fatal 记录Fatal级别日志（会终止程序）
func Fatal(msg string, fields ...zap.Field) {
	zapLogger.Fatal(msg, fields...)
}

// GetLogger 获取底层的zap.Logger（供ginzap等需要使用）
func GetLogger() *zap.Logger {
	return zapLogger
}
