package notify

import (
	"log"
	"runtime/debug"
)

// this allows Plandex Cloud to inject error monitoring
// all non-streaming handlers are already wrapped with different logic, so this is only needed for errors in streaming handlers

type Severity int

const (
	SeverityInfo Severity = iota
	SeverityError
)

var NotifyErrFn func(severity Severity, data ...interface{})

func RegisterNotifyErrFn(fn func(severity Severity, data ...interface{})) {
	NotifyErrFn = fn
}

func NotifyErr(severity Severity, data ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("panic in NotifyErr: %v\n%s", r, debug.Stack())
		}
	}()

	if NotifyErrFn != nil {
		NotifyErrFn(severity, data...)
	}
}
