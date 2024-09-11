// Stubbed version of the sound manager - lets us test things without having
// to link in fmod.

//go:build nosound
// +build nosound

package sound

import (
	"github.com/runningwild/glop/sprite"
)

func Init()                                 {}
func MapSounds(m map[string]string)         {}
func trigger(s *sprite.Sprite, name string) {}
func PlaySound(string, float64)             {}
func SetBackgroundMusic(file string)        {}
func PlayMusic(string)                      {}
func StopMusic()                            {}
func SetMusicParam(string, float64)         {}
