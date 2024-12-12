package logging

import (
	"fmt"

	"github.com/runningwild/glop/glog"
)

type stdLogInterceptor interface {
	Printf(format string, v ...interface{})
}

type Logger interface {
	glog.Logger
	stdLogInterceptor
}

func Error(args ...interface{}) {
	fmt.Printf("%v\n", args)
}
