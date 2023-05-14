# Simple Server-Side Admin Dashboards

Simply implement `Widget`

```go
type Widget interface {
	HandleAction(ctx context.Context, data url.Values) (any, error)
	HTML() string
	Title() string
	Description() *string
}
```

Create a dashboard and serve it:

```go
widget1 := sssadmin.LogSomethingWidget{Name: "Widg1"}
widget2 := sssadmin.LogSomethingWidget{Name: "Widg2"}

dash := sssadmin.New("Test dash", "A test dash", widget1, widget2)

addr := ":8002"
http.ListenAndServe(addr, dash)
```
