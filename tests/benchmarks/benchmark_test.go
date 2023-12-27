package main

import (
	"fmt"
	"testing"

	m "github.com/nubificus/urunc/pkg/metrics"
)

func BenchmarkZerologWriter(b *testing.B) {
	var zerologWriter = m.NewZerologMetrics(("/tmp/urunc.zlog"))
	for i := 0; i < b.N; i++ {
		for j := 0; j < 20; j++ {
			zerologWriter.Capture("test-container", fmt.Sprintf("TS%02d", j))
		}
	}
}

func BenchmarkMockWriter(b *testing.B) {
	var mockWriter = m.NewMockMetrics(("/tmp/urunc.zlog"))
	for i := 0; i < b.N; i++ {
		for j := 0; j < 20; j++ {
			mockWriter.Capture("test-container", fmt.Sprintf("TS%02d", j))
		}
	}
}
