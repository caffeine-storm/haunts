package game

import (
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/mrgnet"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/strmanip"
)

type GamePanel struct {
	// TODO(tmckee:#38): hide this field from client code; too much coupling
	// ensues when everyone is expected to manipulate this themselves.
	*gui.AnchorBox

	main_bar *MainBar

	// Keep track of this so we know how much time has passed between
	// calls to Think()
	last_think int64

	complete gui.Widget

	script *gameScript
	game   *Game
}

func (gp *GamePanel) SetLosModeAll() {
	render.MustNotBeOnRenderThread()
	gp.game.SetLosMode(SideExplorers, LosModeAll, nil)
	gp.game.SetLosMode(SideHaunt, LosModeAll, nil)
}

// TODO(#36): add a render.RenderQueueInterface to the parameter list and store
// it with the GamePanel so that the GamePanel can forward it to ... stuff.
func MakeGamePanel(scenario Scenario, p *Player, data map[string]string, game_key mrgnet.GameKey) *GamePanel {
	var gp GamePanel
	if p == nil {
		p = &Player{}
	}
	if scenario.Script == "" {
		scenario.Script = p.Script_path
	}
	startGameScript(&gp, scenario, p, data, game_key)
	return &gp
}

// Returns  true iff the game panel has an active game with a viewer already
// installed.
func (gp *GamePanel) Active() bool {
	return gp.game != nil && gp.game.House != nil && gp.game.viewer != nil
}

func (gp *GamePanel) Think(ui *gui.Gui, t int64) {
	gp.scriptThinkOnce()
	gp.AnchorBox.Think(ui, t)
	if !gp.Active() {
		return
	}
	if gp.game != nil {
		gp.game.modal = (ui.FocusWidget() != nil)
	}

	if gp.last_think == 0 {
		gp.last_think = t
	}
	dt := t - gp.last_think
	gp.last_think = t
	logging.TraceBracket(func() {
		gp.game.Think(dt)
	})

	if gp.main_bar != nil {
		if gp.game.selected_ent != nil {
			gp.main_bar.SelectEnt(gp.game.selected_ent)
		} else {
			gp.main_bar.SelectEnt(gp.game.hovered_ent)
		}
	}
}

func (gp *GamePanel) Draw(region gui.Region, ctx gui.DrawingContext) {
	region.PushClipPlanes()
	defer region.PopClipPlanes()
	logging.Trace("GamePanel.Draw", "anchorbox.Children", strmanip.Show(gp.AnchorBox.GetChildren()))
	gp.AnchorBox.Draw(region, ctx)
}

func (gp *GamePanel) Respond(ui *gui.Gui, group gui.EventGroup) bool {
	if gp.AnchorBox.Respond(ui, group) {
		return true
	}
	if !gp.Active() {
		return false
	}

	if group.PrimaryEvent().IsRelease() {
		return false
	}

	if mpos, ok := ui.UseMousePosition(group); ok {
		if gp.game.hovered_ent != nil {
			gp.game.hovered_ent.hovered = false
		}
		gp.game.hovered_ent = nil
		for i := range gp.game.Ents {
			fx, fy := gp.game.Ents[i].FPos()
			wx, wy := gp.game.viewer.BoardToWindow(float32(fx), float32(fy))
			if gp.game.Ents[i].Stats != nil && gp.game.Ents[i].Stats.HpCur() <= 0 {
				continue // Don't bother showing dead units
			}
			x := wx - int(gp.game.Ents[i].last_render_width/2)
			y := wy
			x2 := wx + int(gp.game.Ents[i].last_render_width/2)
			y2 := wy + int(150*gp.game.Ents[i].last_render_width/100)
			if mpos.X >= x && mpos.X <= x2 && mpos.Y >= y && mpos.Y <= y2 {
				if gp.game.hovered_ent != nil {
					gp.game.hovered_ent.hovered = false
				}
				gp.game.hovered_ent = gp.game.Ents[i]
				gp.game.hovered_ent.hovered = true
			}
		}
	}

	if group.IsPressed(gin.AnyEscape) {
		if gp.game.selected_ent != nil {
			switch gp.game.Action_state {
			case noAction:
				gp.game.selected_ent.selected = false
				gp.game.selected_ent.hovered = false
				gp.game.selected_ent = nil
				return true

			case preppingAction:
				gp.game.SetCurrentAction(nil)
				return true

			case doingAction:
				// Do nothing - we don't cancel an action that's in progress
			}
		}
	}

	if gp.game.Action_state == noAction {
		if group.IsPressed(gin.AnyMouseLButton) {
			if gp.game.hovered_ent != nil && gp.game.hovered_ent.Side() == gp.game.Side {
				if gp.game.selected_ent != nil {
					gp.game.selected_ent.selected = false
				}
				gp.game.selected_ent = gp.game.hovered_ent
				gp.game.selected_ent.selected = true
			}
			return true
		}
	}

	if gp.game.Action_state == preppingAction {
		consumed, exec := gp.game.current_action.HandleInput(ui, group, gp.game)
		if consumed {
			if exec != nil {
				gp.game.current_exec = exec
				// TODO: Should send the exec across the wire here
			}
			return true
		}
	}

	// After this point all events that we check for require that we have a
	// selected entity
	if gp.game.selected_ent == nil {
		return false
	}
	if gp.game.Action_state == noAction || gp.game.Action_state == preppingAction {
		if len(group.Events) == 1 && group.Events[0].Key.Id().Index >= '1' && group.Events[0].Key.Id().Index <= '9' {
			index := int(group.Events[0].Key.Id().Index - '1')
			if index >= 0 && index < len(gp.game.selected_ent.Actions) {
				action := gp.game.selected_ent.Actions[index]
				if action != gp.game.current_action && action.Prep(gp.game.selected_ent, gp.game) {
					gp.game.SetCurrentAction(action)
				}
			}
		}
	}

	return false
}

func (gp *GamePanel) GetViewer() house.Viewer {
	return gp.game.viewer
}
