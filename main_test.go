package main

import "testing"

func TestShouldUseSingleInstanceLock(t *testing.T) {
	tests := []struct {
		name                   string
		goos                   string
		multiInstanceOnWindows bool
		want                   bool
	}{
		{name: "windows default", goos: "windows", want: true},
		{name: "windows multi-instance", goos: "windows", multiInstanceOnWindows: true, want: false},
		{name: "darwin multi-instance flag", goos: "darwin", multiInstanceOnWindows: true, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldUseSingleInstanceLock(tt.goos, tt.multiInstanceOnWindows); got != tt.want {
				t.Fatalf("shouldUseSingleInstanceLock(%q, %t) = %t, want %t", tt.goos, tt.multiInstanceOnWindows, got, tt.want)
			}
		})
	}
}
