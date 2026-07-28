package main

import (
	"reflect"
	"testing"
)

func TestBuildRestartArgsPreservesDeploymentOptions(t *testing.T) {
	input := []string{"--container-mode", "--address", "127.0.0.1:3211", "--delay=25", "--log-level", "2"}
	want := []string{"--container-mode", "--address", "127.0.0.1:3211", "--log-level", "2", "--delay=1"}
	if got := buildRestartArgs(input, "1"); !reflect.DeepEqual(got, want) {
		t.Fatalf("buildRestartArgs() = %#v, want %#v", got, want)
	}
}

func TestBuildRestartArgsRemovesSplitDelay(t *testing.T) {
	input := []string{"--delay", "15", "--hide-ui-when-boot"}
	want := []string{"--hide-ui-when-boot", "--delay=3"}
	if got := buildRestartArgs(input, "3"); !reflect.DeepEqual(got, want) {
		t.Fatalf("buildRestartArgs() = %#v, want %#v", got, want)
	}
}
