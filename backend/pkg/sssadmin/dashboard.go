package sssadmin

import (
	"html/template"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type Dashboard struct {
	title       string
	description string
	widgets     map[string]Widget
}

const dashboardHTML = `<!DOCTYPE html>
	<html>
		<head>
			<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/bootstrap@5.2.3/dist/css/bootstrap.min.css" integrity="sha384-rbsA2VBKQhggwzxH7pPCaAqO46MgnOM80zW1RWuH61DGLwZJEdK2Kadq2F9CUG65" crossorigin="anonymous">
		</head>
		<body>
			<div class="container">
				<div class="row mt-4">
					<div class="col">
						<h1>{{.Title}}</h1>
						<p>{{.Description}}</p>
						{{range .WidgetsHTML}}
							{{.}}
						{{end}}
					</div>
				</div>
			</div>
		</body>
	</html>
`

type dashboardData struct {
	Title       string
	Description string
	WidgetsHTML []template.HTML
}

func New(title, description string, widgets ...Widget) *Dashboard {
	widgetMap := make(map[string]Widget, len(widgets))
	for _, w := range widgets {
		path := endpoint(w)
		widgetMap[path] = w
	}
	return &Dashboard{widgets: widgetMap, title: title, description: description}
}

func endpoint(widget Widget) string {
	replace := regexp.MustCompile(`(\.|\?|\*|\s)+`)
	return strings.ToLower(replace.ReplaceAllString(widget.Title(), "_"))
}

func (d Dashboard) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	data := dashboardData{
		Title:       d.title,
		Description: d.description,
	}
	for endpoint, widget := range d.widgets {
		// pass data to this widget if applicable
		var requestForm url.Values
		var result any
		if req.URL.Path == "/"+endpoint {
			if err := req.ParseForm(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			requestForm = req.Form
			handlerResult, err := widget.HandleAction(req.Context(), requestForm)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			result = handlerResult
		}

		rendered, err := renderWidget(widget, "/"+endpoint, result, requestForm)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		data.WidgetsHTML = append(data.WidgetsHTML, rendered)
	}

	tmpl, err := template.New("sssadmin_dash").Parse(dashboardHTML)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	err = tmpl.Execute(w, data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
