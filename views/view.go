package views

import (
	"embed"
	"html/template"
	"io"
	"log"
)

//go:embed **/*.html
var viewFS embed.FS

var views *template.Template

func init() {
	_views, err := template.ParseFS(viewFS, "./**/*.html")
	if err != nil {
		log.Fatalf("Failed to read templates with error: %v", err)
	}
	views = _views
}

func Render(w io.Writer, name string, v any) error {
	return views.ExecuteTemplate(w, name, v)
}
