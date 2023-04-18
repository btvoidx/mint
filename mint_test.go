package mint_test

import (
	"testing"

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

	received := false
	off := mint.On(e, func(e event) { received = true })
	defer off()

	mint.Emit(e, event{"hello", "world"})

	if !received {
		t.Fatalf("didn't receive")
	}
}

func TestOff(t *testing.T) {
	e := new(mint.Emitter)

	c := 0
	off := mint.On(e, func(v int) { c = v })

	mint.Emit(e, 1)
	off()
	mint.Emit(e, 2)

	if c != 1 {
		t.Fatalf("expected c to be %d; got %d", 1, c)
	}
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

	defer mint.On(e, func(int) {})

	mint.Emit(e, event{"hello", "world"})

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

	var recieved bool
	off := mint.On(e, func(int) { recieved = true })
	defer off()

	mint.Emit(e, 0)
	if recieved {
		t.Fatalf("received despite block")
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

	var rec event
	off := mint.On(e, func(v event) { rec = v })
	defer off()

	mint.Emit(e, event{"hello", "world"})

	if rec.F1 != "bye" || rec.F2 != "space" {
		t.Fatalf("rewrite failed")
	}
}
