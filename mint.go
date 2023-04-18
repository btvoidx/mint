package mint

import (
	"errors"
	"sync"
)

// Emitter holds all active consumers and Emit hooks.
//
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
	})
}

// Pushes v to all consumers. They do not block each other, but block Emit.
//
// Sequentially calls BeforeEmit before pushing the value to consumers,
// and AfterEmit after all consumers received the value.
func Emit[T any](e *Emitter, v T) {
	e.mu.RLock()
	for _, h := range e.before {
		if h(&v) {
			e.mu.RUnlock()
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
	e.mu.RUnlock()

	wg.Wait()

	e.mu.RLock()
	for _, h := range e.after {
		h(&v)
	}
	e.mu.RUnlock()
}

// Registers a new consumer. ch receives all values which implement T.
// So if T is any, ch will receive any emitted value.
//
// Calling off closes ch. Calling off multiple times is a no-op.
func On[T any](e *Emitter) (ch <-chan T, off func()) {
	e.mu.Lock()
	defer e.mu.Unlock()

	k := e.kc
	chn := make(chan T)

	e.init()
	e.kc += 1
	e.subs[k] = any(chn)

	once := sync.Once{}
	return chn, func() {
		once.Do(func() {
			e.mu.Lock()
			defer e.mu.Unlock()
			delete(e.subs, k)

			// drain channel
			// so pending emits don't panic
			for {
				select {
				case <-chn:
				default:
					close(chn)
					return
				}
			}
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
