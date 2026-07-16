package panics

import (
	"github.com/sirupsen/logrus"
)

func Log() {
	if r := recover(); r != nil {
		logrus.Errorf("panic: %s", r)
	}
}
