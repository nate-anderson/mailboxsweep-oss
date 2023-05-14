package sssadmin

import (
	"context"
	"fmt"
	"log"
	"net/url"
)

type LogSomethingWidget struct {
	Name string
}

func (w LogSomethingWidget) HTML() string {
	return `<div>
		<form method="POST" action="{{.Endpoint}}">
			<input type="text" name="thing_to_log">
			<input type="submit">
		</form>
		{{with .Response}}
		<p>Last result: {{.Msg}}</p>
		{{end}}
	</div>`
}

type ActionResult struct {
	Msg string
}

func (w LogSomethingWidget) HandleAction(ctx context.Context, data url.Values) (any, error) {
	if len(data) == 0 {
		return nil, nil
	}

	val := data.Get("thing_to_log")
	log.Println(val)

	result := ActionResult{Msg: fmt.Sprintf("Logged %s", val)}
	return result, nil
}

func (w LogSomethingWidget) Description() *string {
	return nil
}

func (w LogSomethingWidget) Title() string {
	return w.Name
}
