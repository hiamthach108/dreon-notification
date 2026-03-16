package email

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestXmlEscape(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"empty", "", ""},
		{"no special chars", "Hello World", "Hello World"},
		{"ampersand", "a & b", "a &amp; b"},
		{"less than", "x < y", "x &lt; y"},
		{"greater than", "x > y", "x &gt; y"},
		{"double quote", `say "hi"`, `say &quot;hi&quot;`},
		{"single quote", "it's", "it&apos;s"},
		{"mixed", `<a href="?foo=1&bar=2">`, `&lt;a href=&quot;?foo=1&amp;bar=2&quot;&gt;`},
		{"url with angle brackets", "https://example.com/<path>", "https://example.com/&lt;path&gt;"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xmlEscape(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestXmlEscapeAny(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{"nil", nil, ""},
		{"string", "hello", "hello"},
		{"string with <", "a<b", "a&lt;b"},
		{"int", 42, "42"},
		{"int zero", 0, "0"},
		{"float", 3.14, "3.14"},
		{"bool", true, "true"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := xmlEscapeAny(tt.input)
			assert.Equal(t, tt.expected, got)
		})
	}
}

// minimalValidMJML is the smallest valid MJML that compiles and uses a template variable.
const minimalValidMJML = `<mjml>
<mj-body>
  <mj-section>
    <mj-column>
      <mj-text>{{xml .Message}}</mj-text>
      <mj-button href="{{xml .Link}}">Click</mj-button>
    </mj-column>
  </mj-section>
</mj-body>
</mjml>`

func TestRenderer_Render_success(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "test.mjml"), []byte(minimalValidMJML), 0644)
	require.NoError(t, err)

	r := NewRenderer(WithTemplateDir(dir))
	ctx := context.Background()
	params := map[string]any{
		"Message": "Hello",
		"Link":    "https://example.com",
	}

	html, err := r.Render(ctx, "test", params)
	require.NoError(t, err)
	require.NotEmpty(t, html)
	assert.Contains(t, html, "Hello")
	assert.Contains(t, html, "https://example.com")
}

func TestRenderer_Render_escapesParams(t *testing.T) {
	// Params with < and & must be escaped so MJML (XML) parses (no "unescaped < inside quoted string").
	// Without {{xml .}} these would cause parse errors; with it, Render succeeds and content appears in HTML.
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "safe.mjml"), []byte(minimalValidMJML), 0644)
	require.NoError(t, err)

	r := NewRenderer(WithTemplateDir(dir))
	ctx := context.Background()
	params := map[string]any{
		"Message": "Say <hello>",
		"Link":    "https://example.com?a=1&b=2",
	}

	html, err := r.Render(ctx, "safe", params)
	require.NoError(t, err)
	require.NotEmpty(t, html)
	// Escaping allowed MJML to parse; final HTML contains the visible content (entities may be decoded in output).
	assert.Contains(t, html, "Say ")
	assert.Contains(t, html, "hello")
	assert.Contains(t, html, "https://example.com")
	assert.Contains(t, html, "Click")
}

func TestRenderer_Render_missingTemplate(t *testing.T) {
	dir := t.TempDir()
	r := NewRenderer(WithTemplateDir(dir))
	ctx := context.Background()

	_, err := r.Render(ctx, "nonexistent", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "read mjml template")
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestRenderer_Render_invalidMJML(t *testing.T) {
	dir := t.TempDir()
	// Invalid MJML: unclosed tag
	err := os.WriteFile(filepath.Join(dir, "bad.mjml"), []byte("<mjml><mj-body><mj-section>"), 0644)
	require.NoError(t, err)

	r := NewRenderer(WithTemplateDir(dir))
	ctx := context.Background()

	_, err = r.Render(ctx, "bad", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mjml render")
}

func TestNewRenderer_defaultDir(t *testing.T) {
	r := NewRenderer()
	assert.Equal(t, "templates/emails", r.templateDir)
}

func TestNewRenderer_withTemplateDir(t *testing.T) {
	r := NewRenderer(WithTemplateDir("/custom/path"))
	assert.Equal(t, "/custom/path", r.templateDir)
}
