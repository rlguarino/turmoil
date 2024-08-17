package main

import (
	"testing"
	"time"
)

func TestTestName(t *testing.T) {
	input := time.Date(2024, 8, 15, 16, 17, 18, int(900*time.Millisecond), time.UTC)
	result := testTempDir(input)
	expected := "store/20240815161718.900"
	if result != expected {
		t.Errorf("unexpected result from testName(%v), got:%v expected: %v", input, result, expected)
	}
}
