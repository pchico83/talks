package logger

import (
	"log"
	"os"

	raven "github.com/getsentry/raven-go"
	"github.com/pkg/errors"
)

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// Info logs straight to std
func Info(format string, v ...interface{}) {
	log.Printf(format, v...)
}

// Error logs to std and to sentry
func Error(err error) {
	if err != nil {
		raven.CaptureError(err, nil)
		logWithStacktrace(err)
	}
}

// Fatal logs to sentry, std and then exits with an error code
func Fatal(err error) {
	raven.CaptureErrorAndWait(err, nil)
	logWithStacktrace(err)
	os.Exit(1)
}

func logWithStacktrace(err error) {
	stack, ok := err.(stackTracer)
	if ok {
		log.Printf("error: %s %+v", err.Error(), stack.StackTrace()[1:])
	} else {
		log.Printf("error: %s", err)
	}
}
