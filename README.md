# Mint 🍃
> Tiny generic event emitter.

- **Very simple**: mint has 3 exported functions and 1 type
- **Type safe**: built on generics
- **Fast**: no reflection - no overhead
- **Independant**: has no external dependencies

### Get
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
