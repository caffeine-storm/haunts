// This is an easy way to turn on/off depending on whether or not it is a
// devel or release build.

//go:build release
// +build release

package base

func IsDevel() bool { return false }
