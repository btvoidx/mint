# Mint 🍃
> Tiny generic event emitter.

- **Very simple**: mint has 3 exported functions and 1 type
- **Type safe**: built on generics
- **Fast**: does not use reflection
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

By importing the package as `"github.com/btvoidx/mint/context"` 
you can access contextful api.
```go
import "github.com/btvoidx/mint/context"

ctx, cancel := context.WithTimeout(context.Background(), time.Second)
defer cancel()

// Not all consumers may receive the message due to timeout,
// but Emit does wait for the active consumer to finish.
err := mint.Emit(e, ctx, MyEvent{Msg: "A message"})

// err is always ctx.Err()
if err != ctx.Err() {	/* unreachable code */ }
```

Both versions can operate
on the same `mint.Emitter` instance, as contextless api
just wraps the contextful one with `context.Background()`.
```go
// MyEvent consumers will receive both events.
mintctx.Emit(e, ctx, MyEvent{Msg: "A message"})
mint.Emit(e, MyEvent{Msg: "A message"}) // uses context.Background()
```

If you prefer channel-based consumers you can create a wrapper
for `mint.On` so that it forwards all data to a channel.
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
