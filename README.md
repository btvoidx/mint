# Mint 🍃
> Tiny generic event emitter.

- **Small and simple**: mint has 3 exported functions and 1 type
- **Type safe**: built on generics
- **Fast**: no reflection - no overhead
- **Independant**: has no external dependencies
- **Extensible**: use `mint.Use` to hook into the process

### Install
```sh
go get github.com/btvoidx/mint
```

### Use
```go
// Create an emitter
e := new(mint.Emitter) // or &mint.Emmiter{}

// Create a consumer
func OnMyEvent(MyEvent) {
	// do stuff with the value
}

// Subscribe
off := mint.On(e, OnMyEvent)
defer off() // don't forget to unsubsribe later!

// ...

// In some other place
mint.Emit(e, MyEvent{Msg: "Hi", FromID: 1})
mint.Emit(e, MyEvent{Msg: "Hello indeed", FromID: 2})
```

If you want more control, you can jam into emitting process like so:
```go
type StringBlocker struct{ Enabled bool }

// BeforeEmit is called before sending out emitted values
func (sb StringBlocker) BeforeEmit(v any) (block bool) {
	_, ok := v.(*string) // v is *T, so you can even change it
	return sb.Enabled && ok
}

// AfterEmit is called after all consumers got the value
func (sb StringBlocker) AfterEmit(v any) {
	if v, ok := v.(*string); sb.Enabled && ok {
		panic("A pesky string got through the blocker!", v)
	}
}

// ...

// Add to emitter like so
mint.Use(e, StringBlocker{Enabled: true})

// Now receiving a value of string type is impossible,
// as it will get blocked by StringBlocker
mint.Emit(e, "a string value")
```

By default consumers receive data concurrently, but this behaivor
can be changed on per-emitter basis with `Emitter.SingleThread` field.
```go
// This particular emitter will emit data
// sequentially on the same thread as mint.Emit call.
e := &mint.Emitter{SingleThread: true}

// or

e := new(mint.Emitter)
// The setting can be flipped whenever,
// in-flight single threaded emits will block
// subsequent concurrent emits and vise versa
e.SingleThread = true 
```

If you prefer channel-based consumers you can create a wrapper
for `mint.Emit` so that it forwards all data to a channel. Be
mindful of this example as it will not work in single thread mode.
```go
func MyOn[T any](e *mint.Emitter) (<-ch T, off func()) {
	ch := make(chan T)
	off := mint.On(e, func(v T) {
		go func() { ch <- v }
	})

	return ch, func() {
		off()
		// drain channel so pending emits don't panic
		for {
			select {
			case <-ch:
			default:
				close(ch)
				return
			}
		}
	}
}

// Use it like so
ch, off := MyOn[MyEvent](e)
defer off()

for event := range ch {
	// deal with incoming data
}
```

For additional examples see [mint_test.go](mint_test.go).

### Reporting issues
If you have questions or problems just open [a new issue](../../issues/new).
