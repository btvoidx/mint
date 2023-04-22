// Package mint provides a tiny generic event emitter.
//
//	e := new(mint.Emitter) // create an emitter
//	mint.On(e, func(MyEvent)) // subscribe to MyEvent
//	mint.Emit(e, MyEvent{ ... }) // emit values to consumers
package mint

import (
	"errors"
	"sync"
	"sync/atomic"
)

// Emitter holds all active consumers and Emit hooks.
//
// Zero value is ready to use.
type Emitter struct {
	// If true, all emits will run sequentially on the same thread
	// Emit was called on. While single-thread emit is running,
	// multi-threaded emits will block.
	//
	// Order in which consumers receive emits is still not determenistic,
	// as `Emitter` uses a map to keep track of active consumers.
	SingleThread bool

	mu   sync.RWMutex
	once sync.Once
	idc  atomic.Uint64

	subs   map[uint64]any // func(T)
	before []func(any) bool
	after  []func(any)
}

func (e *Emitter) init() {
	e.once.Do(func() {
		e.subs = make(map[uint64]any)
	})
}

// Pushes v to all consumers.
//
// Sequentially calls BeforeEmit before pushing the value to consumers,
// and AfterEmit after all consumers received the value.
func Emit[T any](e *Emitter, v T) {
	e.mu.RLock()
	fns := make([]func(T), 0, len(e.subs))
	for _, fn := range e.subs {
		if fn, ok := fn.(func(T)); ok {
			fns = append(fns, fn)
		}
	}
	e.mu.RUnlock()

	if e.SingleThread {
		e.mu.Lock()
		defer e.mu.Unlock()

		for _, h := range e.before {
			if h(&v) {
				return
			}
		}

		for _, fn := range fns {
			fn(v)
		}

		for _, h := range e.after {
			h(&v)
		}
	} else {
		e.mu.RLock()
		defer e.mu.RUnlock()

		for _, h := range e.before {
			if h(&v) {
				return
			}
		}

		wg := new(sync.WaitGroup)

		for _, fn := range fns {
			wg.Add(1)
			fn := fn
			go func() { fn(v); wg.Done() }()
		}

		e.mu.RUnlock()
		wg.Wait()
		e.mu.RLock()

		for _, h := range e.after {
			h(&v)
		}
	}
}

// Registers a new consumer. fn is called with all values which implement T.
// So if T is any, fn will receive any emitted value.
//
// Calling off stops consumer. Multiple calls are no-op.
func On[T any](e *Emitter, fn func(T)) (off func()) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.init()
	key := e.idc.Add(1)
	e.subs[key] = fn

	var once sync.Once
	return func() {
		once.Do(func() {
			// try deleting on the same thread
			if e.mu.TryLock() {
				defer e.mu.Unlock()
				delete(e.subs, key)
				return
			}

			// otherwise put into goroutine to avoid dead-lock
			// which happens if off() is called by consumer
			go func() {
				e.mu.Lock()
				defer e.mu.Unlock()
				delete(e.subs, key)
			}()
		})
	}
}

// Use allows to hook into event emitting process.
//
// h must implement at least one of:
//
//	interface{ BeforeEmit(v any) }
//	interface{ BeforeEmit(v any) (block bool) }
//	interface{ AfterEmit(v any) }
//
// If it does not or is nil, an error is returned.
// Handler methods are called sequentially in order they were registered.
//
// BeforeEmit receives pointer to value, not value itself, so you can modify it before
// it gets pushed to consumers.
func Use(e *Emitter, h interface{}) error {
	if h == nil {
		return errors.New("h is nil")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.init()

	err := errors.New("h is not a valid interface; see doc comment")

	if h, ok := h.(interface{ BeforeEmit(v any) }); ok {
		wrap := func(v any) bool { h.BeforeEmit(v); return false }
		e.before = append(e.before, wrap)
		err = nil
	}
	if h, ok := h.(interface{ BeforeEmit(v any) bool }); ok {
		e.before = append(e.before, h.BeforeEmit)
		err = nil
	}
	if h, ok := h.(interface{ AfterEmit(v any) }); ok {
		e.after = append(e.after, h.AfterEmit)
		err = nil
	}

	return err
}
