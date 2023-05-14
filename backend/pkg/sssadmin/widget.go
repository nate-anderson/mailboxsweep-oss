package sssadmin

import (
	"bytes"
	"context"
	"fmt"
	"html/template"
	"net/url"
)

type Widget interface {
	HandleAction(ctx context.Context, data url.Values) (any, error)
	HTML() string
	Title() string
	Description() *string
}

type StyledWidget interface {
	Widget
	CSS() string
}

type renderWidgetData struct {
	Endpoint string
	Request  url.Values
	Response any
}

func renderWidget(widget Widget, endpoint string, response any, request url.Values) (template.HTML, error) {
	html := widget.HTML()
	title := widget.Title()

	innerTemplate, err := template.New(title).Parse(html)
	if err != nil {
		return "", fmt.Errorf("failed parsing template inner HTML: %w", err)
	}

	widgetData := renderWidgetData{
		Endpoint: endpoint,
		Request:  request,
		Response: response,
	}
	innerBuffer := bytes.Buffer{}
	if err = innerTemplate.Execute(&innerBuffer, widgetData); err != nil {
		return "", fmt.Errorf("failed executing inner template: %w", err)
	}

	innerHTML := template.HTML(innerBuffer.String())

	// todo: only read this once
	outerTemplate, err := template.New("container").Parse(cardHTML)
	if err != nil {
		return "", fmt.Errorf("failed parsing outer HTML (template): %w", err)
	}

	cd := cardData{
		InnerHTML: innerHTML,
		Title:     title,
		Endpoint:  endpoint,
	}

	outerBuffer := bytes.Buffer{}
	if err = outerTemplate.Execute(&outerBuffer, cd); err != nil {
		return "", err
	}

	return template.HTML(outerBuffer.String()), nil
}

type cardData struct {
	Endpoint  string
	Title     string
	InnerHTML template.HTML
}

const cardHTML = `
	<div class="card mb-3">
		<div class="card-header">
			{{.Title}}
		</div>
		<div class="card-body">
			{{.InnerHTML}}
		</div>
	</div>
`
