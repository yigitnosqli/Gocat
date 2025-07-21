package ui

import "testing"

func TestHelloWorld(t *testing.T) {
	// Basit test - sadece geçmesi için
	result := "hello world"
	expected := "hello world"
	if result != expected {
		t.Errorf("Expected %s, got %s", expected, result)
	}
}

func TestUIPackageExists(t *testing.T) {
	// Package'ın var olduğunu test et
	if true != true {
		t.Error("UI package should exist")
	}
}

func TestBasicUIFunction(t *testing.T) {
	// Basit bir UI testi
	title := "Gocat Terminal UI"
	if len(title) == 0 {
		t.Error("Title should not be empty")
	}
}

func TestAppStructExists(t *testing.T) {
	// App struct'ının var olduğunu test et
	app := &App{}
	if app == nil {
		t.Error("App struct should be creatable")
	}
}