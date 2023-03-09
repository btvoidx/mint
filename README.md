# Mint 🍃
> Tiny generic event emitter / pubsub.

- **Small and simple**: mint has 4 exported functions and 1 type
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

// Subscribe to MyEvent
ch, off := mint.On[MyEvent](e)
defer off() // don't forget to unsubsribe after!

for event := range ch {
	// do something with event!
}

// ...

// In some other place
e.Emit(e, MyEvent{Msg: "Hi", FromID: 1})
e.Emit(e, MyEvent{Msg: "Hello indeed", FromID: 2})
```

If you want more control, you can jam into emitting process like so:
```go
type StringBlocker struct{ Enabled bool }

// BeforeEmit is called before sending out emitted values
func (sb StringBlocker) BeforeEmit(v any) (block bool) {
	_, ok := v.(*string) // v is *T, so you can even change it
	return sb.Enabled && ok
}

// AfterEmit is called after all listener got the value
func (sb StringBlocker) AfterEmit(v any) {
	if v, ok := v.(*string); sb.Enabled && ok {
		panic("A pesky string got through the blocker!", v)
	}
}

// ...

// Add to emitter like so
mint.Use(e, StringBlocker{Enabled: true})

// ...

// Now receiving a value of type "string" is impossible,
// as it will get blocked by StringBlocker
ch, _ := mint.On[string](e)
go mint.Emit(e, "a string value")
<-ch // will not receive the event
```

For additional examples see `mint_test.go`.

### Reporting issues
If you have questions or problems just open [a new issue](../../issues/new).
