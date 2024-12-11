package logging

import (
	"github.com/runningwild/glop/glog"
)

type Logger interface {
	glog.Slogger
}
