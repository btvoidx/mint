package mint

import (
	"errors"
	"sync"
)

// Zero value is ready to use.
type Emitter struct {
	mu   sync.RWMutex
	once sync.Once
	kc   uint64

	subs   map[uint64]any
	before []func(any) bool
	after  []func(any)
}

func (e *Emitter) init() {
	e.once.Do(func() {
		e.subs = make(map[uint64]any)
		// e.before = make([]func(*Emitter, any) bool, 0)
		// e.after = make([]func(*Emitter, any), 0)
	})
}

// Sends v to all registered channels/functions. They receive it asyncronously.
// Blocks until all listeners receive the value.
func Emit[T any](e *Emitter, v T) {
	if len(e.subs) == 0 {
		return
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	for _, h := range e.before {
		if h(&v) {
			return
		}
	}

	wg := new(sync.WaitGroup)
	for _, ch := range e.subs {
		if ch, ok := ch.(chan T); ok {
			wg.Add(1)
			go func() { ch <- v; wg.Done() }()
		}
	}
	wg.Wait()

	for _, h := range e.after {
		h(&v)
	}
}

// Registers a new receiver. Channel receives all emitted values which implement T.
// Such if T is any, ch will receive any emitted value.
//
// Calling off closes ch.
func On[T any](e *Emitter) (ch <-chan T, off func()) {
	e.mu.Lock()
	defer e.mu.Unlock()

	k := e.kc
	chn := make(chan T)

	e.init()
	e.kc += 1
	e.subs[k] = any(chn)

	return chn, func() {
		e.mu.Lock()
		defer e.mu.Unlock()
		delete(e.subs, k)
		close(chn)
	}
}

// Registers a new receiver and sends all values to fn.
// It is called with all emitted values which implement T.
// Such if T is any, fn will be called with any emitted value.
//
// Calling off unsubscribes fn from receiving new values.
func OnFn[T any](e *Emitter, fn func(v T)) (off func()) {
	ch, off := On[T](e)
	go func() {
		for v := range ch {
			fn(v)
		}
	}()
	return off
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
// Handler methods are called sequentially and receive *T, not T.
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
