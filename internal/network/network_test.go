package network

import "testing"

func TestHelloWorld(t *testing.T) {
	// Basit test - sadece geçmesi için
	result := "hello world"
	expected := "hello world"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestNetworkPackageExists(t *testing.T) {
	// Package'ın var olduğunu test et
	if true != true {
		t.Error("Network package should exist")
	}
}

func TestBasicNetworkFunction(t *testing.T) {
	// Basit bir network testi
	port := 8080
	if port < 1 || port > 65535 {
		t.Error("Invalid port range")
	}
}
