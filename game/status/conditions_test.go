package status_test

import (
	"bytes"
	"encoding/gob"
	"path/filepath"
	"testing"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game/status"
	"github.com/MobRulesGames/haunts/logging"
	. "github.com/smartystreets/goconvey/convey"
)

var datadir string

func init() {
	datadir, _ = filepath.Abs("../../data_test")
	base.SetDatadir(datadir)
	logging.SetupLogger(datadir)
}

func TestConditions(t *testing.T) {
	status.RegisterAllConditions()
	Convey("Conditions Specs", t, ConditionsSpec)
}

func ConditionsSpec() {
	Convey("Conditions are loaded properly.", func() {
		basic := status.MakeCondition("Basic Test")
		_, ok := basic.(*status.BasicCondition)
		So(ok, ShouldEqual, true)
		So(basic.Strength(), ShouldEqual, 5)
		So(basic.Kind(), ShouldEqual, status.Fire)
		var b status.Base
		b = basic.ModifyBase(b, status.Unspecified)
		So(b.Attack, ShouldEqual, 3)
	})

	Convey("Conditions can be gobbed without loss of type.", func() {
		buf := bytes.NewBuffer(nil)
		enc := gob.NewEncoder(buf)

		var cs []status.Condition
		cs = append(cs, status.MakeCondition("Basic Test"))

		err := enc.Encode(cs)
		So(err, ShouldEqual, nil)

		dec := gob.NewDecoder(buf)
		var cs2 []status.Condition
		err = dec.Decode(&cs2)
		So(err, ShouldEqual, nil)

		_, ok := cs2[0].(*status.BasicCondition)
		So(ok, ShouldEqual, true)
	})

	Convey("Conditions stack properly", func() {
		var s status.Inst
		fd := status.MakeCondition("Fire Debuff Attack")
		pd := status.MakeCondition("Poison Debuff Attack")
		pd2 := status.MakeCondition("Poison Debuff Attack 2")
		So(s.AttackBonusWith(status.Unspecified), ShouldEqual, 0)
		s.ApplyCondition(pd)
		So(s.AttackBonusWith(status.Unspecified), ShouldEqual, -1)
		s.ApplyCondition(fd)
		So(s.AttackBonusWith(status.Unspecified), ShouldEqual, -2)
		s.ApplyCondition(fd)
		So(s.AttackBonusWith(status.Unspecified), ShouldEqual, -2)
		s.ApplyCondition(pd)
		So(s.AttackBonusWith(status.Unspecified), ShouldEqual, -2)
		s.ApplyCondition(pd2)
		So(s.AttackBonusWith(status.Unspecified), ShouldEqual, -3)
	})

	Convey("Resistances work", func() {
		var s status.Inst
		fr1 := status.MakeCondition("Fire Resistance")
		fr2 := status.MakeCondition("Greater Fire Resistance")
		So(s.CorpusVs("Fire"), ShouldEqual, s.CorpusVs("Unspecified"))
		s.ApplyCondition(fr1)
		So(s.CorpusVs("Fire"), ShouldEqual, s.CorpusVs("Unspecified")+1)
		So(s.CorpusVs("Panic"), ShouldEqual, s.CorpusVs("Unspecified"))
		So(s.CorpusVs("Brutal"), ShouldEqual, s.CorpusVs("Unspecified"))
		s.ApplyCondition(fr2)
		So(s.CorpusVs("Fire"), ShouldEqual, s.CorpusVs("Unspecified")+3)
		So(s.CorpusVs("Panic"), ShouldEqual, s.CorpusVs("Unspecified"))
		So(s.CorpusVs("Brutal"), ShouldEqual, s.CorpusVs("Unspecified"))
	})

	Convey("Basic conditions last the appropriate amount of time", func() {
		var s status.Inst
		s.UnmarshalJSON([]byte(`
      {
        "Base": {
          "Hp_max": 100,
          "Ap_max": 10
        },
        "Dynamic": {
          "Hp": 100
        }
      }`))
		pd := status.MakeCondition("Poison Debuff Attack")
		pd2 := status.MakeCondition("Poison Debuff Attack 2")
		pd.Strength()
		pd2.Strength()
		So(s.HpCur(), ShouldEqual, 100)
		s.ApplyCondition(status.MakeCondition("Fire Debuff Attack"))
		So(s.HpCur(), ShouldEqual, 100)
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 99)
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 98)
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 97)
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 97)

		s.ApplyCondition(status.MakeCondition("Fire Debuff Attack"))
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 96)
		s.ApplyCondition(status.MakeCondition("Fire Debuff Attack"))
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 95)
		s.ApplyCondition(status.MakeCondition("Fire Debuff Attack"))
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 94)
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 93)
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 92)
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 92)

		s.ApplyCondition(status.MakeCondition("Fire Debuff Attack"))
		s.ApplyCondition(status.MakeCondition("Poison Debuff Attack"))
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 90)
		s.ApplyCondition(status.MakeCondition("Fire Debuff Attack"))
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 88)
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 86)
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 85)
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 85)

		s.ApplyCondition(status.MakeCondition("Fire Debuff Attack"))
		s.ApplyCondition(status.MakeCondition("Poison Debuff Attack"))
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 83)
		s.ApplyCondition(status.MakeCondition("Poison Debuff Attack 2"))
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 80)
		s.ApplyCondition(status.MakeCondition("Poison Debuff Attack"))
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 77)
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 75)
		s.OnRound()
		So(s.HpCur(), ShouldEqual, 75)
	})
}
