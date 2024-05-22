package main

import (
	"fmt"
	"testing"

	"github.com/nubificus/urunc/internal/constants"
	m "github.com/nubificus/urunc/internal/metrics"
)

func BenchmarkZerologWriter(b *testing.B) {
	var zerologWriter = m.NewZerologMetrics(constants.TimestampTargetFile)
	for i := 0; i < b.N; i++ {
		for j := 0; j < 20; j++ {
			zerologWriter.Capture(fmt.Sprintf("container%02d", i), fmt.Sprintf("TS%02d", j))
		}
	}
}

func BenchmarkMockWriter(b *testing.B) {
	var mockWriter = m.NewMockMetrics(constants.TimestampTargetFile)
	for i := 0; i < b.N; i++ {
		for j := 0; j < 20; j++ {
			mockWriter.Capture(fmt.Sprintf("container%02d", i), fmt.Sprintf("TS%02d", j))
		}
	}
}
