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

func makeChooserFromOptionBasicsFile(path string) (*Chooser, <-chan []Scenario, error) {
	var bops []OptionBasic
	err := base.LoadAndProcessObject(path, "json", &bops)
	if err != nil {
		return nil, nil, err
	}
	var opts []Option
	algorithm.Map(bops, &opts, func(ob OptionBasic) Option { return &ob })
	return MakeChooser(opts)
}

func makeChooseGoalMenu() (*Chooser, <-chan []Scenario, error) {
	path := filepath.Join(base.GetDataDir(), "ui", "start", "versus", "goals.json")
	return makeChooserFromOptionBasicsFile(path)
}

func makeChooseSideMenu() (*Chooser, <-chan []Scenario, error) {
	path := filepath.Join(base.GetDataDir(), "ui", "start", "versus", "side.json")
	return makeChooserFromOptionBasicsFile(path)
}

func makeChooseVersusMetaMenu() (*Chooser, <-chan []Scenario, error) {
	path := filepath.Join(base.GetDataDir(), "ui", "start", "versus", "meta.json")
	return makeChooserFromOptionBasicsFile(path)
}

type (
	chooserMaker func() (*Chooser, <-chan []string, error)
	replacer     func(gui.WidgetParent) error
	inserter     func(gui.WidgetParent, replacer) error
)

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
			logging.Debug("doChooserMenu", "chose", m)
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
			logging.Debug("insertGoalMenu", "chose", m)
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

// TODO(#35): this is not called except in tests. Keeping it around for now.
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
			logging.Debug("Versus Menu", "chose", m)
			// TODO(tmckee:#35): For now, everyone gets the tutorial house :p
			scenario := Scenario{
				Script:    "versus/basic.lua",
				HouseName: "tutorial",
			}
			ui.AddChild(MakeGamePanel(scenario, nil, nil, ""))

			/*
				switch m[0] {
				case "Select House":
					ui.AddChild(MakeGamePanel("versus/basic.lua", nil, map[string]string{"map": "select"}, ""))
				case "Random House":
					ui.AddChild(MakeGamePanel("versus/basic.lua", nil, map[string]string{"map": "random"}, ""))
				case "Continue":
					ui.AddChild(MakeGamePanel("versus/basic.lua", nil, map[string]string{"map": "continue"}, ""))
				default:
					panic(fmt.Errorf("unknown meta choice: %v", m[0]))
				}
			*/

		} else {
			err := replace(ui)
			if err != nil {
				logging.Error("replacing menu", "err", err)
			}
		}
	}()
	return nil
}
