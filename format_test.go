package slogformat

import (
	"bytes"
	"fmt"
	"github.com/utvecklat/slogformat/internal/buffer"
	"io"
	"log/slog"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"
)

const timeFormat = "15:04:05.000Z07:00"
const dateTimeFormat = "2006-01-02T15:04:05.000Z07:00"

func nextLineNumber() int {
	var pcs [1]uintptr
	// skip [runtime.Callers, this function, this function's caller]
	runtime.Callers(2, pcs[:])
	pc := pcs[0]

	fs := runtime.CallersFrames([]uintptr{pc})
	f, _ := fs.Next()
	return f.Line + 1
}

func TestWithDateTime(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := slog.New(New(buf, &HandlerOptions{
		Date:      true,
		AddSource: true,
		Level:     slog.LevelInfo,
	}))

	testOutput(t, dateTimeFormat, logger, buf)
}

func TestWithTime(t *testing.T) {
	buf := new(bytes.Buffer)
	logger := slog.New(New(buf, &HandlerOptions{
		AddSource: true,
		Level:     slog.LevelInfo,
	}))
	testOutput(t, timeFormat, logger, buf)
}

func testOutput(t *testing.T, timeFormat string, logger *slog.Logger, buf *bytes.Buffer) {
	lineNum := nextLineNumber()
	logger.Info("some info", "field", "value")
	result := buf.String()

	rx := regexp.MustCompile(`^([^ ]+) `)

	matches := rx.FindStringSubmatch(result)
	if len(matches) < 2 {
		t.Errorf("could not match time field in '%s' - %v", result, matches)
		t.FailNow()
	}
	logTime, err := time.Parse(timeFormat, matches[1])
	if err != nil {
		t.Errorf("could not parse time '%s': %v", matches[1], err)
	}

	result = strings.TrimRight(result, "\n")

	expected := fmt.Sprintf("%s INFO  format_test.go:%d some info - field=value", logTime.Format(timeFormat), lineNum)
	if result != expected {
		t.Errorf("expected: '%s', got: '%s' %x!=%x", expected, result, expected, result)
	}
}

func TestWriteNumber(t *testing.T) {
	buf := buffer.New()

	const bits = 32 << (^uint(0) >> 63)
	const maxInt = 1<<(bits-1) - 1

	expect := "9223372036854775807"
	if bits == 32 {
		expect = "2147483647"
	}
	writeNumber(buf, maxInt)

	if buf.String() != expect {
		t.Errorf("expected '%s' got '%s'", expect, buf.String())
	}
}

func TestWriteTime(t *testing.T) {
	testTime := time.Unix(0, 0)
	buf := buffer.New()
	writeTime(buf, testTime.UTC())

	expect := time.Unix(0, 0).UTC().Format(timeFormat)
	result := buf.String()
	if result != expect {
		t.Errorf("expected '%s', got: '%s'", expect, result)
	}

}

func TestWriteDateTime(t *testing.T) {
	testTime := time.Unix(0, 0)
	buf := buffer.New()
	writeDateTime(buf, testTime.UTC())

	expect := time.Unix(0, 0).UTC().Format(dateTimeFormat)
	result := buf.String()
	if result != expect {
		t.Errorf("expected '%s', got: '%s'", expect, result)
	}
}

func TestAllocWriteLineNumber(t *testing.T) {
	buf := buffer.New()
	const bits = 32 << (^uint(0) >> 63)
	const maxInt = 1<<(bits-1) - 1

	got := testing.AllocsPerRun(5, func() {
		writeNumber(buf, maxInt)
	})
	if got > 0 {
		t.Errorf("got %f allocs, expected 3", got)
	}
}

func TestAllocLog(t *testing.T) {
	logger := slog.New(New(io.Discard, &HandlerOptions{}))
	got := int(testing.AllocsPerRun(5, func() {
		logger.Info("test", "key", "val")
	}))
	if got != 0 {
		t.Errorf("got %d allocs, expected 0", got)
	}
}

func TestAllocLogWithDate(t *testing.T) {
	logger := slog.New(New(io.Discard, &HandlerOptions{
		AddSource:         false,
		AddSourceFullPath: false,
		Level:             slog.LevelDebug,
		BoldMessage:       true,
		ColorSeverity:     true,
		Date:              true,
	}))

	got := int(testing.AllocsPerRun(5, func() {
		logger.Info("test", "key", "val")
	}))
	if got != 0 {
		t.Errorf("got %d allocs, expected 0", got)
	}
}

func TestAllocWithGroup(t *testing.T) {
	logger := slog.New(New(io.Discard, &HandlerOptions{}))
	logger = logger.WithGroup("group1")
	logger = logger.With("fruit", "apple", "drink", "coffee")

	got := int(testing.AllocsPerRun(5, func() {
		logger.Info("test", "key", "val")
	}))
	if got != 0 {
		t.Errorf("got %d allocs, expected 0", got)
	}
}

func TestAllocWithSource(t *testing.T) {
	logger := slog.New(New(io.Discard, &HandlerOptions{
		AddSource: true,
		Date:      true,
	}))
	got := int(testing.AllocsPerRun(5, func() {
		logger.Info("test", "key", "val")
	}))
	if got != 2 {
		t.Errorf("got %d allocs, expected 2", got)
	}
}

// Keep track of standard slog allocations
func TestAllocWithSlogDefault(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		AddSource:   true,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}))
	got := int(testing.AllocsPerRun(5, func() {
		logger.Info("test", "key", "val")
	}))
	if got != 5 {
		t.Errorf("got %d allocs, expected 2", got)
	}
}

func BenchmarkText(b *testing.B) {
	logger := slog.New(New(io.Discard, &HandlerOptions{}))

	for i := 0; i < b.N; i++ {
		logger.Info("test message", "key", "some value")
	}
}

func BenchmarkFull(b *testing.B) {
	logger := slog.New(New(io.Discard, &HandlerOptions{
		AddSource:         false,
		AddSourceFullPath: false,
		Level:             slog.LevelDebug,
		BoldMessage:       true,
		ColorSeverity:     true,
		Date:              true,
	}))
	for i := 0; i < b.N; i++ {
		logger.Info("test message", "key", "some value")
	}
}

func BenchmarkDefault(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	for i := 0; i < b.N; i++ {
		logger.Info("test message", "key", "some value")
	}
}

func BenchmarkDefaultFull(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{
		AddSource:   true,
		Level:       slog.LevelDebug,
		ReplaceAttr: nil,
	}))

	for i := 0; i < b.N; i++ {
		logger.Info("test message", "key", "some value")
	}
}
