package common

import (
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Logger interface {
	Info(msg string, keysAndValues ...interface{})
	Debug(msg string, keysAndValues ...interface{})
	Error(err error, msg string, keysAndValues ...interface{})
}

type logger struct {
	log logr.Logger
}

func NewLogger(name string, names ...string) Logger {
	l := ctrl.Log.WithName(name)
	for _, n := range names {
		l = l.WithName(n)
	}
	return logger{log: l}
}

func NewLoggerWithLogr(l logr.Logger) Logger {
	return logger{log: l}
}

func (l logger) Info(msg string, keysAndValues ...interface{}) {
	l.log.V(0).Info(msg, keysAndValues...)
}

func (l logger) Debug(msg string, keysAndValues ...interface{}) {
	l.log.V(1).Info(msg, keysAndValues...)
}

func (l logger) Error(err error, msg string, keysAndValues ...interface{}) {
	l.log.Error(err, msg, keysAndValues...)
}

func (l logger) WithValues(keysAndValues ...interface{}) Logger {
	return logger{log: l.log.WithValues(keysAndValues...)}
}
