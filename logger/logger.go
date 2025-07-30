package logger

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
	"strings"

	"github.com/EPecherkin/catty-counting/config"
	"github.com/pkg/errors"
)

var level string

const (
	ERROR                = "error"
	CALLER               = "caller"
	MESSAGE_ID           = "telegram_message_id"
	TELEGRAM_USER_ID     = "telegram_user_id"
	TELEGRAM_UPDATE_ID   = "telegram_update_id"
	TELEGRAM_CHAT_ID     = "telegram_chat_id"
	TELEGRAM_MESSAGE_ID  = "telegram_message_id"
	TELEGRAM_DOCUMENT_ID = "telegram_document_id"
	TELEGRAM_PHOTO_ID    = "telegram_photo_id"
)

// Build a logger that prints error stacktrace
// Inspired by https://stackoverflow.com/questions/77304845/how-to-log-errors-with-log-slog
func NewLogger() *slog.Logger {
	logLevel := slog.LevelInfo
	if config.LogDebug() {
		logLevel = slog.LevelDebug
	}

	handler := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: replaceAttr,
		Level:       logLevel,
	})
	logger := slog.New(handler)
	logger.Info(fmt.Sprintf("Logging level: %v", logLevel))
	slog.SetDefault(logger)
	return logger
}

func IsDebug() bool {
	return level == "debug"
}

func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	switch a.Value.Kind() {
	// other cases

	case slog.KindAny:
		switch v := a.Value.Any().(type) {
		case error:
			a.Value = fmtErr(v)
		}
	}

	return a
}

// fmtErr returns a slog.GroupValue with keys "msg" and "trace". If the error
// does not implement interface { StackTrace() errors.StackTrace }, the "trace"
// key is omitted.
func fmtErr(err error) slog.Value {
	var groupValues []slog.Attr

	groupValues = append(groupValues, slog.String("msg", err.Error()))

	type StackTracer interface {
		StackTrace() errors.StackTrace
	}

	// Find the trace to the location of the first errors.New,
	// errors.Wrap, or errors.WithStack call.
	var st StackTracer
	for err := err; err != nil; err = errors.Unwrap(err) {
		if x, ok := err.(StackTracer); ok {
			st = x
		}
	}

	if st != nil {
		groupValues = append(groupValues,
			slog.Any("trace", traceLines(st.StackTrace())),
		)
	}

	return slog.GroupValue(groupValues...)
}

func traceLines(frames errors.StackTrace) []string {
	traceLines := make([]string, len(frames))

	// Iterate in reverse to skip uninteresting, consecutive runtime frames at
	// the bottom of the trace.
	var skipped int
	skipping := true
	for i := len(frames) - 1; i >= 0; i-- {
		// Adapted from errors.Frame.MarshalText(), but avoiding repeated
		// calls to FuncForPC and FileLine.
		pc := uintptr(frames[i]) - 1
		fn := runtime.FuncForPC(pc)
		if fn == nil {
			traceLines[i] = "unknown"
			skipping = false
			continue
		}

		name := fn.Name()

		if skipping && strings.HasPrefix(name, "runtime.") {
			skipped++
			continue
		} else {
			skipping = false
		}

		filename, lineNr := fn.FileLine(pc)

		traceLines[i] = fmt.Sprintf("%s %s:%d", name, filename, lineNr)
	}

	return traceLines[:len(traceLines)-skipped]
}
