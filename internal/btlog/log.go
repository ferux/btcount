package btlog

import (
	"fmt"
	"os"

	"github.com/ferux/btcount/internal/btcount"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type LogFormat string

const (
	LogFormatText LogFormat = "text"
	LogFormatJSON LogFormat = "json"
)

func NewLog(level string, format LogFormat, development bool) (log *zap.Logger, err error) {
	var enconfig zapcore.EncoderConfig
	if development {
		enconfig = zap.NewDevelopmentEncoderConfig()
	} else {
		enconfig = zap.NewProductionEncoderConfig()
	}

	var enc zapcore.Encoder
	switch format {
	case LogFormatJSON:
		enc = zapcore.NewJSONEncoder(enconfig)
	case LogFormatText:
		enc = zapcore.NewConsoleEncoder(enconfig)
	default:
		return nil, fmt.Errorf(
			"%w: %q (allowed are: %q, %q)",
			btcount.ErrInvalidParameter,
			format,
			LogFormatJSON,
			LogFormatText,
		)
	}

	var zaplevel zapcore.Level
	err = zaplevel.UnmarshalText([]byte(level))
	if err != nil {
		return nil, fmt.Errorf("parsing level: %w", err)
	}

	core := zapcore.NewCore(enc, zapcore.Lock(os.Stdout), zaplevel)

	return zap.New(core), nil
}
