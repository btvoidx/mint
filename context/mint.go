// Package mint provides a tiny generic event emitter.
//
//	e := new(mint.Emitter) // create an emitter
//	mint.On(e, func(context.Context, MyEvent)) // subscribe to MyEvent
//	mint.Emit(e, context.Background(), MyEvent{ ... }) // emit values to consumers
package mint

import (
	"context"
	"sync"
)

type key[T any] struct{}

// Emitter holds all active consumers and Emit hooks.
type Emitter struct {
	subc    uint64
	plugins []func(context.Context, any) func()
	// map[mkey[T]{}]map[uint64]func(context.Context, T)
	subs map[any]map[uint64]any

	mu sync.RWMutex
}

func (e *Emitter) init() {
	if e.subs == nil {
		e.subs = make(map[any]map[uint64]any)
	}
}

// Emit Sequentially pushes value v to all consumers of type T.
// Receive order is indetermenistic. Cancelling ctx waits
// for active consumer to return and stops emitting further.
//
// Using nil context will use context.Background() instead.
// error is always ctx.Err()
func Emit[T any](e *Emitter, ctx context.Context, v T) error {
	if e == nil {
		return ctx.Err()
	}

	if ctx == nil {
		ctx = context.Background()
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, fn := range e.plugins {
		after := fn(ctx, v)
		if after != nil {
			func() { defer after() }()
		}
	}

	subs, ok := e.subs[key[T]{}]
	if !ok {
		return ctx.Err()
	}

	for _, fn := range subs {
		if err := ctx.Err(); err != nil {
			return err
		}
		fn.(func(context.Context, T))(ctx, v)
	}

	return ctx.Err()
}

// On Registers a new consumer that receives all values which were
// emitted as T. So that On(e, func(context.Context, any)) will
// receive all values emitted with Emit[any](e, ...)
//
// Reliance on a certain value to be present in the context
// signals that it should be included in T.
//
// Call to off schedules consumer to stop once all concurrent Emits stop
// and returns a chan which will get closed once it is done.
// It is possible for consumer to receive values after a call to stop if
// other concurrent emits are ongoing.
func On[T any](e *Emitter, fn func(context.Context, T)) (off func() <-chan struct{}) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.init()

	if _, ok := e.subs[key[T]{}]; !ok {
		e.subs[key[T]{}] = make(map[uint64]any)
	}

	id := e.subc
	e.subc += 1
	e.subs[key[T]{}][id] = fn

	done := make(chan struct{})
	var once sync.Once
	return func() <-chan struct{} {
		go once.Do(func() {
			e.mu.Lock()
			defer e.mu.Unlock()

			delete(e.subs[key[T]{}], id)
			if len(e.subs[key[T]{}]) == 1 {
				delete(e.subs, key[T]{})
			}

			close(done)
		})
		return done
	}
}

// Use allows to hook into event emitting process. Plugins are
// called sequentially in order they were added to Emitter.
// Plugin is a function that takes Emitted values and
// returns nil or a function that will be called after
// all consumers got the Emitted value. Returned functions
// are called in reverse order via `defer` statement.
func Use(e *Emitter, plugin func(context.Context, any) func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.plugins = append(e.plugins, plugin)
}
