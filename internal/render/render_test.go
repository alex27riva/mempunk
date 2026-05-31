package render_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/alex27riva/mempunk/internal/render"
	"github.com/alex27riva/mempunk/web"
)

func TestNew(t *testing.T) {
	_, err := render.New(web.FS, true)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
}

func TestRenderOverview(t *testing.T) {
	r, err := render.New(web.FS, true)
	if err != nil {
		t.Fatal(err)
	}
	page := render.Page{
		Title:       "Overview",
		Network:     "regtest",
		AccentClass: "net-regtest",
		Data: map[string]any{
			"Chain":        "regtest",
			"Blocks":       int64(100),
			"Headers":      int64(100),
			"BestHash":     "aaaa0000000000000000000000000000000000000000000000000000000000aa",
			"Difficulty":   float64(4.656e-10),
			"IBD":          false,
			"RecentBlocks": []map[string]any{},
		},
	}
	var buf bytes.Buffer
	if err := r.Render(&buf, "overview", page, nil); err != nil {
		t.Fatalf("Render: %v", err)
	}
	out := buf.String()
	checks := []struct {
		desc string
		want string
	}{
		{"doctype", "<!DOCTYPE html>"},
		{"title", "Overview"},
		{"accent class", "net-regtest"},
		{"network badge", ">regtest<"},
		{"stylesheet link", "/static/style.css"},
		{"search form", `action="/search"`},
		{"chain stat", "regtest"},
	}
	for _, c := range checks {
		if !strings.Contains(out, c.want) {
			t.Errorf("%s: output missing %q", c.desc, c.want)
		}
	}
}

func TestRenderUnknownTemplate(t *testing.T) {
	r, err := render.New(web.FS, true)
	if err != nil {
		t.Fatal(err)
	}
	err = r.Render(nil, "nonexistent", nil, nil)
	if err == nil {
		t.Error("expected error for unknown template, got nil")
	}
}
