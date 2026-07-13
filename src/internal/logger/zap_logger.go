package logger

import (
	"fmt"
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// *-------------------------------------------------------------*//
// New
// *-------------------------------------------------------------*//
func New(logDir string, maxFiles int, logLevel string, isDevelopment bool) (*zap.Logger, error) {
	// ログディレクトリが存在しない場合は作成
	if err := os.MkdirAll(logDir, os.ModePerm); err != nil {
		return nil, fmt.Errorf("failed to create log directory %s: %w", logDir, err)
	}

	// lumberjackによるログローテーションの設定
	w := &lumberjack.Logger{
		// Filename:   logDir + "/" + time.Now().Format("20060102-150405") + ".log",
		Filename:   logDir + "/app.log",
		MaxSize:    10,       // MB
		MaxBackups: maxFiles, // files
		MaxAge:     30,       // days
		Compress:   true,     // disabled by default
	}

	// ログレベルの設定
	level, err := zapcore.ParseLevel(logLevel)
	if err != nil {
		level = zapcore.InfoLevel
	}

	// zapのエンコーダー
	var encoderConfig zapcore.EncoderConfig
	if isDevelopment {
		encoderConfig = zap.NewDevelopmentEncoderConfig()
	} else {
		encoderConfig = zap.NewProductionEncoderConfig()
	}
	encoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

	// コンソール出力用のコア
	consoleEncoder := zapcore.NewConsoleEncoder(encoderConfig)
	consoleCore := zapcore.NewCore(
		consoleEncoder,
		zapcore.AddSync(os.Stdout),
		level,
	)

	// ファイル出力用のコア
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
	fileCore := zapcore.NewCore(
		fileEncoder,
		zapcore.AddSync(w),
		level,
	)

	// 複数のコアを組み合わせる
	core := zapcore.NewTee(consoleCore, fileCore)

	// ロガーの作成
	logger := zap.New(core, zap.AddCaller())
	return logger, nil
}
