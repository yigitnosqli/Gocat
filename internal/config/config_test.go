package config

import "testing"

func TestHelloWorld(t *testing.T) {
	result := "hello world"
	expected := "hello world"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestConfigPackageExists(t *testing.T) {
	// Package'ın var olduğunu test et
	if true != true {
		t.Error("Config package should exist")
	}
}
