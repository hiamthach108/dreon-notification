package sms

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/hiamthach108/dreon-notification/config"
)

// IBodyRenderer renders a text template with params to produce the SMS body.
// Template name maps to a file (e.g. "verify-otp" -> verify-otp.txt).
type IBodyRenderer interface {
	RenderBody(ctx context.Context, templateName string, params map[string]any) (body string, err error)
}

// BodyRenderer reads .txt templates and executes them with params.
type BodyRenderer struct {
	templateDir string
}

// NewBodyRenderer returns a renderer that reads templates from cfg.SMS.SMSTemplateDir (default: templates/sms).
func NewBodyRenderer(cfg *config.AppConfig) IBodyRenderer {
	dir := cfg.SMS.SMSTemplateDir
	if dir == "" {
		dir = "templates/sms"
	}
	return &BodyRenderer{templateDir: dir}
}

// RenderBody implements IBodyRenderer.
func (r *BodyRenderer) RenderBody(ctx context.Context, templateName string, params map[string]any) (string, error) {
	path := filepath.Join(r.templateDir, templateName+".txt")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read sms template %s: %w", templateName, err)
	}
	tpl, err := template.New(templateName).Parse(string(data))
	if err != nil {
		return "", fmt.Errorf("parse template %s: %w", templateName, err)
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, params); err != nil {
		return "", fmt.Errorf("execute template %s: %w", templateName, err)
	}
	return buf.String(), nil
}
