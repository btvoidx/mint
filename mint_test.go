package mint_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/btvoidx/mint"
	ctxmint "github.com/btvoidx/mint/context"
)

type event struct {
	F1, F2 string
}

func TestEmitSimple(t *testing.T) {
	e := new(mint.Emitter)

	received := false
	off := mint.On(e, func(e event) { received = true })
	defer off()

	mint.Emit(e, event{"hello", "world"})

	if !received {
		t.Fatalf("didn't receive")
	}
}

func TestEmitRecursive(t *testing.T) {
	e := new(mint.Emitter)

	var i int
	mint.On(e, func(event) {
		if i < 5 {
			i += 1
			mint.Emit(e, event{})
		}
	})

	mint.Emit(e, event{})

	if i != 5 {
		t.Fatalf("didn't receive")
	}
}

func TestEmitConcurrent(t *testing.T) {
	e := new(mint.Emitter)

	var i atomic.Uint32
	mint.On(e, func(event) {
		i.Add(1)
	})

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			mint.Emit(e, event{})
			wg.Done()
		}()
	}

	wg.Wait()
	if i := i.Load(); i != 100 {
		t.Fatalf("lost emits; got %d expected 100", i)
	}
}

func TestBroadReceiverMisfire(t *testing.T) {
	e := new(mint.Emitter)

	mint.On(e, func(any) { t.Error("misfired 'any' consumer with 'event' emit") })
	mint.Emit(e, event{})
}

func TestOffSimple(t *testing.T) {
	e := new(mint.Emitter)

	c := 0
	off := mint.On(e, func(v int) { c = v })

	mint.Emit(e, 1)
	<-off() // wait for it to synchronize
	mint.Emit(e, 2)

	if c != 1 {
		t.Fatalf("expected c to be %d; got %d", 1, c)
	}
}

func TestContextCancel(t *testing.T) {
	e := new(mint.Emitter)

	ctxmint.On(e, func(_ context.Context, v event) {
		t.Errorf("consumer called despite context cancel")
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_ = ctxmint.Emit(e, ctx, event{})
}

func TestUse(t *testing.T) {
	e := new(mint.Emitter)

	var before, after bool
	mint.Use(e, func(any) func() {
		before = true
		return func() { after = true }
	})

	mint.Emit(e, event{})

	if !before || !after {
		t.Fatalf("plugin was not called; before: %t, after: %t", before, after)
	}
}
