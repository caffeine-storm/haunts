package game

import (
	"path/filepath"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/util/algorithm"
)

// Choose side
// Choose map
// Intruders:
//   Choose mode
//   Choose units
//   Choose gear
// Denizens:
//   Choose whatever
// Place stuff down, blizitch

func makeChooserFromOptionBasicsFile(path string) (*Chooser, <-chan []string, error) {
	var bops []OptionBasic
	err := base.LoadAndProcessObject(path, "json", &bops)
	if err != nil {
		return nil, nil, err
	}
	var opts []Option
	algorithm.Map(bops, &opts, func(ob OptionBasic) Option { return &ob })
	return MakeChooser(opts)
}

func makeChooseGoalMenu() (*Chooser, <-chan []string, error) {
	path := filepath.Join(base.GetDataDir(), "ui", "start", "versus", "goals.json")
	return makeChooserFromOptionBasicsFile(path)
}

func makeChooseSideMenu() (*Chooser, <-chan []string, error) {
	path := filepath.Join(base.GetDataDir(), "ui", "start", "versus", "side.json")
	return makeChooserFromOptionBasicsFile(path)
}

func makeChooseVersusMetaMenu() (*Chooser, <-chan []string, error) {
	path := filepath.Join(base.GetDataDir(), "ui", "start", "versus", "meta.json")
	return makeChooserFromOptionBasicsFile(path)
}

type chooserMaker func() (*Chooser, <-chan []string, error)
type replacer func(gui.WidgetParent) error
type inserter func(gui.WidgetParent, replacer) error

func doChooserMenu(ui gui.WidgetParent, cm chooserMaker, r replacer, i inserter) error {
	chooser, done, err := cm()
	if err != nil {
		return err
	}
	ui.AddChild(chooser)
	go func() {
		m := <-done
		ui.RemoveChild(chooser)
		if m != nil {
			logging.Info("doChooserMenu", "chose", m)
			err = i(ui, r)
			if err != nil {
				logging.Error("doChooserMenu", "i(nsert) failed", err)
			}
		} else {
			err := r(ui)
			if err != nil {
				logging.Error("doChooserMenu", "r(eplacing failed", err)
			}
		}
	}()
	return nil
}

func insertGoalMenu(ui gui.WidgetParent, replace replacer) error {
	chooser, done, err := makeChooseGoalMenu()
	if err != nil {
		return err
	}
	ui.AddChild(chooser)
	go func() {
		m := <-done
		ui.RemoveChild(chooser)
		if m != nil {
			logging.Info("insertGoalMenu", "chose", m)
			err = insertGoalMenu(ui, replace)
			if err != nil {
				logging.Error("insertGoalMenu", "failed", err)
			}
		} else {
			err := replace(ui)
			if err != nil {
				logging.Error("insertGoalMenu", "replacing failed", err)
			}
		}
	}()
	return nil
}

func InsertVersusMenu(ui gui.WidgetParent, replace func(gui.WidgetParent) error) error {
	// return doChooserMenu(ui, makeChooseVersusMetaMenu, replace, inserter(insertGoalMenu))
	chooser, done, err := makeChooseVersusMetaMenu()
	if err != nil {
		return err
	}
	ui.AddChild(chooser)
	go func() {
		m := <-done
		ui.RemoveChild(chooser)
		if m != nil && len(m) == 1 {
			logging.Info("Versus Menu", "chose", m)
			switch m[0] {
			case "Select House":
				ui.AddChild(MakeGamePanel("versus/basic.lua", nil, map[string]string{"map": "select"}, ""))
			case "Random House":
				ui.AddChild(MakeGamePanel("versus/basic.lua", nil, map[string]string{"map": "random"}, ""))
			case "Continue":
				ui.AddChild(MakeGamePanel("versus/basic.lua", nil, map[string]string{"map": "continue"}, ""))
			default:
				logging.Warn("unknown meta choice", "choice", m[0])
				return
			}
		} else {
			err := replace(ui)
			if err != nil {
				logging.Error("replacing menu", "err", err)
			}
		}
	}()
	return nil
}
