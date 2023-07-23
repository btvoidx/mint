// Package mint provides a tiny generic event emitter.
// Version under mint/context exposes context api.
//
//	e := new(mint.Emitter) // create an emitter
//	mint.On(e, func(MyEvent)) // subscribe to MyEvent
//	mint.Emit(e, MyEvent{ ... }) // emit values to consumers
package mint

import (
	"context"

	cm "github.com/btvoidx/mint/context"
)

// Emitter holds all active consumers and Emit hooks.
type Emitter = cm.Emitter

// Emit Sequentially pushes value v to all consumers of type T.
// Receive order is indetermenistic.
func Emit[T any](e *Emitter, v T) {
	_ = cm.Emit(e, context.Background(), v)
}

// On Registers a new consumer that receives all values which were
// emitted as T. So that On(e, func(any)) will
// receive all values emitted with Emit[any](e, ...)
//
// Call to off schedules consumer to stop once all concurrent Emits stop
// and returns a <-chan which will get closed once it is done.
// It is possible for consumer to receive values after a call to stop if
// other concurrent emits are ongoing.
func On[T any](e *Emitter, fn func(T)) (off func() <-chan struct{}) {
	return cm.On(e, func(_ context.Context, v T) { fn(v) })
}

// Use allows to hook into event emitting process. Plugins are
// called sequentially in order they were added to Emitter.
// Plugin is a function that takes Emitted values and
// returns nil or a function that will be called after
// all consumers got the Emitted value. Returned functions
// are called in reverse order via `defer` statement.
func Use(e *Emitter, plugin func(any) func()) {
	cm.Use(e, func(_ context.Context, v any) func() { return plugin(v) })
}
