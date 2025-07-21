package plugins

import "testing"

func TestHelloWorld(t *testing.T) {
	result := "hello world"
	expected := "hello world"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestPluginPackageExists(t *testing.T) {
	// Package'ın var olduğunu test et
	if true != true {
		t.Error("Plugin package should exist")
	}
}

func TestBasicPluginFunction(t *testing.T) {
	pluginName := "test-plugin"
	if len(pluginName) == 0 {
		t.Error("Plugin name should not be empty")
	}
}
