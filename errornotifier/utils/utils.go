// Package utils provides notifiers that implement the
// errornotifier.Notifier interface for use in testing.
package utils

import (
	"net/http"
	"testing"
)

// notice structurizes arguments of Notify function.
type notice struct {
	Error   interface{}
	Request *http.Request
}

// BufferNotifier implements errornotifier.Notifier interface.
// It stores all notified notices.
type BufferNotifier struct {
	Notices []notice
}

// Notify part of errornotifier.Notifier.
func (b *BufferNotifier) Notify(err interface{}, req *http.Request) error {
	b.Notices = append(b.Notices, notice{Error: err, Request: req})

	return nil
}

// TestNotifier is notifier that will call T.Fatal on any notification
type TestNotifier struct {
	T *testing.T
}

// Notify part of errornotifier.Notifier
func (t TestNotifier) Notify(err interface{}, req *http.Request) error {
	t.T.Fatal(err)
	return nil
}
