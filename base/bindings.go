package base

import (
	"fmt"
	"strings"

	"github.com/runningwild/glop/gin"
)

type KeyBinds map[string]string
type KeyMap map[string]gin.Key

var (
	default_map KeyMap
)

func SetDefaultKeyMap(km KeyMap) {
	default_map = km
}
func GetDefaultKeyMap() KeyMap {
	return default_map
}

// TODO(tmckee): return an error instead of panicing if we don't recognize the
// key by name.
func getKeysFromString(str string) []gin.KeyId {
	parts := strings.Split(str, "+")
	var kids []gin.KeyId
	for _, part := range parts {
		part = strings.ToLower(osSpecifyKey(part))
		var kid gin.KeyId
		switch {
		case len(part) == 1: // Single character - should be ascii
			kid = gin.KeyId{
				Device: gin.DeviceId{
					Type:  gin.DeviceTypeKeyboard,
					Index: gin.DeviceIndexAny,
				},
				Index: gin.KeyIndex(part[0]),
			}

		case part == "ctrl":
			// TODO(tmckee): gin distinguishes between left/right but we want an 'any
			// control'.
			kid = gin.AnyLeftControl

		case part == "shift":
			// TODO(tmckee): gin distinguishes between left/right but we want an 'any
			// shift'.
			kid = gin.AnyLeftShift

		case part == "alt":
			// TODO(tmckee): gin distinguishes between left/right but we want an 'any
			// alt'.
			kid = gin.AnyLeftAlt

		case part == "gui":
			// TODO(tmckee): gin distinguishes between left/right but we want an 'any
			// gui'.
			kid = gin.AnyLeftGui

		case part == "space":
			kid = gin.AnySpace

		case part == "rmouse":
			kid = gin.AnyMouseRButton

		case part == "lmouse":
			kid = gin.AnyMouseLButton

		case part == "vwheel":
			kid = gin.AnyMouseWheelVertical

		case part == "up":
			kid = gin.AnyUp

		case part == "down":
			kid = gin.AnyDown

		default:
			panic(fmt.Sprintf("Unknown key '%s'", part))

		}
		kids = append(kids, kid)
	}
	return kids
}

func (kb KeyBinds) MakeKeyMap() KeyMap {
	var key_map KeyMap = map[string]gin.Key{}
	for keyName, val := range kb {
		parts := strings.Split(val, ",")
		var binds []gin.Key
		for i, part := range parts {
			kids := getKeysFromString(part)

			if len(kids) == 1 {
				binds = append(binds, gin.In().GetKeyById(kids[0]))
			} else {
				// The last kid is the main kid and the rest are modifiers
				main := kids[len(kids)-1]
				kids = kids[0 : len(kids)-1]
				var down []bool
				for range kids {
					down = append(down, true)
				}
				binds = append(binds, gin.In().BindDerivedKey(fmt.Sprintf("%s:%d", keyName, i), gin.In().MakeBinding(main, kids, down)))
			}
		}
		if len(binds) == 1 {
			key_map[keyName] = binds[0]
		} else {
			var actual_binds []gin.Binding
			for _, bind := range binds {
				// TODO(#17): uhh... doesn't passing nil, nil at the end mean we don't
				// support modifiers for compound derived-keys? Shouldn't we just pass
				// in bind.Modifiers and bind.Down???
				actual_binds = append(actual_binds, gin.In().MakeBinding(bind.Id(), nil, nil))
			}
			key_map[keyName] = gin.In().BindDerivedKey(keyName, actual_binds...)
		}
	}
	return key_map
}
