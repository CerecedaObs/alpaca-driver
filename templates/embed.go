package templates

import (
	"embed"
	"html/template"
)

//go:embed *.html
var FS embed.FS

// LoadTemplates loads all templates from the embedded filesystem
func LoadTemplates() (*template.Template, error) {
	return template.ParseFS(FS, "*.html")
}
