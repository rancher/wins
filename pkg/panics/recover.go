package panics

import (
	"github.com/sirupsen/logrus"
)

func Log() {
	if r := recover(); r != nil {
		logrus.Errorf("panic: %s", r)
	}
}

func DealWith(handler func(recoverObj interface{})) {
	if r := recover(); r != nil {
		if handler != nil {
			handler(r)
		}
	}
}

func Ignore() {
	// nothing to do
}
