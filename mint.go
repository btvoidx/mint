// Package mint provides a tiny generic event emitter.
//
//	e := new(mint.Emitter) // create an emitter
//	mint.On(e, func(MyEvent)) // subscribe to MyEvent
//	mint.Emit(e, MyEvent{ ... }) // emit values to consumers
package mint

import (
	"reflect"
	"sync"
)

// Emitter holds all active consumers and Emit hooks.
type Emitter struct {
	subc    uint64
	plugins []func(any) func()
	// map[fntype[T]()]map[uint64]func(T)
	subs map[reflect.Type]map[uint64]any

	mu   sync.Mutex
	once sync.Once
}

func (e *Emitter) init() {
	e.once.Do(func() { e.subs = make(map[reflect.Type]map[uint64]any) })
}

func fntype[T any]() reflect.Type {
	var t func(T)
	return reflect.TypeOf(t)
}

// Sequentially pushes value v to all consumers of type T. Order in which consumers
// receive the value is not determenistic.
func Emit[T any](e *Emitter, v T) {
	for _, fn := range e.plugins {
		after := fn(v)
		if after != nil {
			defer after()
		}
	}

	subs, ok := e.subs[fntype[T]()]
	if !ok {
		return
	}

	// WARN a panic can happen here, and it did once
	// but I haven't been able to reproduce it
	// premise: call `off()` during concurrent Emit call
	for _, fn := range subs {
		fn.(func(T))(v)
	}
}

// Registers a new consumer that receives all values which were
// emitted as T. So that On(e, func(any)) will receive all values
// emitted with Emit[any](e, ...)
//
// Calling off stops consumer. Multiple calls are no-op.
func On[T any](e *Emitter, fn func(T)) (off func()) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.init()

	typ := fntype[T]()
	if _, ok := e.subs[typ]; !ok {
		e.subs[typ] = make(map[uint64]any)
	}

	id := e.subc
	e.subc += 1
	e.subs[typ][id] = fn

	var once sync.Once
	return func() {
		once.Do(func() {
			e.mu.Lock()
			defer e.mu.Unlock()
			delete(e.subs[typ], id)
		})
	}
}

// Use allows to hook into event emitting process. Plugns are
// called sequentially in order they were added to Emitter.
// Plugin is a function that takes Emitted values and
// returns nil or a function that will be called after
// all consumers got the Emitted value. Returned functions
// are called in reverse order.
func Use(e *Emitter, plugin func(any) func()) {
	e.plugins = append(e.plugins, plugin)
}
