// Package render provides a custom Echo renderer backed by embedded templates.
package render

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"strings"

	"github.com/labstack/echo/v4"
)

// Page wraps a view model with the common metadata the base layout needs.
type Page struct {
	Title       string
	Network     string
	AccentClass string
	Version     string
	Data        any
}

// Renderer implements echo.Renderer using templates loaded from an fs.FS.
// It auto-discovers every *.html file in templates/ (except base.html) and
// pairs each one with base.html into a named template set.
type Renderer struct {
	templates map[string]*template.Template
}

var _ echo.Renderer = (*Renderer)(nil)

// New parses all page templates from fsys and returns a ready Renderer.
// shortenHashes controls whether the shortenHash template func truncates hashes;
// when false it becomes an identity function.
func New(fsys fs.FS, shortenHashes bool) (*Renderer, error) {
	entries, err := fs.ReadDir(fsys, "templates")
	if err != nil {
		return nil, fmt.Errorf("read templates dir: %w", err)
	}
	r := &Renderer{templates: make(map[string]*template.Template)}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".html") || e.Name() == "base.html" {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".html")
		t, err := template.New("base").Funcs(funcMap(shortenHashes)).ParseFS(fsys,
			"templates/base.html",
			"templates/"+e.Name(),
		)
		if err != nil {
			return nil, fmt.Errorf("parse template %q: %w", name, err)
		}
		r.templates[name] = t
	}
	return r, nil
}

// Render executes the named template with data, writing output to w.
func (r *Renderer) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	t, ok := r.templates[name]
	if !ok {
		return fmt.Errorf("template %q not registered", name)
	}
	return t.ExecuteTemplate(w, "base", data)
}

func funcMap(shortenHashes bool) template.FuncMap {
	shorten := shortenHash
	if !shortenHashes {
		shorten = func(h string) string { return h }
	}
	return template.FuncMap{
		"btc":         formatBTC,
		"shortenHash": shorten,
		"add":         func(a, b int) int { return a + b },
		"pct":         func(v float64) string { return fmt.Sprintf("%.0f%%", v*100) },
		// not is needed because html/template has no built-in negation of values.
		"not": func(v any) bool {
			if v == nil {
				return true
			}
			switch val := v.(type) {
			case bool:
				return !val
			case string:
				return val == ""
			case int:
				return val == 0
			case int64:
				return val == 0
			default:
				return false
			}
		},
	}
}

func formatBTC(sats int64) string {
	neg := sats < 0
	if neg {
		sats = -sats
	}
	s := fmt.Sprintf("%d.%08d", sats/1e8, sats%1e8)
	if neg {
		return "-" + s
	}
	return s
}

func shortenHash(h string) string {
	if len(h) <= 16 {
		return h
	}
	return h[:8] + "…" + h[len(h)-8:]
}
