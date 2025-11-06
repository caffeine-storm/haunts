package game

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/MobRulesGames/golua/lua"
	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game/hui"
	"github.com/MobRulesGames/haunts/game/status"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/mrgnet"
	"github.com/MobRulesGames/haunts/sound"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/util/algorithm"
)

type gameScript struct {
	L *lua.State

	// Since the scripts can do anything they want sometimes we want make sure
	// certain things only run when the game is ready for them.
	sync chan struct{}
}

func (gs *gameScript) syncStart() {
	<-gs.sync
}

func (gs *gameScript) syncEnd() {
	gs.sync <- struct{}{}
}

func (gs *gameScript) mustRunString(cmd string) {
	gs.L.MustDoString(cmd)
}

func makeNewLuaState(gp *GamePanel, player *Player, isOnline bool) *lua.State {
	ret := lua.NewState()
	ret.OpenLibs()
	ret.SetExecutionLimit(25000)
	ret.NewTable()

	LuaPushSmartFunctionTable(ret, FunctionTable{
		"ChooserFromFile":                   func() { ret.PushGoFunction(chooserFromFile(gp)) },
		"StartScript":                       func() { ret.PushGoFunction(startScript(gp, player)) },
		"GameOnRound":                       func() { ret.PushGoFunction(doGameOnRound(gp)) },
		"SaveGameState":                     func() { ret.PushGoFunction(saveGameState(gp)) },
		"LoadGameState":                     func() { ret.PushGoFunction(loadGameState(gp)) },
		"DoExec":                            func() { ret.PushGoFunction(doExec(gp)) },
		"SelectEnt":                         func() { ret.PushGoFunction(selectEnt(gp)) },
		"FocusPos":                          func() { ret.PushGoFunction(focusPos(gp)) },
		"FocusZoom":                         func() { ret.PushGoFunction(focusZoom(gp)) },
		"SelectHouse":                       func() { ret.PushGoFunction(selectHouse(gp)) },
		"LoadHouse":                         func() { ret.PushGoFunction(loadHouse(gp)) },
		"SaveStore":                         func() { ret.PushGoFunction(saveStore(gp, player)) },
		"ShowMainBar":                       func() { ret.PushGoFunction(showMainBar(gp, player)) },
		"SpawnEntityAtPosition":             func() { ret.PushGoFunction(spawnEntityAtPosition(gp)) },
		"GetSpawnPointsMatching":            func() { ret.PushGoFunction(getSpawnPointsMatching(gp)) },
		"SpawnEntitySomewhereInSpawnPoints": func() { ret.PushGoFunction(spawnEntitySomewhereInSpawnPoints(gp)) },
		"IsSpawnPointInLos":                 func() { ret.PushGoFunction(isSpawnPointInLos(gp)) },
		"PlaceEntities":                     func() { ret.PushGoFunction(placeEntities(gp)) },
		"RoomAtPos":                         func() { ret.PushGoFunction(roomAtPos(gp)) },
		"SetLosMode":                        func() { ret.PushGoFunction(setLosMode(gp)) },
		"GetAllEnts":                        func() { ret.PushGoFunction(getAllEnts(gp)) },
		"DialogBox":                         func() { ret.PushGoFunction(dialogBox(gp)) },
		"PickFromN":                         func() { ret.PushGoFunction(pickFromN(gp)) },
		"SetGear":                           func() { ret.PushGoFunction(setGear(gp)) },
		"BindAi":                            func() { ret.PushGoFunction(bindAi(gp)) },
		"SetVisibility":                     func() { ret.PushGoFunction(setVisibility(gp)) },
		"EndPlayerInteraction":              func() { ret.PushGoFunction(endPlayerInteraction(gp)) },
		"GetLos":                            func() { ret.PushGoFunction(getLos(gp)) },
		"SetVisibleSpawnPoints":             func() { ret.PushGoFunction(setVisibleSpawnPoints(gp)) },
		"SetCondition":                      func() { ret.PushGoFunction(setCondition(gp)) },
		"SetPosition":                       func() { ret.PushGoFunction(setPosition(gp)) },
		"SetHp":                             func() { ret.PushGoFunction(setHp(gp)) },
		"SetAp":                             func() { ret.PushGoFunction(setAp(gp)) },
		"RemoveEnt":                         func() { ret.PushGoFunction(removeEnt(gp)) },
		"PlayAnimations":                    func() { ret.PushGoFunction(playAnimations(gp)) },
		"PlayMusic":                         func() { ret.PushGoFunction(playMusic(gp)) },
		"StopMusic":                         func() { ret.PushGoFunction(stopMusic(gp)) },
		"SetMusicParam":                     func() { ret.PushGoFunction(setMusicParam(gp)) },
		"PlaySound":                         func() { ret.PushGoFunction(playSound(gp)) },
		"SetWaypoint":                       func() { ret.PushGoFunction(setWaypoint(gp)) },
		"RemoveWaypoint":                    func() { ret.PushGoFunction(removeWaypoint(gp)) },
		"Rand":                              func() { ret.PushGoFunction(randFunc(gp)) },
		"Sleep":                             func() { ret.PushGoFunction(sleepFunc(gp)) },
		"EndGame":                           func() { ret.PushGoFunction(endGameFunc(gp)) },
	})

	ret.SetMetaTable(-2)
	ret.SetGlobal("Script")

	ret.NewTable()
	LuaPushSmartFunctionTable(ret, FunctionTable{
		"Active": func() {
			ret.PushGoFunction(
				func(L *lua.State) int {
					L.PushBoolean(isOnline)
					return 1
				})
		},
		"Side":                func() { ret.PushGoFunction(netSideFunc(gp)) },
		"UpdateState":         func() { ret.PushGoFunction(updateStateFunc(gp)) },
		"UpdateExecs":         func() { ret.PushGoFunction(updateExecsFunc(gp)) },
		"Wait":                func() { ret.PushGoFunction(netWaitFunc(gp)) },
		"LatestStateAndExecs": func() { ret.PushGoFunction(netLatestStateAndExecsFunc(gp)) },
	})
	ret.SetMetaTable(-2)
	ret.SetGlobal("Net")

	registerUtilityFunctions(ret)

	return ret
}

func startGameScript(gp *GamePanel, scenario Scenario, player *Player, data map[string]string, game_key mrgnet.GameKey) {
	// Clear out the panel, now the script can do whatever it wants
	player.Script_path = scenario.Script
	gp.ClearCanvas()
	logging.Debug("startGameScript", "scenario", scenario)
	if scenario.Script != "" && !filepath.IsAbs(scenario.Script) {
		scenario.Script = filepath.Join(base.GetDataDir(), "scripts", filepath.FromSlash(scenario.Script))
	}

	// The game script runs in a separate go routine and functions that need to
	// communicate with the game will do so via channels - DUH why did i even
	// write this comment?
	prog, err := os.ReadFile(scenario.Script)
	if err != nil {
		logging.Error("Unable to load game script", "scenario", scenario, "err", err)
		return
	}

	luaState := makeNewLuaState(gp, player, string(game_key) != "")
	gp.script = &gameScript{
		L:    luaState,
		sync: make(chan struct{}),
	}

	if player.Lua_store != nil {
		loadGameStateRaw(gp, gp.script.L, player.Game_state)
		err := LuaDecodeTable(bytes.NewBuffer(player.Lua_store), gp.script.L, gp.game)
		if err != nil {
			logging.Warn("Error decoding lua state", "err", err)
		}
		gp.script.L.SetGlobal("store")
	} else {
		gp.script.L.NewTable()
		gp.script.L.SetGlobal("store")
	}

	if game_key == "" {
		logging.Warn("!~! DOSTRING !~!", "progpath", scenario.Script)
		gp.script.mustRunString(string(prog))
	}

	logging.Debug("Sync", "gp.script.sync", gp.script.sync)

	// Make sure we don't return from this function until 'gp.Game' is set. We
	// make the game in another goroutine so we'll block on a signalling channel.
	// Why make the game in a separte goroutine? I don't know 🙃.
	// TODO(tmckee): how important is it that some of this setup happens off of
	// the calling goroutine? I'd much rather just call a function.
	gameStartChan := make(chan error)

	go func() {
		if game_key != "" {
			var net_id mrgnet.NetId
			fmt.Sscanf(base.GetStoreVal("netid"), "%d", &net_id)
			var req mrgnet.StatusRequest
			req.Game_key = game_key
			req.Id = net_id
			var resp mrgnet.StatusResponse
			mrgnet.DoAction("status", req, &resp)
			if resp.Err != "" {
				logging.Error("mrgnet.DoAction returned an error response", "err", resp.Err)
				gameStartChan <- fmt.Errorf("gameStart failure: %s", resp.Err)
				return
			}

			// This sets the script on the server
			if resp.Game != nil {
				if resp.Game.Script == nil {
					var req mrgnet.UpdateGameRequest
					req.Id = net_id
					req.Game_key = game_key
					req.Script = prog
					var resp mrgnet.UpdateGameResponse
					err := mrgnet.DoAction("update", req, &resp)
					if err != nil {
						gameStartChan <- fmt.Errorf("unable to make initial update: %w", err)
						return
					}
				} else {
					prog = resp.Game.Script
				}
				res := gp.script.L.DoString(string(prog))
				if res != nil {
					logging.Error("script error", "script_path", scenario.Script, "script_contents", prog, "err", res)
					gameStartChan <- fmt.Errorf("script error: %w", res)
					return
				}
				if len(resp.Game.Execs) > 0 {
					var side Side
					if net_id == resp.Game.Denizens_id {
						side = SideHaunt
					} else {
						side = SideExplorers
					}
					var states []byte
					if (len(resp.Game.Execs)%2 == 1) == (side == SideExplorers) {
						// It is our turn to play, so we grab the last state so that we
						// can do the replay.
						states = resp.Game.After[len(resp.Game.Execs)-1]
					} else {
						// We made the last move, so we can grab what the state of the
						// game was after we finished our turn and then just wait.
						states = resp.Game.After[len(resp.Game.Execs)-1]
					}
					logging.Trace("startGameScript>[unnamed gr]>[network game]>[run server actions]>loadGameStateRaw")
					loadGameStateRaw(gp, gp.script.L, string(states))
					gp.script.mustRunString("OnStartup()")

					gp.game.net.game = resp.Game
					gp.game.net.key = game_key
					gp.game.Turn = len(resp.Game.Execs) + 1

					if net_id == resp.Game.Denizens_id {
						logging.Debug("Setting side to Denizens", "turn", gp.game.Turn)
						gp.game.net.side = SideHaunt
						gp.game.Side = SideHaunt
					} else {
						logging.Debug("Setting side to Intruders", "turn", gp.game.Turn)
						gp.game.net.side = SideExplorers
						gp.game.Side = SideExplorers
					}

					if (len(resp.Game.Execs)%2 == 1) == (side == SideExplorers) {
						gp.game.OnRound(false)
					}
				}
			}
		}
		if gp.game == nil {
			gp.script.L.NewTable()
			for k, v := range data {
				gp.script.L.PushString(k)
				gp.script.L.PushString(v)
				gp.script.L.SetTable(-3)
			}
			gp.script.L.SetGlobal("__data")
			gp.script.L.SetExecutionLimit(250000)

			gp.game = makeGameTheWrongWay(scenario)
			gameStartChan <- nil
			gp.game.script = gp.script
			gp.ClearCanvas()
			gp.AddChild(gp.game.viewer, gui.Anchor{Wx: 0.5, Wy: 0.5, Bx: 0.5, By: 0.5})
			gp.AddChild(MakeOverlay(gp.game), gui.Anchor{Wx: 0.5, Wy: 0.5, Bx: 0.5, By: 0.5})
			if player.No_init {
				gp.script.syncStart()
				logging.Trace("startGameScript>[unnamed gr]>gp.game-is-nil>loadGameStateRaw")
				loadGameStateRaw(gp, gp.script.L, player.Game_state)
				gp.game.script = gp.script
				gp.script.syncEnd()
			} else {
				logging.Trace("startGameScript>[unnamed gr]>gp.game-is-nil>player-has-init")
				gp.script.mustRunString("print(tostring(Init))")
				gp.script.mustRunString("print(tostring(__data))")
				gp.script.mustRunString("Init(__data)")
				gp.script.mustRunString("OnStartup()")
				for i := range gp.game.Ents {
					gp.game.Ents[i].Ai.Activate()
				}
				if gp.game.Side == SideHaunt {
					gp.game.Ai.minions.Activate()
					gp.game.Ai.denizens.Activate()
					gp.game.player_inactive = gp.game.Ai.denizens.Active()
				} else {
					gp.game.Ai.intruders.Activate()
					gp.game.player_inactive = gp.game.Ai.intruders.Active()
				}
			}
		} else {
			gameStartChan <- nil
		}

		if gp.game == nil {
			logging.Error("script failed to load a house during Init()")
		} else {
			gp.game.net.key = game_key
			gp.game.comm.script_to_game <- nil
		}
	}()

	// On success cases, nil will get sent on the channel.
	select {
	case gameStartErr := <-gameStartChan:
		if gameStartErr != nil {
			panic(fmt.Errorf("couldnt' create a game: %w", gameStartErr))
		}
		logging.Debug("game started")
	case <-time.After(1 * time.Second):
		panic(fmt.Errorf("game startup deadline exceeded"))
	}
}

func (gs *gameScript) OnRoundWaiting(g *Game) {
	g.Side = g.net.side
	g.Turn--
	go func() {
		// // round begins automatically
		// <-round_middle
		// for
		//   <-action stuff
		// <- round end
		// <- round end done
		// base.Log().Printf("Game script: %p", gs)
		// base.Log().Printf("Lua state: %p", gs.L)
		// gs.L.SetExecutionLimit(250000)
		// cmd := fmt.Sprintf("RoundStart(%t, %d)", g.Side == SideExplorers, (g.Turn+1)/2)
		// base.Log().Printf("cmd: '%s'", cmd)
		// gs.L.DoString(cmd)

		// signals to the game that we're done with the startup stuff
		g.comm.script_to_game <- nil
		// base.Log().Printf("ScriptComm: Done with RoundStart")

		g.player_inactive = true
		_exec := <-g.comm.game_to_script
		if _exec != nil {
			panic("Got an exec when we shouldn't have gotten one.")
		}

		gs.L.SetExecutionLimit(250000)
		base.DeprecatedLog().Printf("Doing RoundEnd(%t, %d)", g.Side == SideExplorers, (g.Turn+1)/2)
		gs.mustRunString(fmt.Sprintf("RoundEnd(%t, %d)", g.Side == SideExplorers, (g.Turn+1)/2))

		base.DeprecatedLog().Printf("ScriptComm: Starting the RoundEnd phase out")
		g.comm.script_to_game <- nil
		base.DeprecatedLog().Printf("ScriptComm: Starting the RoundEnd phase in")

		// Signal that we're done with the round end
		base.DeprecatedLog().Printf("ScriptComm: Done with the RoundEnd phase in")
		g.comm.script_to_game <- nil
		base.DeprecatedLog().Printf("ScriptComm: Done with the RoundEnd phase out")
	}()
}

// Runs RoundStart
// Lets the game know that the round middle can begin
// Runs RoundEnd
func (gs *gameScript) OnRound(g *Game) {
	logging.Debug("Launching script.RoundStart")
	if (g.Turn%2 == 1) != (g.Side == SideHaunt) {
		logging.Debug("SCRIPT: OnRoundWaiting")
		gs.OnRoundWaiting(g)
		return
	}

	if gs == nil {
		panic(fmt.Errorf("gameScript.OnRound called on nil gameScript!"))
	}
	if gs.L == nil {
		panic(fmt.Errorf("gameScript.OnRound called on invalid gameScript: %v", *gs))
	}

	go func() {
		// // round begins automatically
		// <-round_middle
		// for
		//   <-action stuff
		// <- round end
		// <- round end done
		cmd := fmt.Sprintf("RoundStart(%t, %d)", g.Side == SideExplorers, (g.Turn+1)/2)
		logging.Debug("gameScript.OnRound", "script", gs, "state", gs.L, "cmd", cmd)
		gs.L.SetExecutionLimit(250000)
		gs.mustRunString(cmd)

		// signals to the game that we're done with the startup stuff
		g.comm.script_to_game <- nil
		base.DeprecatedLog().Printf("ScriptComm: Done with RoundStart")

		for {
			logging.Debug("ScriptComm", "state", "waiting to verify action")
			_exec := <-g.comm.game_to_script
			logging.Debug("ScriptComm", "state", "got exec", "_exec", _exec)
			if _exec == nil {
				logging.Debug("ScriptComm", "state", "no more exec")
				break
			}
			logging.Debug("ScriptComm", "state", "verifying action")

			exec := _exec.(ActionExec)
			if vpath := exec.GetPath(); vpath != nil {
				gs.L.SetExecutionLimit(250000)
				exec.Push(gs.L, g)
				gs.L.NewTable()
				for i := range vpath {
					gs.L.PushInteger(int64(i) + 1)
					_, x, y := g.FromVertex(vpath[i])
					LuaPushPoint(gs.L, int(x), int(y))
					gs.L.SetTable(-3)
				}
				logging.Debug("got exec path", "pathlength", len(vpath))
				gs.L.SetGlobal("__path")
				LuaPushEntity(gs.L, g.EntityById(exec.EntityId()))
				gs.L.SetGlobal("__ent")
				cmd = fmt.Sprintf("__truncate = OnMove(__ent, __path)")
				logging.Debug("sending lua cmd", "cmd", cmd)
				func() {
					gs.L.DoString(cmd)
					gs.L.GetGlobal("__truncate")
					truncate := gs.L.ToInteger(-1)
					gs.L.Pop(1)
					logging.Debug("exec.TruncatePath'ing", "len", truncate)
					exec.TruncatePath(truncate)
				}()
			}

			g.comm.script_to_game <- nil

			// The action is sent when it happens, and a nil is sent when it is done
			// being executed, we want to wait until then so that the game is in a
			// stable state before we do anything.
			<-g.comm.game_to_script
			logging.Debug("ScriptComm", "state", "got action secondary")
			// Run OnAction here
			gs.L.SetExecutionLimit(250000)
			exec.Push(gs.L, g)
			str, err := base.ToGobToBase64([]ActionExec{exec})
			if err != nil {
				panic(fmt.Errorf("Unable to encode exec: %w", err))
			}

			gs.L.PushString("__encoded")
			gs.L.PushString(str)
			gs.L.SetTable(-3)

			//      base.Log().Printf("exec: ", LuaStringifyParam(gs.L, -1))
			gs.L.SetGlobal("__exec")
			cmd = fmt.Sprintf("OnAction(%t, %d, %s)", g.Side == SideExplorers, (g.Turn+1)/2, "__exec")
			logging.Debug("sending lua cmd", "cmd", cmd)
			gs.mustRunString(cmd)
			g.comm.script_to_game <- nil
			logging.Debug("ScriptComm", "state", "done with OnAction")
		}

		gs.L.SetExecutionLimit(250000)
		gs.L.DoString(fmt.Sprintf("RoundEnd(%t, %d)", g.Side == SideExplorers, (g.Turn+1)/2))

		logging.Debug("ScriptComm", "state", "starting the RoundEnd phase out")
		g.comm.script_to_game <- nil
		logging.Debug("ScriptComm", "state", "starting the RoundEnd phase in")

		// Signal that we're done with the round end
		logging.Debug("ScriptComm", "state", "finishing the RoundEnd phase in")
		g.comm.script_to_game <- nil
		logging.Debug("ScriptComm", "state", "finishing the RoundEnd phase out")
	}()
}

// Can be called occassionally and will allow a script to progress whenever
// it is ready
func (gp *GamePanel) scriptThinkOnce() {
	if gp.script.L == nil {
		return
	}
	done := false
	s := gp.script.sync
	for !done {
		select {
		// If a script has tried to run a function that requires running during
		// Think then it can run now and we'll wait for it to finish before
		// continuing.
		case s <- struct{}{}:
			<-s
		default:
			done = true
		}
	}
}

// TODO(tmckee:#34): I don't think this is actually used; it's referenced in a
// .lua script that isn't itself referenced.
func startScript(gp *GamePanel, player *Player) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "StartScript", LuaString) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		script := L.ToString(-1)
		player.Script_path = script
		player.No_init = false
		gp.script.syncEnd()
		res := gp.script.L.DoString("Script.SaveStore()")
		gp.script.syncStart()
		if res != nil {
			base.DeprecatedError().Printf("Unable to properly autosave.")
		}
		scenario := Scenario{
			Script:    script,
			HouseName: gp.game.House.Name,
		}
		startGameScript(gp, scenario, player, nil, gp.game.net.key)
		return 0
	}
}

func selectHouse(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SelectHouse") {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		selector, output, err := MakeUiSelectMap(gp)
		if err != nil {
			base.DeprecatedError().Printf("Error selecting map: %v", err)
			return 0
		}
		logging.Trace("selectHouse>abox-addchild>UiSelectMap")
		gp.AddChild(selector, gui.Anchor{0.5, 0.5, 0.5, 0.5})
		gp.script.syncEnd()

		name := <-output
		base.DeprecatedLog().Printf("Received '%s'", name)
		gp.script.syncStart()
		gp.RemoveChild(selector)
		base.DeprecatedLog().Printf("Removed seletor")
		L.PushString(name)
		return 1
	}
}

type totalState struct {
	Game  **Game
	Store []byte
}

func saveGameState(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SaveGameState") {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()

		buf := bytes.NewBuffer(nil)
		L.GetGlobal("store")
		LuaEncodeValue(buf, L, -1)
		L.Pop(1)
		logging.Debug("SaveGameState", "buf.Len()", buf.Len())
		ts := totalState{
			Game:  &gp.game,
			Store: buf.Bytes(),
		}
		str, err := base.ToGobToBase64(ts)
		if err != nil {
			panic(fmt.Errorf("lua>saveGameState: gobbing failure: %w", err))
		}
		logging.Debug("SaveGameState", "delta", len(str)-buf.Len())

		L.PushString(str)
		return 1
	}
}

func doGameOnRound(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "GameOnRound") {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		gp.game.OnRound(false)
		return 0
	}
}

// TODO(tmckee:#24): writing to 'gp' is code smelly
func loadGameStateRaw(gp *GamePanel, L *lua.State, state string) {
	var viewer gui.Widget
	var hv_state house.HouseViewerState
	if gp.game != nil {
		viewer = gp.game.viewer
		hv_state = gp.game.viewer.GetState()
	}
	var ts totalState
	ts.Game = &gp.game
	err := base.FromBase64FromGob(&ts, state)
	if err != nil {
		base.DeprecatedError().Printf("Error decoding game state: %v", err)
		return
	}
	// gp.game = ts.Game
	gp.game.script = gp.script
	LuaDecodeValue(bytes.NewBuffer(ts.Store), L, gp.game)
	if false {
		L.GetGlobal("store")
		// Other side's store on the stack, with our store on top, we're going to
		// take every key/value pair from our store and put it into theirs, then
		// that one becomes ours.
		L.PushNil()
		for L.Next(-2) != 0 {
			// Stack: RemoteStore LocalStore K V
			L.Pop(1)
			// Stack: RemoteStore LocalStore K
			L.PushValue(-1)
			// Stack: RemoteStore LocalStore K K
			L.PushValue(-1)
			// Stack: RemoteStore LocalStore K K K
			L.GetTable(-4)
			// Stack: RemoteStore LocalStore K K V
			L.SetTable(-5)
			// Stack: RemoteStore LocalStore K
			// So we can call next and repeat this process
		}
		// Stack: UpdateRemoteStore LocalStore
		L.Pop(1)
	}
	L.SetGlobal("store")

	gp.RemoveChild(viewer)
	base.DeprecatedLog().Printf("LoadGameStateRaw: Turn = %d, Side = %d", gp.game.Turn, gp.game.Side)
	gp.game.OnRound(false)

	for _, child := range gp.GetChildren() {
		if o, ok := child.(*Overlay); ok {
			gp.RemoveChild(o)
			break
		}
	}
	if viewer != nil {
		gp.game.viewer.SetState(hv_state)
	}
	logging.Trace("loadGameStateRaw>abox-addchild>gameviewer+makeoverlay(game)")
	gp.AddChild(gp.game.viewer, gui.Anchor{0.5, 0.5, 0.5, 0.5})
	gp.AddChild(MakeOverlay(gp.game), gui.Anchor{0.5, 0.5, 0.5, 0.5})
}

func loadGameState(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "LoadGameState", LuaString) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		logging.Trace("loadGameState>loadGameStateRaw")
		loadGameStateRaw(gp, L, L.ToString(-1))
		return 0
	}
}

func doExec(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "DoExec", LuaTable) {
			return 0
		}
		base.DeprecatedLog().Printf("DEBUG: Listing Entities named 'Teen'...")
		for _, ent := range gp.game.Ents {
			if ent.Name == "Teen" {
				x, y := ent.FloorPos()
				base.DeprecatedLog().Printf("DEBUG: %p: (%d, %d)", ent, x, y)
			}
		}
		base.DeprecatedLog().Printf("DEBUG: Done")

		L.PushString("__encoded")
		L.GetTable(-2)
		str := L.ToString(-1)
		L.Pop(1)
		var execs []ActionExec
		base.DeprecatedLog().Printf("Decoding from: '%s'", str)
		err := base.FromBase64FromGob(&execs, str)
		if err != nil {
			base.DeprecatedError().Printf("Error decoding exec: %v", err)
			return 0
		}
		if len(execs) != 1 {
			base.DeprecatedError().Printf("Error decoding exec: Found %d execs instead of exactly 1.", len(execs))
			return 0
		}
		base.DeprecatedLog().Printf("ScriptComm: Exec: %v", execs[0])
		gp.game.comm.script_to_game <- execs[0]
		base.DeprecatedLog().Printf("ScriptComm: Sent exec")
		<-gp.game.comm.game_to_script
		base.DeprecatedLog().Printf("ScriptComm: exec done")
		done := make(chan bool)
		gp.script.syncStart()
		go func() {
			for i := range gp.game.Ents {
				gp.game.Ents[i].Sprite().Wait([]string{"ready", "killed"})
			}
			done <- true
		}()
		gp.script.syncEnd()
		<-done
		return 0
	}
}

func selectEnt(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SelectEnt", LuaEntity) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		ent := LuaToEntity(L, gp.game, -1)
		if ent == nil {
			base.DeprecatedError().Printf("Tried to SelectEnt on a non-existent entity.")
			return 0
		}
		if ent.Side() != gp.game.Side {
			base.DeprecatedError().Printf("Tried to SelectEnt on an entity that's not on the current side.")
			return 0
		}
		gp.game.SelectEnt(ent)
		return 0
	}
}

func focusPos(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "FocusPos", LuaPoint) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		x, y := LuaToPoint(L, -1)
		gp.game.viewer.SetFocusTarget(float32(x), float32(y))
		return 0
	}
}

func focusZoom(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "FocusZoom", LuaFloat) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		gp.game.viewer.SetZoomTarget(L.ToNumber(-1))
		return 0
	}
}

func chooserFromFile(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "ChooserFromFile", LuaString) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		path := filepath.Join(base.GetDataDir(), L.ToString(-1))
		chooser, done, err := makeChooserFromOptionBasicsFile(path)
		if err != nil {
			logging.Error("chooserFromFile: making chooser failed", "err", err)
			return 0
		}
		gp.AddChild(chooser, gui.Anchor{Wx: 0.5, Wy: 0.5, Bx: 0.5, By: 0.5})
		gp.script.syncEnd()

		// TODO(tmckee#42): 'done' can yield nil if we click 'cancel'. If it does,
		// we end up starting a game anyway :(. Need to find a way to plumb a 'go
		// back' callback here.
		res := <-done
		L.NewTable()
		for i, scenario := range res {
			L.PushInteger(int64(i) + 1)
			L.PushString(scenario.Script)
			L.SetTable(-3)
		}
		gp.script.syncStart()
		gp.RemoveChild(chooser)
		return 1
	}
}

func loadHouse(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		logging.Debug("in ur loadHouse")
		if !LuaCheckParamsOk(L, "LoadHouse", LuaString) {
			logging.Error("LuaCheckParamsOk rejection: LoadHouse(LuaString)")
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()

		name := L.ToString(-1)
		logging.Debug("in ur loadHouse", "loadin ur house", name)
		def := house.MakeHouseFromName(name)
		if def == nil || len(def.Floors) == 0 {
			logging.Error("no house with given name", "name", name)
			return 0
		}
		// TODO(tmckee): this is a bug; we will get a nil sprite manager from
		// GetSpriteManager because gp.game isn't initialized ... right?
		gp.game = MakeGame(def, gp.game.GetSpriteManager())
		gp.game.viewer.Edit_mode = true
		gp.game.script = gp.script

		logging.Trace("loadHouse>abox-addchild>gameviewer+makeoverlay(game)")
		gp.ClearCanvas()
		gp.AddChild(gp.game.viewer, gui.Anchor{Wx: 0.5, Wy: 0.5, Bx: 0.5, By: 0.5})
		gp.AddChild(MakeOverlay(gp.game), gui.Anchor{Wx: 0.5, Wy: 0.5, Bx: 0.5, By: 0.5})

		logging.Debug("done 'LouseHouse'")
		return 0
	}
}

func showMainBar(gp *GamePanel, player *Player) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "ShowMainBar", LuaBoolean) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		show := L.ToBoolean(-1)

		// Remove it regardless of whether or not we want to hide it
		for _, child := range gp.GetChildren() {
			if child == gp.main_bar {
				gp.RemoveChild(child)
				break
			}
		}

		if show {
			var err error
			gp.main_bar, err = MakeMainBar(gp.game)
			if err != nil {
				LuaDoError(L, err.Error())
				return 0
			}
			logging.Trace("showMainBar>abox-addchild>mainbar")
			gp.AddChild(gp.main_bar, gui.Anchor{0.5, 0, 0.5, 0})
			system, err := MakeSystemMenu(gp, player)
			if err != nil {
				LuaDoError(L, err.Error())
				return 0
			}
			gp.AddChild(system, gui.Anchor{1, 1, 1, 1})
		}
		return 0
	}
}

func spawnEntityAtPosition(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SpawnEntityAtPosition", LuaString, LuaPoint) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		name := L.ToString(-2)
		x, y := house.BoardSpaceUnitPair(LuaToPoint(L, -1))
		ent := MakeEntity(name, gp.game)
		if gp.game.SpawnEntity(ent, x, y) {
			LuaPushEntity(L, ent)
		} else {
			L.PushNil()
		}
		return 1
	}
}

func getSpawnPointsMatching(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "GetSpawnPointsMatching", LuaString) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		spawn_pattern := L.ToString(-1)
		re, err := regexp.Compile(spawn_pattern)
		if err != nil {
			LuaDoError(L, fmt.Sprintf("Failed to compile regexp '%s': %v", spawn_pattern, err))
			return 0
		}
		L.NewTable()
		count := 0
		for _, sp := range gp.game.House.Floors[0].Spawns {
			if !re.MatchString(sp.Name) {
				continue
			}
			count++
			L.PushInteger(int64(count))
			LuaPushSpawnPoint(L, gp.game, sp)
			L.SetTable(-3)
		}
		return 1
	}
}

func spawnEntitySomewhereInSpawnPoints(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SpawnEntitySomewhereInSpawnPoints", LuaString, LuaArray, LuaBoolean) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		name := L.ToString(-3)
		hidden := L.ToBoolean(-1)
		L.Pop(1)

		var tx, ty house.BoardSpaceUnit

		var count int64 = 0
		L.PushNil()
		ent := MakeEntity(name, gp.game)
		var side Side
		if ent.Side() == SideExplorers {
			side = SideHaunt
		}
		if ent.Side() == SideHaunt {
			side = SideExplorers
		}
		for L.Next(-2) != 0 {
			sp := LuaToSpawnPoint(L, gp.game, -1)
			L.Pop(1)
			if sp == nil {
				continue
			}
			sx, sy := sp.FloorPos()
			sdx, sdy := sp.Dims()
			for x := sx; x < sx+sdx; x++ {
				for y := sy; y < sy+sdy; y++ {
					if gp.game.IsCellOccupied(x, y) {
						continue
					}
					if hidden && ent.game.TeamLos(side, x, y, 1, 1) {
						continue
					}
					// This will choose a random position from all positions and giving
					// all positions an equal chance of being chosen.
					count++
					if gp.game.Rand.Int63()%count == 0 {
						tx = x
						ty = y
					}
				}
			}
		}
		if count == 0 {
			base.DeprecatedError().Printf("Unable to find an available position to spawn %s", name)
			return 0
		}
		if ent == nil {
			base.DeprecatedError().Printf("Cannot make an entity named '%s', no such thing.", name)
			return 0
		}
		if gp.game.SpawnEntity(ent, tx, ty) {
			LuaPushEntity(L, ent)
		} else {
			L.PushNil()
		}
		return 1
	}
}

func isSpawnPointInLos(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "IsSpawnPointInLos", LuaSpawnPoint, LuaString) {
			return 0
		}
		spawn := LuaToSpawnPoint(L, gp.game, -2)
		side_str := L.ToString(-1)
		var in_los bool
		switch side_str {
		case "intruders":
			in_los = gp.game.TeamLos(SideExplorers, spawn.X, spawn.Y, spawn.Dx, spawn.Dy)
		case "denizens":
			in_los = gp.game.TeamLos(SideHaunt, spawn.X, spawn.Y, spawn.Dx, spawn.Dy)
		default:
			base.DeprecatedError().Printf("Unexpected side in IsSpawnPointInLos: '%s'", side_str)
			return 0
		}
		L.PushBoolean(in_los)
		return 1
	}
}

func placeEntities(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "PlaceEntities", LuaString, LuaTable, LuaInteger, LuaInteger) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		L.PushNil()
		var names []string
		var costs []int
		for L.Next(-4) != 0 {
			L.PushInteger(1)
			L.GetTable(-2)
			names = append(names, L.ToString(-1))
			L.Pop(1)
			L.PushInteger(2)
			L.GetTable(-2)
			costs = append(costs, L.ToInteger(-1))
			L.Pop(2)
		}
		ep, done, err := MakeEntityPlacer(gp.game, names, costs, L.ToInteger(-2), L.ToInteger(-1), L.ToString(-4))
		if err != nil {
			logging.Error("placeEntities: MakeEntityPlacer failed", "err", err)
			return 0
		}
		logging.Trace("placeEntities>abox-addchild>entityPlacer")
		gp.AddChild(ep, gui.Anchor{})
		for _, kid := range gp.GetChildren() {
			logging.Trace("placeEntities", "kid", kid.String())
		}
		gp.script.syncEnd()
		ents := <-done
		L.NewTable()
		for i := range ents {
			L.PushInteger(int64(i) + 1)
			LuaPushEntity(L, ents[i])
			L.SetTable(-3)
		}
		gp.script.syncStart()
		gp.RemoveChild(ep)
		return 1
	}
}

func roomAtPos(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "RoomAtPos", LuaPoint) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		intx, inty := LuaToPoint(L, -1)
		room, _, _ := gp.game.House.Floors[0].RoomFurnSpawnAtPos(house.BoardSpaceUnitPair(intx, inty))
		for i, r := range gp.game.House.Floors[0].Rooms {
			if r == room {
				L.PushInteger(int64(i))
				return 1
			}
		}
		LuaDoError(L, fmt.Sprintf("Tried to get the room at position (%d,%d), but there is no room there.", intx, inty))
		return 0
	}
}

func getAllEnts(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "GetAllEnts") {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		logging.Debug("LUA>Script.GetAllEnts", "numents", len(gp.game.Ents))
		L.NewTable()
		for i := range gp.game.Ents {
			L.PushInteger(int64(i) + 1)
			LuaPushEntity(L, gp.game.Ents[i])
			L.SetTable(-3)
		}
		return 1
	}
}

func dialogBox(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if L.GetTop() == 1 {
			if !LuaCheckParamsOk(L, "DialogBox", LuaString) {
				return 0
			}
		} else {
			if !LuaCheckParamsOk(L, "DialogBox", LuaString, LuaTable) {
				return 0
			}
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		path := L.ToString(1)
		var args map[string]string
		if L.GetTop() > 1 {
			args = make(map[string]string)
			L.PushValue(2)
			L.PushNil()
			for L.Next(-2) != 0 {
				args[L.ToString(-2)] = L.ToString(-1)
				L.Pop(1)
			}
			L.Pop(1)
		}
		box, output, err := MakeDialogBox(filepath.ToSlash(path), args)
		if err != nil {
			logging.Error("couldn't MakeDialogBox", "err", err)
			return 0
		}
		logging.Trace("dialogBox>abox-addchild>dialogBox")
		gp.AddChild(box, gui.Anchor{Wx: 0.5, Wy: 0.5, Bx: 0.5, By: 0.5})
		gp.script.syncEnd()

		var choices []string
		for choice := range output {
			choices = append(choices, choice)
		}
		logging.Debug("dialogBox", "choices", choices)

		gp.script.syncStart()
		gp.RemoveChild(box)
		L.NewTable()
		for i, choice := range choices {
			L.PushInteger(int64(i) + 1)
			L.PushString(choice)
			L.SetTable(-3)
		}
		return 1
	}
}

type iconWithText struct {
	Name string
	Icon texture.Object
	Data interface{}
}

func (c *iconWithText) Draw(hovered, selected, selectable bool, region gui.Region) {
	var f float64
	switch {
	case selected:
		f = 1.0
	case hovered || selectable:
		f = 0.6
	default:
		f = 0.4
	}
	gl.Color4d(f, f, f, 1)
	c.Icon.Data().RenderNatural(region.X, region.Y)
	if hovered && selectable {
		if selected {
			gl.Color4d(1, 0, 0, 0.3)
		} else {
			gl.Color4d(1, 0, 0, 1)
		}
		gl.Disable(gl.TEXTURE_2D)
		gl.Begin(gl.LINES)
		x := region.X + 1
		y := region.Y + 1
		x2 := region.X + region.Dx - 1
		y2 := region.Y + region.Dy - 1

		gl.Vertex2i(x, y)
		gl.Vertex2i(x, y2)

		gl.Vertex2i(x, y2)
		gl.Vertex2i(x2, y2)

		gl.Vertex2i(x2, y2)
		gl.Vertex2i(x2, y)

		gl.Vertex2i(x2, y)
		gl.Vertex2i(x, y)
		gl.End()
	}
}

func (c *iconWithText) Think(dt int64) {
}

func pickFromN(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "PickFromN", LuaInteger, LuaInteger, LuaTable) {
			return 0
		}
		min := L.ToInteger(-3)
		max := L.ToInteger(-2)
		var options []hui.Option
		var option_names []string
		L.PushNil()
		for L.Next(-2) != 0 {
			name := L.ToString(-2)
			option_names = append(option_names, name)
			path := L.ToString(-1)
			if !filepath.IsAbs(path) {
				path = filepath.Join(base.GetDataDir(), path)
			}
			option := iconWithText{
				Name: name,
				Icon: texture.Object{},
			}
			option.Icon.ResetPath(base.Path(path))
			options = append(options, &option)
			L.Pop(1)
		}
		var selector hui.Selector
		if min == 1 && max == 1 {
			selector = hui.SelectExactlyOne
		} else {
			selector = hui.SelectInRange(min, max)
		}
		var chooser *hui.RosterChooser
		done := make(chan struct{})
		on_complete := func(m map[int]bool) {
			gp.RemoveChild(chooser)
			L.NewTable()
			count := 0
			for i := range options {
				if m[i] {
					count++
					L.PushInteger(int64(count))
					L.PushString(option_names[i])
					L.SetTable(-3)
				}
			}
			done <- struct{}{}
		}
		logging.Debug("HHHHUUUUUUUUUUUUUUUUIIIIIIIIIIIIII!")
		chooser = hui.MakeRosterChooser(options, selector, on_complete, nil)
		gp.script.syncStart()
		gp.AddChild(chooser, gui.Anchor{0.5, 0.5, 0.5, 0.5})
		gp.script.syncEnd()
		<-done
		return 1
	}
}

func setGear(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SetGear", LuaEntity, LuaString) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		gear_name := L.ToString(-1)
		ent := LuaToEntity(L, gp.game, -2)
		if ent == nil {
			base.DeprecatedError().Printf("Called SetGear on an invalid entity.")
			return 0
		}
		L.PushBoolean(ent.SetGear(gear_name))
		return 1
	}
}

// bindAi(target, source)
// bindAi("denizen", "denizen.lua")
// bindAi("intruder", "intruder.lua")
// bindAi("minions", "minions.lua")
// bindAi(ent, "fudgecake.lua")
// special sources: "human", "inactive", and in the future: "net"
// special targets: "denizen", "intruder", "minions", or an entity table
func bindAi(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "BindAi", LuaAnything, LuaString) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		source := L.ToString(-1)
		if L.IsTable(-2) {
			L.PushString("id")
			L.GetTable(-3)
			target := EntityId(L.ToInteger(-1))
			L.Pop(1)
			ent := gp.game.EntityById(target)
			if ent == nil {
				base.DeprecatedError().Printf("Referenced an entity with id == %d which doesn't exist.", target)
				return 0
			}
			ent.Ai_file_override = base.Path(filepath.Join(base.GetDataDir(), "ais", filepath.FromSlash(L.ToString(-1))))
			ent.LoadAi()
			return 0
		}
		target := L.ToString(-2)
		switch target {
		case "denizen":
			switch source {
			case "human":
				gp.game.Ai.denizens = inactiveAi{}
			case "net":
				base.DeprecatedError().Printf("bindAi('denizen', 'net') is not implemented.")
				return 0
			default:
				gp.game.Ai.denizens = nil
				path := filepath.Join(base.GetDataDir(), "ais", source)
				gp.game.Ai.Path.Denizens = path
				ai_maker(path, gp.game, nil, &gp.game.Ai.denizens, DenizensAi)
				if gp.game.Ai.denizens == nil {
					gp.game.Ai.denizens = inactiveAi{}
				}
			}
		case "intruder":
			switch source {
			case "human":
				gp.game.Ai.intruders = inactiveAi{}
			case "net":
				base.DeprecatedError().Printf("bindAi('intruder', 'net') is not implemented.")
				return 0
			default:
				gp.game.Ai.intruders = nil
				path := filepath.Join(base.GetDataDir(), "ais", source)
				gp.game.Ai.Path.Intruders = path
				ai_maker(path, gp.game, nil, &gp.game.Ai.intruders, IntrudersAi)
				if gp.game.Ai.intruders == nil {
					gp.game.Ai.intruders = inactiveAi{}
				}
			}
		case "minions":
			gp.game.Ai.minions = nil
			path := filepath.Join(base.GetDataDir(), "ais", source)
			gp.game.Ai.Path.Minions = path
			ai_maker(path, gp.game, nil, &gp.game.Ai.minions, MinionsAi)
			if gp.game.Ai.minions == nil {
				gp.game.Ai.minions = inactiveAi{}
			}
		default:
			base.DeprecatedError().Printf("Specified unknown Ai target '%s'", target)
			return 0
		}

		return 0
	}
}

func setVisibility(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SetVisibility", LuaString) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		side_str := L.ToString(-1)
		var side Side
		switch side_str {
		case "denizens":
			side = SideHaunt
		case "intruders":
			side = SideExplorers
		default:
			base.DeprecatedError().Printf("Cannot pass '%s' as first parameter of setVisibility()", side_str)
			return 0
		}
		base.DeprecatedLog().Printf("SetVisibility: %s", side_str)
		gp.game.SetVisibility(side)
		return 0
	}
}

func endPlayerInteraction(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "EndPlayerInteraction") {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		gp.game.player_inactive = true
		return 0
	}
}

func saveStore(gp *GamePanel, player *Player) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SaveStore") {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		UpdatePlayer(player, gp.script.L)
		str, err := base.ToGobToBase64(gp.game)
		if err != nil {
			base.DeprecatedError().Printf("Error gobbing game state: %v", err)
			return 0
		}
		player.Game_state = str
		player.Name = "autosave"
		err = SavePlayer(player)
		if err != nil {
			base.DeprecatedWarn().Printf("Unable to save player: %v", err)
		}
		return 0
	}
}

func getLos(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "GetLos", LuaEntity) {
			return 0
		}
		ent := LuaToEntity(L, gp.game, -1)
		if ent == nil {
			base.DeprecatedError().Printf("Tried to GetLos on an invalid entity.")
			return 0
		}
		if ent.los == nil || ent.los.grid == nil {
			base.DeprecatedError().Printf("Tried to GetLos on an entity without vision.")
			return 0
		}
		L.NewTable()
		count := 0
		for x := range ent.los.grid {
			for y := range ent.los.grid[x] {
				if ent.los.grid[x][y] {
					count++
					L.PushInteger(int64(count))
					LuaPushPoint(L, x, y)
					L.SetTable(-3)
				}
			}
		}
		return 1
	}
}

func setVisibleSpawnPoints(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SetVisibleSpawnPoints", LuaString, LuaString) {
			return 0
		}
		switch L.ToString(-2) {
		case "denizens":
			gp.game.Los_spawns.Denizens.Pattern = L.ToString(-1)
		case "intruders":
			gp.game.Los_spawns.Intruders.Pattern = L.ToString(-1)
		default:
			base.DeprecatedError().Printf("First parameter to SetVisibleSpawnPoints must be either 'denizens' or 'intruders'.")
			return 0
		}
		return 1
	}
}

func setCondition(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SetCondition", LuaEntity, LuaString, LuaBoolean) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		ent := LuaToEntity(L, gp.game, -3)
		if ent == nil {
			base.DeprecatedWarn().Printf("Tried to SetCondition on an entity that doesn't exist.")
			return 0
		}
		if ent.Stats == nil {
			base.DeprecatedWarn().Printf("Tried to SetCondition on an entity that doesn't have stats.")
			return 0
		}
		name := L.ToString(-2)
		if L.ToBoolean(-1) {
			ent.Stats.ApplyCondition(status.MakeCondition(name))
		} else {
			ent.Stats.RemoveCondition(name)
		}
		return 0
	}
}

func setPosition(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SetPosition", LuaEntity, LuaPoint) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		ent := LuaToEntity(L, gp.game, -2)
		if ent == nil {
			base.DeprecatedWarn().Printf("Tried to SetPosition on an entity that doesn't exist.")
			return 0
		}
		x, y := LuaToPoint(L, -1)
		ent.X = float64(x)
		ent.Y = float64(y)
		return 0
	}
}

func setHp(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SetHp", LuaEntity, LuaInteger) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		ent := LuaToEntity(L, gp.game, -2)
		if ent == nil {
			base.DeprecatedWarn().Printf("Tried to SetHp on an entity that doesn't exist.")
			return 0
		}
		if ent.Stats == nil {
			base.DeprecatedWarn().Printf("Tried to SetHp on an entity that doesn't have stats.")
			return 0
		}
		ent.Stats.SetHp(L.ToInteger(-1))
		return 0
	}
}

func setAp(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SetAp", LuaEntity, LuaInteger) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		ent := LuaToEntity(L, gp.game, -2)
		if ent == nil {
			base.DeprecatedWarn().Printf("Tried to SetAp on an entity that doesn't exist.")
			return 0
		}
		if ent.Stats == nil {
			base.DeprecatedWarn().Printf("Tried to SetAp on an entity that doesn't have stats.")
			return 0
		}
		ent.Stats.SetAp(L.ToInteger(-1))
		return 0
	}
}

func removeEnt(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "RemoveEnt", LuaEntity) {
			return 0
		}
		ent := LuaToEntity(L, gp.game, -1)
		if ent == nil {
			base.DeprecatedWarn().Printf("Tried to RemoveEnt on an entity that doesn't exist.")
			return 0
		}
		removed := false
		for i := range gp.game.Ents {
			if gp.game.Ents[i] == ent {
				gp.game.Ents[i] = gp.game.Ents[len(gp.game.Ents)-1]
				gp.game.Ents = gp.game.Ents[0 : len(gp.game.Ents)-1]
				gp.game.viewer.RemoveDrawable(ent)
				removed = true
				break
			}
		}
		if !removed {
			base.DeprecatedWarn().Printf("Tried to RemoveEnt an entity that wasn't in the game.")
		}
		return 0
	}
}

func playAnimations(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "PlayAnimations", LuaEntity, LuaArray) {
			return 0
		}
		gp.script.syncStart()
		ent := LuaToEntity(L, gp.game, -2)
		if ent == nil {
			base.DeprecatedWarn().Printf("Tried to PlayAnimation on an entity that doesn't exist.")
			return 0
		}
		gp.script.syncEnd()
		ent.Sprite().Wait([]string{"ready", "killed"})
		if ent.Sprite().AnimState() == "ready" {
			L.PushNil()
			for L.Next(-2) != 0 {
				ent.Sprite().Command(L.ToString(-1))
				L.Pop(1)
			}
			ent.Sprite().Wait([]string{"ready", "killed"})
		}
		return 0
	}
}

func playMusic(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "PlayMusic", LuaString) {
			return 0
		}
		sound.PlayMusic(L.ToString(-1))
		return 0
	}
}

func stopMusic(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "StopMusic", LuaString) {
			return 0
		}
		sound.StopMusic()
		return 0
	}
}

func setMusicParam(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SetMusicParam", LuaString, LuaFloat) {
			return 0
		}
		sound.SetMusicParam(L.ToString(-2), L.ToNumber(-1))
		return 0
	}
}

func playSound(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "PlaySound", LuaString) {
			return 0
		}
		sound.PlaySound(L.ToString(-1), 1.0)
		return 0
	}
}

func removeWaypoint(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "RemoveWaypoint", LuaString) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		hit := false
		name := L.ToString(-1)
		for i := 0; i < len(gp.game.Waypoints); i++ {
			if gp.game.Waypoints[i].Name == name {
				hit = true
				gp.game.viewer.RemoveFloorDrawable(&gp.game.Waypoints[i])
				l := len(gp.game.Waypoints)
				gp.game.Waypoints[i] = gp.game.Waypoints[l-1]
				gp.game.Waypoints = gp.game.Waypoints[0 : l-1]
			}
		}
		if !hit {
			base.DeprecatedError().Printf("RemoveWaypoint on waypoint '%s' which doesn't exist.", name)
			return 0
		}
		return 0
	}
}

func setWaypoint(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SetWaypoint", LuaString, LuaString, LuaPoint, LuaFloat) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()

		var wp Waypoint
		side_str := L.ToString(-3)
		switch side_str {
		case "intruders":
			wp.Side = SideExplorers
		case "denizens":
			wp.Side = SideHaunt
		default:
			base.DeprecatedError().Printf("Specified '%s' for the side parameter in SetWaypoint, must be 'intruders' or 'denizens'.", side_str)
			return 0
		}
		wp.Name = L.ToString(-4)
		// Remove any existing waypoint by the same name
		algorithm.Choose(&gp.game.Waypoints, func(w Waypoint) bool {
			return w.Name != wp.Name
		})
		px, py := LuaToPoint(L, -2)
		wp.X = float64(px)
		wp.Y = float64(py)
		wp.Radius = L.ToNumber(-1)
		gp.game.Waypoints = append(gp.game.Waypoints, wp)
		return 0
	}
}

func setLosMode(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "SetLosMode", LuaString, LuaAnything) {
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		side_str := L.ToString(-2)
		var mode_str string
		if L.IsString(-1) {
			mode_str = L.ToString(-1)
		} else {
			mode_str = "rooms"
		}
		var side Side
		switch side_str {
		case "denizens":
			side = SideHaunt
		case "intruders":
			side = SideExplorers
		default:
			base.DeprecatedError().Printf("Cannot pass '%s' as first parameters of setLosMode()", side_str)
			return 0
		}
		switch mode_str {
		case "none":
			gp.game.SetLosMode(side, LosModeNone, nil)
		case "blind":
			gp.game.SetLosMode(side, LosModeBlind, nil)
		case "all":
			gp.game.SetLosMode(side, LosModeAll, nil)
		case "entities":
			gp.game.SetLosMode(side, LosModeEntities, nil)
		case "rooms":
			if !L.IsTable(-1) {
				base.DeprecatedError().Printf("The last parameter to setLosMode should be an array of rooms if mode == 'rooms'")
				return 0
			}
			L.PushNil()
			all_rooms := gp.game.House.Floors[0].Rooms
			var rooms []*house.Room
			for L.Next(-2) != 0 {
				index := L.ToInteger(-1)
				if index < 0 || index > len(all_rooms) {
					base.DeprecatedError().Printf("Tried to reference room #%d which doesn't exist.", index)
					continue
				}
				rooms = append(rooms, all_rooms[index])
				L.Pop(1)
			}
			gp.game.SetLosMode(side, LosModeRooms, rooms)

		default:
			base.DeprecatedError().Printf("Unknown los mode '%s'", mode_str)
			return 0
		}
		return 0
	}
}

func randFunc(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "Rand", LuaInteger) {
			return 0
		}
		n := L.ToInteger(-1)
		L.PushInteger(int64(gp.game.Rand.Int63()%int64(n)) + 1)
		return 1
	}
}

func sleepFunc(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "Sleep", LuaFloat) {
			return 0
		}
		seconds := L.ToNumber(-1)
		time.Sleep(time.Microsecond * time.Duration(1000000*seconds))
		return 1
	}
}

func endGameFunc(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "EndGame") {
			return 0
		}
		gp.game.Ents = nil
		gp.game.Think(1) // This should clean things up
		Restart()
		return 1
	}
}

func netSideFunc(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "Side") {
			return 0
		}
		if gp.game.net.game == nil {
			// If we haven't gotten the game yet that is because it is the first
			// turn, so it must be the Denizens turn.
			L.PushString("Denizens")
			return 1
		}
		var net_id mrgnet.NetId
		fmt.Sscanf(base.GetStoreVal("netid"), "%d", &net_id)
		switch {
		case gp.game.net.game.Denizens_id == net_id:
			L.PushString("Denizens")
		case gp.game.net.game.Intruders_id == net_id:
			L.PushString("Intruders")
		default:
			base.DeprecatedError().Printf("Asked for a net side, but don't know the side.")
			L.PushString("Unknown")
		}
		return 1
	}
}

func updateStateFunc(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "UpdateState", LuaString) {
			return 0
		}
		if gp.game.net.key == "" {
			base.DeprecatedError().Printf("Tried to UpdateState in a non-Net game.")
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		var net_id mrgnet.NetId
		fmt.Sscanf(base.GetStoreVal("netid"), "%d", &net_id)
		var req mrgnet.UpdateGameRequest
		req.Id = net_id
		req.Game_key = gp.game.net.key
		req.Round = (gp.game.Turn+1)/2 - 1 // Server is base-0, lua is base-1
		req.Intruders = gp.game.Side == SideExplorers
		req.Before = []byte(L.ToString(-1))
		var resp mrgnet.UpdateGameResponse
		mrgnet.DoAction("update", req, &resp)
		if resp.Err != "" {
			base.DeprecatedError().Printf("Error updating game state: %v", resp.Err)
			return 0
		}
		base.DeprecatedLog().Printf("UpdateState: Turn = %d, Side = %d", gp.game.Turn, gp.game.Side)
		return 0
	}
}

func updateExecsFunc(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "UpdateExecs", LuaString, LuaArray) {
			return 0
		}
		if gp.game.net.key == "" {
			base.DeprecatedError().Printf("Tried to UpdateExecs in a non-Net game.")
			return 0
		}
		buf := bytes.NewBuffer(nil)
		err := LuaEncodeValue(buf, L, -1)
		if err != nil {
			base.DeprecatedError().Printf("Unable to serialize execs: %v", err)
			return 0
		}
		gp.script.syncStart()
		defer gp.script.syncEnd()
		var net_id mrgnet.NetId
		fmt.Sscanf(base.GetStoreVal("netid"), "%d", &net_id)
		var req mrgnet.UpdateGameRequest
		req.Id = net_id
		req.Game_key = gp.game.net.key
		req.Round = (gp.game.Turn+1)/2 - 1 // Server is base-0, lua is base-1
		req.Intruders = gp.game.Side == SideExplorers
		req.Execs = buf.Bytes()
		req.After = []byte(L.ToString(-2))
		var resp mrgnet.UpdateGameResponse
		mrgnet.DoAction("update", req, &resp)
		if resp.Err != "" {
			base.DeprecatedError().Printf("Error updating game execs: %v", resp.Err)
			return 0
		}
		base.DeprecatedLog().Printf("Successfully update game execs: %v", gp.game.net.key)
		return 0
	}
}

func netWaitFunc(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "Wait") {
			return 0
		}
		if gp.game.net.key == "" {
			base.DeprecatedError().Printf("Tried to Wait in a non-net game.")
			return 0
		}
		var net_id mrgnet.NetId
		fmt.Sscanf(base.GetStoreVal("netid"), "%d", &net_id)
		var req mrgnet.StatusRequest
		req.Game_key = gp.game.net.key
		req.Id = net_id
		req.Sizes_only = true
		for {
			var resp mrgnet.StatusResponse
			mrgnet.DoAction("status", req, &resp)
			if resp.Err != "" {
				base.DeprecatedError().Printf("%s", resp.Err)
				return 0
			}
			expect := gp.game.Turn + 1
			if len(resp.Game.Before) == len(resp.Game.Execs) && len(resp.Game.Before) == expect {
				base.DeprecatedLog().Printf("Found the expected %d states", expect)
				req.Sizes_only = false
				mrgnet.DoAction("status", req, &resp)
				if resp.Err != "" {
					base.DeprecatedError().Printf("%s", resp.Err)
					return 0
				}
				break
			}
			base.DeprecatedLog().Printf("Found %d instead of %d states", len(resp.Game.Execs), expect)
			time.Sleep(time.Second * 5)
		}
		return 0
	}
}

func netLatestStateAndExecsFunc(gp *GamePanel) lua.LuaGoFunction {
	return func(L *lua.State) int {
		if !LuaCheckParamsOk(L, "LatestStateAndExecs") {
			return 0
		}
		if gp.game.net.key == "" {
			base.DeprecatedError().Printf("Tried to get LatestStateAndExecs in a non-net game.")
			return 0
		}
		var net_id mrgnet.NetId
		fmt.Sscanf(base.GetStoreVal("netid"), "%d", &net_id)
		var req mrgnet.StatusRequest
		req.Game_key = gp.game.net.key
		req.Id = net_id
		var resp mrgnet.StatusResponse
		mrgnet.DoAction("status", req, &resp)
		if resp.Err != "" {
			base.DeprecatedError().Printf("%s", resp.Err)
			return 0
		}
		if len(resp.Game.Before) != len(resp.Game.Execs) {
			base.DeprecatedError().Printf("Not the same number of States and Execss")
			return 0
		}
		state := resp.Game.Before[len(resp.Game.Before)-1]
		L.PushString(string(state))
		buf := bytes.NewBuffer(resp.Game.Execs[len(resp.Game.Execs)-1])
		gp.script.syncStart()
		LuaDecodeValue(buf, L, gp.game)
		gp.script.syncEnd()
		return 2
	}
}

// Ripped from game/ai/ai.go - should probably sync up with it
func registerUtilityFunctions(L *lua.State) {
	L.Register("print", func(L *lua.State) int {
		var res string
		n := L.GetTop()
		for i := -n; i < 0; i++ {
			res += LuaStringifyParam(L, i) + " "
		}
		logging.Info("GameScript::print", "msg", res)
		return 0
	})
}
