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
			zerologWriter.Log(fmt.Sprintf("run%d", j))
		}
	}
}

func BenchmarkMockWriter(b *testing.B) {
	var mockWriter = m.NewMockMetrics(("/tmp/urunc.zlog"))
	for i := 0; i < b.N; i++ {
		for j := 0; j < 20; j++ {
			mockWriter.Log(fmt.Sprintf("run%d", j))
		}
	}
}
