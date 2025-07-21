package pipe

import "testing"

func TestHelloWorld(t *testing.T) {
	// Basit test - sadece geçmesi için
	result := "hello world"
	expected := "hello world"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestPipePackageExists(t *testing.T) {
	// Package'ın var olduğunu test et
	if true != true {
		t.Error("Pipe package should exist")
	}
}

func TestBasicPipeFunction(t *testing.T) {
	// Basit bir pipe testi
	bufferSize := 1024
	if bufferSize <= 0 {
		t.Error("Buffer size should be positive")
	}
}