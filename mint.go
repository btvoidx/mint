// Package mint provides a tiny generic event emitter.
//
//	e := new(mint.Emitter) // create an emitter
//	mint.On(e, func(MyEvent)) // subscribe to MyEvent
//	mint.Emit(e, MyEvent{ ... }) // emit values to consumers
package mint

import (
	"sync"
)

// Plugin is a function that takes Emitted values of type T
// and returns nil or a function that will be called after
// all consumers got the Emitted value.
type Plugin[T any] func(T) func()

// Emitter holds all active consumers and Emit hooks.
type Emitter struct {
	subc    uint64
	subs    map[uint64]any // func(T)
	plugins []any          // Plugin

	mu   sync.Mutex
	once sync.Once
}

func (e *Emitter) init() {
	e.once.Do(func() { e.subs = make(map[uint64]any) })
}

// Sequentially pushes value v to all consumers. Order in which consumers
// receive the value is not determenistic.
func Emit[T any](e *Emitter, v T) {
	for _, fn := range e.plugins {
		if fn, ok := fn.(Plugin[T]); ok {
			after := fn(v)
			if after != nil {
				defer after()
			}
		}
	}

	// WARN a panic can happen here, and it did once
	// but I haven't been able to reproduce it
	// premise: call `off()` during concurrent Emit call
	for _, fn := range e.subs {
		if fn, ok := fn.(func(T)); ok {
			fn(v)
		}
	}
}

// Registers a new consumer. If T is an interface, fn is called with all values which implement T.
// So if T is any, fn will receive any emitted value.
//
// Calling off stops consumer. Multiple calls are no-op.
func On[T any](e *Emitter, fn func(T)) (off func()) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.init()
	id := e.subc
	e.subc += 1
	e.subs[id] = fn

	var once sync.Once
	return func() {
		once.Do(func() {
			e.mu.Lock()
			defer e.mu.Unlock()
			delete(e.subs, id)
		})
	}
}

// Use allows to hook into event emitting process. Plugns are
// called sequentially in order they were added to Emitter.
// Plugin is a function that takes Emitted values of type T
// and returns nil or a function that will be called after
// all consumers got the Emitted value. Returned functions
// are called in reverse order.
func Use[T any](e *Emitter, p Plugin[T]) {
	e.plugins = append(e.plugins, p)
}
