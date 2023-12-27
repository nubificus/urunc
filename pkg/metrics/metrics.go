package metrics

import (
	"os"

	"github.com/rs/zerolog"
)

var enableTimestamps = os.Getenv("URUNC_TIMESTAMPS")

type Writer interface {
	Capture(containerID string, timestampID string)
}

type zerologMetrics struct {
	logger *zerolog.Logger
}

func (z *zerologMetrics) Capture(containerID string, timestampID string) {
	z.logger.Log().Str("containerID", containerID).Str("timestampID", timestampID).Msg("")
}

func NewZerologMetrics(target string) Writer {
	if enableTimestamps == "1" {
		file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
		if err != nil {
			return nil
		}
		logger := zerolog.New(file).Level(zerolog.InfoLevel).With().Timestamp().Logger()
		zerolog.TimeFieldFormat = zerolog.TimeFormatUnixNano
		return &zerologMetrics{
			logger: &logger,
		}
	}
	return &mockWriter{}
}

type mockWriter struct{}

func (m *mockWriter) Capture(_, _ string) {}

func NewMockMetrics(_ string) Writer {
	return &mockWriter{}
}
