package main

import "testing"

func TestRunDemo(t *testing.T) {
	if err := runDemo("127.0.0.1:0"); err != nil {
		t.Fatalf("runDemo failed: %v", err)
	}
}
