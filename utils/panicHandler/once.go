// Copyright 2019 the orbs-network-go authors
// This file is part of the orbs-network-go library in the Orbs project.
//
// This source code is licensed under the MIT license found in the LICENSE file in the root directory of this source tree.
// The above notice should be included in all copies or substantial portions of the software.

package panicHandler

type Errorer interface {
	Error(args ...interface{})
}

type ContextEndedChan <-chan struct{}

// Once Runs f() in a new goroutine; if it panics, logs the error and stack trace to the specified Errorer
func Once(errorHandler Errorer, f func()) {
	go func() {
		tryOnce(errorHandler, f)
	}()
}
