package mint_test

import (
	"context"
	"testing"
	"time"

	"github.com/btvoidx/mint"
)

type event struct {
	F1, F2 string
}

type plugin struct {
	before func(any) (block bool)
	after  func(any)
}

func (p *plugin) BeforeEmit(v any) bool { return p.before(v) }
func (p *plugin) AfterEmit(v any)       { p.after(v) }

func TestOn(t *testing.T) {
	e := new(mint.Emitter)

	ch, off := mint.On[event](e)
	defer off()

	event := event{"hello", "world"}

	go mint.Emit(e, event)

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	select {
	case received := <-ch:
		if event != received {
			t.Fatalf("wrong values: %#v != %#v", event, received)
		}
	case <-ctx.Done():
		t.Fatalf("didn't receive")
	}
}

func TestOnFn(t *testing.T) {
	e := new(mint.Emitter)

	received := event{}
	off := mint.OnFn(e, func(e event) {
		received = e
	})
	defer off()

	event := event{"hello", "world"}

	mint.Emit(e, event)

	if event != received {
		t.Fatalf("wrong values: %#v != %#v", event, received)
	}
}

func TestOff(t *testing.T) {
	e := new(mint.Emitter)

	ch, off := mint.On[event](e)
	event := event{"hello", "world"}

	go mint.Emit(e, event)
	go mint.Emit(e, event)
	go mint.Emit(e, event)
	go mint.Emit(e, event)
	go mint.Emit(e, event)

	<-ch
	off()

	mint.Emit(e, event)
}

func TestUse(t *testing.T) {
	e := new(mint.Emitter)

	var bef, aft bool

	err := mint.Use(e, &plugin{
		before: func(any) (cancel bool) { bef = true; return false },
		after:  func(any) { aft = true },
	})
	if err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	ch, off := mint.On[event](e)
	defer off()

	go mint.Emit(e, event{"hello", "world"})
	<-ch

	if !bef {
		t.Fatalf("before fn didn't run")
	}

	if !aft {
		t.Fatalf("after fn didn't run")
	}
}

func TestUseWithBlock(t *testing.T) {
	e := new(mint.Emitter)

	err := mint.Use(e, &plugin{
		before: func(any) bool { return true },
		after:  func(any) {},
	})
	if err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	ch, off := mint.On[event](e)
	defer off()

	go mint.Emit(e, event{"hello", "world"})

	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	defer cancel()

	select {
	case <-ch:
		t.Fatalf("received despite block")
	case <-ctx.Done():
	}
}

func TestUseWithRewrite(t *testing.T) {
	e := new(mint.Emitter)

	err := mint.Use(e, &plugin{
		before: func(v any) (block bool) {
			if e, ok := v.(*event); ok {
				*e = event{"bye", "space"}
			}
			return false
		},
		after: func(any) {},
	})
	if err != nil {
		t.Fatalf("failed to register plugin: %v", err)
	}

	ch, off := mint.On[event](e)
	defer off()

	go mint.Emit(e, event{"hello", "world"})
	event := <-ch
	if event.F1 != "bye" || event.F2 != "space" {
		t.Fatalf("rewrite failed")
	}
}
