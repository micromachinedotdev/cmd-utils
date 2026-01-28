package utils

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

type ColorHandler struct {
	mu    *sync.Mutex
	attrs []slog.Attr
}

func NewColorHandler() *ColorHandler {
	return &ColorHandler{mu: &sync.Mutex{}}
}

func (h *ColorHandler) Enabled(_ context.Context, _ slog.Level) bool {
	return true
}

func (h *ColorHandler) Handle(_ context.Context, r slog.Record) error {
	var color lipgloss.Style
	switch r.Level {
	case slog.LevelDebug:
		color = Muted
	case slog.LevelInfo:
		color = Default
	case slog.LevelWarn:
		color = Warning
	case slog.LevelError:
		color = Fail
	default:
		color = Default
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	msg := Gray.Render(r.Time.Format(time.TimeOnly)) + " " + color.Render(r.Message)

	if r.Level == slog.LevelWarn {
		msg = Gray.Render(r.Time.Format(time.TimeOnly)) + " " + WarningWithBackground.Render("WARNING") + " " + color.Render(r.Message)
	}

	if r.Level == slog.LevelError {
		msg = Gray.Render(r.Time.Format(time.TimeOnly)) + " " + ErrorWithBackground.Render("✗ ERROR") + " " + color.Render(r.Message)
	}

	for _, a := range h.attrs {
		msg += " " + Muted.Render(a.Key) + "=" + fmt.Sprintf("%v", a.Value.Any())
	}

	r.Attrs(func(a slog.Attr) bool {
		msg += " " + Muted.Render(a.Key) + "=" + fmt.Sprintf("%v", a.Value.Any())
		return true
	})

	fmt.Fprintln(os.Stderr, msg)
	return nil
}

func (h *ColorHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &ColorHandler{
		mu:    h.mu,
		attrs: append(slices.Clone(h.attrs), attrs...),
	}
}

func (h *ColorHandler) WithGroup(_ string) slog.Handler {
	return h // groups not implemented
}
