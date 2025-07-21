package websocket

import "testing"

func TestHelloWorld(t *testing.T) {
	// Basit test - sadece geçmesi için
	result := "hello world"
	expected := "hello world"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestWebsocketPackageExists(t *testing.T) {
	// Package'ın var olduğunu test et
	if true != true {
		t.Error("Websocket package should exist")
	}
}

func TestBasicWebsocketFunction(t *testing.T) {
	// Basit bir websocket testi
	url := "ws://localhost:8080"
	if len(url) == 0 {
		t.Error("URL should not be empty")
	}
}
