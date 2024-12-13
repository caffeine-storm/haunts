package actions_test

import (
	"bytes"
	"encoding/gob"
	"path/filepath"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/actions"
	. "github.com/smartystreets/goconvey/convey"
)

var datadir string

func init() {
	datadir, _ = filepath.Abs("../../data_test")
	base.SetDatadir(datadir)
}

func TestActionSpecs(t *testing.T) {
	Convey("Action Specs", t, ActionSpec)
}

func ActionSpec() {
	game.RegisterActions()
	Convey("Actions are loaded properly.", func() {
		basic := game.MakeAction("Basic Test")
		_, ok := basic.(*actions.BasicAttack)
		So(ok, ShouldEqual, true)
	})

	Convey("Actions can be gobbed without loss of type.", func() {
		buf := bytes.NewBuffer(nil)
		enc := gob.NewEncoder(buf)

		var as []game.Action
		as = append(as, game.MakeAction("Move Test"))
		as = append(as, game.MakeAction("Basic Test"))

		err := enc.Encode(as)
		So(err, ShouldEqual, nil)

		dec := gob.NewDecoder(buf)
		var as2 []game.Action
		err = dec.Decode(&as2)
		So(err, ShouldEqual, nil)

		_, ok := as2[0].(*actions.Move)
		So(ok, ShouldEqual, true)

		_, ok = as2[1].(*actions.BasicAttack)
		So(ok, ShouldEqual, true)
	})
}
