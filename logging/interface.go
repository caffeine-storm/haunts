package logging

import (
	"github.com/runningwild/glop/glog"
)

type stdLogInterceptor interface {
	Printf(format string, v ...interface{})
}

type Logger interface {
	glog.Logger
	stdLogInterceptor
}
