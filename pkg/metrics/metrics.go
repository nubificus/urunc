package metrics

import (
	"os"

	"github.com/rs/zerolog"
)

var enableTimestamps = os.Getenv("URUNC_TIMESTAMPS")

type Writer interface {
	Log(msg string)
}

type zerologMetrics struct {
	logger *zerolog.Logger
}

func (z *zerologMetrics) Log(msg string) {
	z.logger.Log().Msg(msg)
}

func NewZerologMetrics(target string) Writer {
	if enableTimestamps == "1" {
		file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil
		}

		logger := zerolog.New(file).Level(zerolog.InfoLevel)
		return &zerologMetrics{
			logger: &logger,
		}
	}
	return &mockWriter{}
}

type mockWriter struct{}

func (m *mockWriter) Log(_ string) {}

func NewMockMetrics(_ string) Writer {
	return &mockWriter{}
}
