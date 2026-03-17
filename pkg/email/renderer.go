package email

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/hiamthach108/dreon-notification/config"
	"github.com/preslavrachev/gomjml/mjml"
)

// xmlEscape escapes s for safe use inside XML/HTML (attributes and text).
// Prevents "unescaped < inside quoted string" and similar MJML parse errors.
func xmlEscape(s string) string {
	return strings.NewReplacer(
		"&", "&amp;",
		"<", "&lt;",
		">", "&gt;",
		`"`, "&quot;",
		"'", "&apos;",
	).Replace(s)
}

// xmlEscapeAny converts v to string (e.g. int for ExpiryMinutes) then escapes for XML.
func xmlEscapeAny(v any) string {
	if v == nil {
		return ""
	}
	return xmlEscape(fmt.Sprint(v))
}

// IRenderer renders an MJML email template with params to HTML.
// Template name maps to a file (e.g. "welcome" -> welcome.mjml).
type IRenderer interface {
	Render(ctx context.Context, templateName string, params map[string]any) (html string, err error)
}

// Renderer loads MJML templates, executes Go template with params, and compiles to HTML.
type Renderer struct {
	templateDir string
}

// NewRenderer returns a renderer that reads MJML from cfg.Email.TemplateDir (default: templates/emails).
func NewRenderer(cfg *config.AppConfig) IRenderer {
	dir := cfg.Email.TemplateDir
	if dir == "" {
		dir = "templates/emails"
	}
	return &Renderer{templateDir: dir}
}

// Render implements IRenderer.
func (r *Renderer) Render(ctx context.Context, templateName string, params map[string]any) (string, error) {
	mjmlPath := filepath.Join(r.templateDir, templateName+".mjml")
	data, err := os.ReadFile(mjmlPath)
	if err != nil {
		return "", fmt.Errorf("read mjml template %s: %w", templateName, err)
	}

	funcMap := template.FuncMap{
		"xml": xmlEscapeAny, // use {{xml .Field}} in MJML for any value (string, int, etc.) so <, &, ", ' are escaped
	}
	tpl, err := template.New(templateName).Funcs(funcMap).Parse(string(data))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", templateName, err)
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("execute template %s: %w", templateName, err)
	}

	html, err := mjml.Render(buf.String())
	if err != nil {
		return "", fmt.Errorf("mjml render %s: %w", templateName, err)
	}

	return html, nil
}
