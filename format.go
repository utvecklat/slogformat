package slogformat

import (
	"context"
	"github.com/utvecklat/slogformat/internal/buffer"
	"io"
	"log/slog"
	"path"
	"runtime"
	"sync"
	"time"
)

var mu *sync.Mutex

func init() {
	mu = &sync.Mutex{}
}

type BuffWriter struct {
	buf *buffer.Buffer
}

func (b *BuffWriter) Write(p []byte) (n int, err error) {
	return b.buf.Write(p)
}

type HandlerOptions struct {
	// AddSource causes the handler to compute the source code position
	// of the log statement and add a SourceKey attribute to the output.
	// This option is copied from slog.HandlerOptions
	// Enabling this option causes the logger to do two memory allocations for each log call.
	AddSource bool

	// AddSourceFullPath Write full file path of source file instead of just the file name.
	// Enabling this option automatically sets AddSource to true.
	AddSourceFullPath bool

	// Level reports the minimum record level that will be logged.
	// The handler discards records with lower levels.
	// If Level is nil, the handler assumes LevelInfo.
	// The handler calls Level.Level for each record processed;
	// to adjust the minimum level dynamically, use a LevelVar.
	// This option is copied from slog.HandlerOptions
	Level slog.Leveler

	// BoldMessage Write the message part of the log row in bold.
	BoldMessage bool

	// ColorSeverity Write severity with ansi colors.
	ColorSeverity bool

	// Date Prepend the date to the log row time for a complete RFC3339 date-time with milliseconds.
	Date bool
}

type Handler struct {
	bw         *BuffWriter
	w          io.Writer
	subHandler slog.Handler
	subAttrs   bool
	opts       *HandlerOptions
}

func replacer(groups []string, a slog.Attr) slog.Attr {
	if len(groups) == 0 {
		switch a.Key {
		case slog.TimeKey:
			return slog.Attr{}
		case slog.LevelKey:
			return slog.Attr{}
		case slog.MessageKey:
			return slog.Attr{}
		case slog.SourceKey:
			return slog.Attr{}
		}
	}
	return a
}

// New return slog.Handler to be used in slog.New, if options is nil, default options will be used
func New(w io.Writer, options *HandlerOptions) slog.Handler {
	bw := &BuffWriter{}
	var subHandler slog.Handler

	if options == nil {
		options = &HandlerOptions{}
	}
	if options.Level == nil {
		options.Level = slog.LevelInfo
	}
	if options.AddSourceFullPath {
		options.AddSource = true
	}

	subOpt := &slog.HandlerOptions{
		AddSource:   false,
		Level:       options.Level,
		ReplaceAttr: replacer,
	}

	subHandler = slog.NewTextHandler(bw, subOpt)
	return &Handler{
		w:          w,
		bw:         bw,
		subHandler: subHandler,
		opts:       options,
	}
}

func (h *Handler) clone() *Handler {
	return &Handler{
		bw:         h.bw,
		w:          h.w,
		subHandler: h.subHandler,
		opts:       h.opts,
	}
}

func (h *Handler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.subHandler.Enabled(ctx, level)
}

func writeNumber(buf *buffer.Buffer, num int) {
	if num < 0 {
		return
	}
	digits := "0123456789"
	nums := [19]byte{}

	i := 0
	for num >= 10 {
		q := num / 10
		nums[i] = digits[uint(num-q*10)]
		i++
		num = q
	}
	nums[i] = digits[uint(num)]
	for ; i >= 0; i-- {
		_ = buf.WriteByte(nums[i])
	}
}

func (h *Handler) appendSource(buf *buffer.Buffer, r slog.Record) {
	if r.PC != 0 {
		fs := runtime.CallersFrames([]uintptr{r.PC})
		f, _ := fs.Next()
		if h.opts.AddSourceFullPath {
			_, _ = buf.WriteString(f.File)
		} else {
			_, _ = buf.WriteString(path.Base(f.File))
		}
		_ = buf.WriteByte(':')
		writeNumber(buf, f.Line)
		_ = buf.WriteByte(' ')
	}
}

func appendSeverity(buf *buffer.Buffer, r slog.Record) {
	switch r.Level {
	case slog.LevelDebug:
		_, _ = buf.WriteString(r.Level.String())
	case slog.LevelInfo:
		_, _ = buf.WriteString(r.Level.String())
		_ = buf.WriteByte(' ')
	case slog.LevelWarn:
		_, _ = buf.WriteString(r.Level.String())
		_ = buf.WriteByte(' ')
	case slog.LevelError:
		_, _ = buf.WriteString(r.Level.String())
	default:
		_, _ = buf.WriteString(r.Level.String())
	}
	_ = buf.WriteByte(' ')
}

func appendSeverityColor(buf *buffer.Buffer, r slog.Record) {
	switch r.Level {
	case slog.LevelDebug:
		// Green
		_, _ = buf.WriteString("\x1b[1;92m")
		_, _ = buf.WriteString(r.Level.String())
	case slog.LevelInfo:
		// White
		_, _ = buf.WriteString("\x1b[1;97m")
		_, _ = buf.WriteString(r.Level.String())
		_ = buf.WriteByte(' ')
	case slog.LevelWarn:
		// Yellow
		_, _ = buf.WriteString("\x1b[1;93m")
		_, _ = buf.WriteString(r.Level.String())
		_ = buf.WriteByte(' ')
	case slog.LevelError:
		// Red
		_, _ = buf.WriteString("\x1b[1;91m")
		_, _ = buf.WriteString(r.Level.String())
	default:
		_, _ = buf.WriteString(r.Level.String())
	}
	_, _ = buf.WriteString("\x1b[0m")
	_ = buf.WriteByte(' ')
}

func appendBoldMessage(buf *buffer.Buffer, r slog.Record) {
	_, _ = buf.WriteString("\x1b[1m")
	_, _ = buf.WriteString(r.Message)
	_, _ = buf.WriteString("\x1b[0m")
}

func appendMessage(buf *buffer.Buffer, r slog.Record) {
	_, _ = buf.WriteString(r.Message)
}

func writeDateTime(buf *buffer.Buffer, t time.Time) {
	// 2006-01-02T15:04:05Z07:00
	writeNumber(buf, t.Year())
	_, _ = buf.WriteString("-")
	writeZeroPad(buf, int(t.Month()))
	_, _ = buf.WriteString("-")
	writeZeroPad(buf, t.Day())
	_, _ = buf.WriteString("T")
	writeTime(buf, t)
}

func writeTime(buf *buffer.Buffer, t time.Time) {
	writeZeroPad(buf, t.Hour())
	_, _ = buf.WriteString(":")
	writeZeroPad(buf, t.Minute())
	_, _ = buf.WriteString(":")
	writeZeroPad(buf, t.Second())
	_, _ = buf.WriteString(".")

	writeMSZeroPad(buf, t.Nanosecond()/1000000)

	z, offset := t.Zone()
	if z == "UTC" {
		_, _ = buf.WriteString("Z")
		return
	}
	if offset < 0 {
		_, _ = buf.WriteString("-")
		offset = -offset
	} else {
		_, _ = buf.WriteString("+")
	}

	hours := offset / 3600
	minutes := offset % 3600 / 60
	writeZeroPad(buf, hours)
	_, _ = buf.WriteString(":")
	writeZeroPad(buf, minutes)
}

func writeMSZeroPad(buf *buffer.Buffer, num int) {
	if num < 10 {
		_, _ = buf.WriteString("0")
	}
	if num < 100 {
		_, _ = buf.WriteString("0")
	}
	writeNumber(buf, num)
}

func writeZeroPad(buf *buffer.Buffer, num int) {
	if num < 10 {
		_, _ = buf.WriteString("0")
		writeNumber(buf, num)
	} else {
		writeNumber(buf, num)
	}
}

func (h *Handler) Handle(ctx context.Context, r slog.Record) error {
	buf := buffer.New()
	defer buf.Free()

	h.bw.buf = buf

	if !r.Time.IsZero() {
		if h.opts.Date {
			writeDateTime(buf, r.Time)
		} else {
			writeTime(buf, r.Time)
		}
		_ = buf.WriteByte(' ')
	}
	if h.opts.ColorSeverity {
		appendSeverityColor(buf, r)
	} else {
		appendSeverity(buf, r)
	}
	if h.opts.AddSource {
		h.appendSource(buf, r)
	}
	if h.opts.BoldMessage {
		appendBoldMessage(buf, r)
	} else {
		appendMessage(buf, r)
	}

	if r.NumAttrs() > 0 || h.subAttrs {
		_, _ = buf.WriteString(" - ")
	}

	mu.Lock()
	err := h.subHandler.Handle(ctx, r)
	if err != nil {
		return err
	}
	defer mu.Unlock()

	_, err = h.w.Write(*buf)

	return err
}

func (h *Handler) WithAttrs(as []slog.Attr) slog.Handler {
	return &Handler{
		bw:         h.bw,
		w:          h.w,
		subHandler: h.subHandler.WithAttrs(as),
		subAttrs:   true,
		opts:       h.opts,
	}
}

func (h *Handler) WithGroup(name string) slog.Handler {
	return &Handler{
		bw:         h.bw,
		w:          h.w,
		subHandler: h.subHandler.WithGroup(name),
		opts:       h.opts,
	}
}
