package log

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	kratoszap "github.com/go-kratos/kratos/contrib/log/zap/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/shengyanli1982/law"
	"go.uber.org/zap"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

var ColorResetStr = "\x1b[0m"
var LenColorResetStr = len(ColorResetStr)

var (
	Red    = color.New(color.FgHiRed).SprintFunc()
	Blue   = color.New(color.FgHiBlue).SprintFunc()
	Yellow = color.New(color.FgHiYellow).SprintFunc()
	Green  = color.New(color.FgHiGreen).SprintFunc()
)

type Logger struct {
	*zap.Logger
}

var globalLogger *Logger

func NewLogger(
	mode int,
	level string,
	logDir string,
	appName string,
) (*Logger, error) {
	logPath := GetLogPath(logDir, appName)

	// 创建日志目录
	if logPath != "" {
		dir := filepath.Dir(logPath)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("创建日志目录失败：%w", err)
		}
	}

	// 解析日志级别
	zLevel, err := zapcore.ParseLevel(level)
	if err != nil {
		return nil, err
	}

	// 编码器配置（带颜色）
	_ = zapcore.EncoderConfig{
		TimeKey:  "T",
		LevelKey: "L",
		NameKey:  "N",
		//CallerKey:        "C",
		MessageKey:       "M",
		StacktraceKey:    "S",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      customLevelColorEncoder,
		EncodeTime:       customTimeEncoder,
		EncodeDuration:   zapcore.StringDurationEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		ConsoleSeparator: " ",
	}

	// 无颜色的编码器配置（用于文件）
	plainEncoderConfig := zapcore.EncoderConfig{
		TimeKey:          "T",
		LevelKey:         "L",
		NameKey:          "N",
		CallerKey:        "C",
		MessageKey:       "M",
		StacktraceKey:    "S",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      zapcore.CapitalLevelEncoder,
		EncodeTime:       zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
		EncodeDuration:   zapcore.StringDurationEncoder,
		EncodeCaller:     zapcore.ShortCallerEncoder,
		ConsoleSeparator: " ",
	}

	//TODO
	isDev := mode == 1

	// 控制台写入器

	var consoleEncoder zapcore.Encoder
	var fileEncoder zapcore.Encoder

	if isDev {
		// 开发模式：控制台使用彩色编码器
		consoleEncoder = zapcore.NewConsoleEncoder(plainEncoderConfig)
		color.NoColor = false
		// 文件使用普通编码器（无颜色）
		fileEncoder = zapcore.NewConsoleEncoder(plainEncoderConfig)
	} else {
		// 生产模式：都使用 JSON 编码器（无颜色）
		consoleEncoder = &CustomEncoder{zapcore.NewJSONEncoder(plainEncoderConfig)}
		fileEncoder = zapcore.NewJSONEncoder(plainEncoderConfig)
	}

	//创建writer
	consoleWriter := color.Output
	//consoleAsyncWriter := NewAsyncWriter(consoleWriter)
	consoleWriterSyncer := zapcore.AddSync(consoleWriter)

	var fileWriteSyncer zapcore.WriteSyncer
	// 文件写入器（如果配置了路径）
	if logPath != "" {
		fileWriter := NewFileWriter(logPath)
		fileAsyncWriter := NewAsyncWriter(fileWriter)
		fileWriteSyncer = zapcore.AddSync(fileAsyncWriter)
	}

	// 创建多个 core
	var cores []zapcore.Core

	// 控制台 core
	cores = append(cores, zapcore.NewCore(consoleEncoder, consoleWriterSyncer, zLevel))

	// 文件 core（如果有文件路径）
	if logPath != "" && fileWriteSyncer != nil {
		cores = append(cores, zapcore.NewCore(fileEncoder, fileWriteSyncer, zLevel))
	}

	// 使用 NewTee 合并多个 core，实现同时输出
	core := zapcore.NewTee(cores...)

	return &Logger{zap.New(core)}, nil
}

type CustomEncoder struct {
	zapcore.Encoder
}

func (c *CustomEncoder) EncodeEntry(entry zapcore.Entry, fields []zap.Field) (*buffer.Buffer, error) {
	buf, err := c.Encoder.EncodeEntry(entry, fields)
	if err != nil {
		return buf, err
	}

	switch entry.Level {
	case zapcore.WarnLevel:
		e := buf.String()[:strings.LastIndex(buf.String(), ColorResetStr)+LenColorResetStr]
		t := buf.String()[strings.LastIndex(buf.String(), ColorResetStr)+LenColorResetStr:]
		buf.Reset()
		buf.WriteString(e + Yellow(t))
	case zapcore.ErrorLevel:
		e := buf.String()[:strings.LastIndex(buf.String(), ColorResetStr)+LenColorResetStr]
		t := buf.String()[strings.LastIndex(buf.String(), ColorResetStr)+LenColorResetStr:]
		buf.Reset()
		buf.WriteString(e + Red(t))
	}
	return buf, nil
}
func customLevelColorEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	var colorize func(a ...interface{}) string

	switch level {
	case zapcore.DebugLevel:
		colorize = Blue
	case zapcore.InfoLevel:
		colorize = Green
	case zapcore.WarnLevel:
		colorize = Yellow
	case zapcore.ErrorLevel:
		colorize = Red
	default:
		colorize = color.New(color.FgHiWhite).SprintFunc()
	}

	enc.AppendString(colorize(level.CapitalString()))
}

// 自定义时间编码器，添加颜色
func customTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(color.New(color.FgCyan).SprintFunc()(t.Format("2006-01-02 15:04:05.000")))
}

//func init() {
//	var err error
//	globalLogger, err = zap.NewDevelopment()
//	if err != nil {
//		// 处理错误，例如使用默认配置或panic
//		panic("failed to initialize logger: " + err.Error())
//	}
//}

func GetLogger() *Logger {
	if globalLogger == nil {
		//var err error
		//globalLogger, err = NewLogger(conf.GetConfig())
		//if err != nil {
		//	fmt.Println("致命错误:创建logger失败，触发panic", err)
		//	panic("致命错误:创建logger失败,触发panic" + err.Error())
		//}
		panic("致命错误:创建logger失败,触发panic")
	}
	return globalLogger
}

func SugaredLogger() *zap.SugaredLogger {
	return GetLogger().Sugar()
}

func SetGlobalLogger(l *Logger) {
	globalLogger = l
}

func NewAsyncWriter(w io.Writer) *law.WriteAsyncer {
	conf := law.NewConfig()
	conf.WithBufferSize(1024 * 1024 * 2)
	return law.NewWriteAsyncer(w, conf)
}

func GetKratosLogger() log.Logger {
	l := kratoszap.NewLogger(GetLogger().Logger)
	//kratoszap.WithMessageKey("default")(l)
	//ll := log.With(l, "time", log.Timestamp(time.DateTime), "caller", log.Caller(4))
	return l
}

func GetKratosLogHelper() *log.Helper {
	//log.With()
	h := log.NewHelper(GetKratosLogger(),
		log.WithSprint(Sprint),
	)
	return h
}
