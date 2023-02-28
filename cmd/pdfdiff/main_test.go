package main

import (
	"github.com/aelbrecht/pdfdump/external/pdf"
	"testing"
)

func BenchmarkPDFComparing(b *testing.B) {
	for i := 0; i < b.N; i++ {
		pdf.Compare("./test/input_a.pdf", "./test/input_b.pdf", false)
	}
}
