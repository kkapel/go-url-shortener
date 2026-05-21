package service

import "testing"

func BenchmarkGenerateRandomString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenerateRandomString(1000)
	}

}
