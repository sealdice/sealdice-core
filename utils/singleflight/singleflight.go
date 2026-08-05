// Package singleflight provides typed duplicate call suppression with cancelable waiting.
package singleflight

import (
	"context"

	xsingleflight "golang.org/x/sync/singleflight"
)

// Group suppresses duplicate calls with the same key.
type Group[T any] struct {
	group xsingleflight.Group
}

// Do waits for a shared call result until it completes or ctx is canceled.
// Canceling a waiter does not cancel the shared call.
func (group *Group[T]) Do(ctx context.Context, key string, fn func() (T, error)) (T, error) {
	result := group.group.DoChan(key, func() (any, error) { return fn() })

	select {
	case <-ctx.Done():
		var zero T
		return zero, ctx.Err()
	case call := <-result:
		if call.Err != nil {
			var zero T
			return zero, call.Err
		}
		return call.Val.(T), nil
	}
}
