package actions

import (
	"encoding/gob"
	"path/filepath"

	"github.com/MobRulesGames/golua/lua"
	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/status"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gui"
)

func registerSummonActions() map[string]func() game.Action {
	summons_actions := make(map[string]*SummonActionDef)
	base.RemoveRegistry("actions-summons_actions")
	base.RegisterRegistry("actions-summons_actions", summons_actions)
	base.RegisterAllObjectsInDir("actions-summons_actions", filepath.Join(base.GetDataDir(), "actions", "summons"), ".json", "json")
	makers := make(map[string]func() game.Action)
	for name := range summons_actions {
		cname := name
		makers[cname] = func() game.Action {
			a := SummonAction{Defname: cname}
			base.GetObject("actions-summons_actions", &a)
			if a.Ammo > 0 {
				a.Current_ammo = a.Ammo
			} else {
				a.Current_ammo = -1
			}
			return &a
		}
	}
	return makers
}

func init() {
	game.RegisterActionMakers(registerSummonActions)
	gob.Register(&SummonAction{})
	gob.Register(&summonExec{})
}

// Summon Actions target a single cell, are instant, and unreadyable.
type SummonAction struct {
	Defname string
	*SummonActionDef
	summonActionTempData

	Current_ammo int
}
type SummonActionDef struct {
	Name         string
	Kind         status.Kind
	Personal_los bool
	Ap           int
	Ammo         int // 0 = infinity
	Range        int
	Ent_name     string
	Animation    string
	Conditions   []string
	Texture      texture.Object
	Sounds       map[string]string
}
type summonActionTempData struct {
	ent *game.Entity
	// TODO(tmckee#47): use BoardSpaceUnit here too
	cx, cy int
	spawn  *game.Entity
}
type summonExec struct {
	game.BasicActionExec
	Pos int
}

func (exec summonExec) Push(L *lua.State, g *game.Game) {
	exec.BasicActionExec.Push(L, g)
	if L.IsNil(-1) {
		return
	}
	_, x, y := g.FromVertex(exec.Pos)
	L.PushString("Pos")
	game.LuaPushPoint(L, int(x), int(y))
	L.SetTable(-3)
}

func (a *SummonAction) SoundMap() map[string]string {
	return a.Sounds
}

func (a *SummonAction) Push(L *lua.State) {
	L.NewTable()
	L.PushString("Type")
	L.PushString("Summon")
	L.SetTable(-3)
	L.PushString("Name")
	L.PushString(a.Name)
	L.SetTable(-3)
	L.PushString("Ap")
	L.PushInteger(int64(a.Ap))
	L.SetTable(-3)
	L.PushString("Entity")
	L.PushString(a.Ent_name)
	L.SetTable(-3)
	L.PushString("Los")
	L.PushBoolean(a.Personal_los)
	L.SetTable(-3)
	L.PushString("Range")
	L.PushInteger(int64(a.Range))
	L.SetTable(-3)
	L.PushString("Ammo")
	if a.Current_ammo == -1 {
		L.PushInteger(1000)
	} else {
		L.PushInteger(int64(a.Current_ammo))
	}
	L.SetTable(-3)

}

func (a *SummonAction) AP() int {
	return a.Ap
}
func (a *SummonAction) FloorPos() (house.BoardSpaceUnit, house.BoardSpaceUnit) {
	return house.BoardSpaceUnitPair(a.cx, a.cy)
}
func (a *SummonAction) Dims() (house.BoardSpaceUnit, house.BoardSpaceUnit) {
	return 1, 1
}
func (a *SummonAction) String() string {
	return a.Name
}
func (a *SummonAction) Icon() *texture.Object {
	return &a.Texture
}
func (a *SummonAction) Readyable() bool {
	return false
}
func (a *SummonAction) Preppable(ent *game.Entity, g *game.Game) bool {
	return a.Current_ammo != 0 && ent.Stats.ApCur() >= a.Ap
}
func (a *SummonAction) Prep(ent *game.Entity, g *game.Game) bool {
	if !a.Preppable(ent, g) {
		return false
	}
	a.ent = ent
	return true
}
func (a *SummonAction) HandleInput(ctx gui.EventHandlingContext, group gui.EventGroup, g *game.Game) (bool, game.ActionExec) {
	if mpos, ok := ctx.UseMousePosition(group); ok {
		bx, by := g.GetViewer().WindowToBoard(mpos.X, mpos.Y)
		bx += 0.5
		by += 0.5
		if bx < 0 {
			bx--
		}
		if by < 0 {
			by--
		}
		a.cx = int(bx)
		a.cy = int(by)
	}

	if group.IsPressed(gin.AnyMouseLButton) {
		if g.IsCellOccupied(a.cx, a.cy) {
			return true, nil
		}
		if a.Personal_los && !a.ent.HasLos(a.cx, a.cy, 1, 1) {
			return true, nil
		}
		if a.ent.Stats.ApCur() >= a.Ap {
			var exec summonExec
			exec.SetBasicData(a.ent, a)
			exec.Pos = a.ent.Game().ToVertex(house.BoardSpaceUnitPair(a.cx, a.cy))
			return true, &exec
		}
		return true, nil
	}
	return false, nil
}
func (a *SummonAction) RenderOnFloor() {
	if a.ent == nil {
		return
	}
	ex, ey := a.ent.FloorPos()
	if dist(int(ex), int(ey), a.cx, a.cy) <= a.Range && a.ent.HasLos(a.cx, a.cy, 1, 1) {
		gl.Color4ub(255, 255, 255, 200)
	} else {
		gl.Color4ub(255, 64, 64, 200)
	}
	base.EnableShader("box")
	base.SetUniformF("box", "dx", 1)
	base.SetUniformF("box", "dy", 1)
	base.SetUniformI("box", "temp_invalid", 0)
	(&texture.Object{}).Data().Render(float64(a.cx), float64(a.cy), 1, 1)
	base.EnableShader("")
}
func (a *SummonAction) Cancel() {
	a.summonActionTempData = summonActionTempData{}
}
func (a *SummonAction) Maintain(dt int64, g *game.Game, ae game.ActionExec) game.MaintenanceStatus {
	if ae != nil {
		exec := ae.(*summonExec)
		ent := g.EntityById(exec.Ent)
		if ent == nil {
			base.DeprecatedError().Printf("Got a summon action without a valid entity.")
			return game.Complete
		}
		a.ent = ent
		_, bsucx, bsucy := a.ent.Game().FromVertex(exec.Pos)
		a.cx, a.cy = int(bsucx), int(bsucy)
		a.ent.Stats.ApplyDamage(-a.Ap, 0, status.Unspecified)
		a.spawn = game.MakeEntity(a.Ent_name, a.ent.Game())
		if a.Current_ammo > 0 {
			a.Current_ammo--
		}
	}
	if a.ent.Sprite().State() == "ready" {
		a.ent.TurnToFace(a.cx, a.cy)
		a.ent.Sprite().Command(a.Animation)
		a.spawn.Stats.OnBegin()
		a.ent.Game().SpawnEntity(a.spawn, a.cx, a.cy)
		return game.Complete
	}
	return game.InProgress
}
func (a *SummonAction) Interrupt() bool {
	return true
}
