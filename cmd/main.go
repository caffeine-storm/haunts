package cmd

import (
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"runtime/pprof"
	"time"

	// note: if cmd/gen does not exist, you need to run 'go generate ./cmd'
	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/cmd/gen"
	"github.com/MobRulesGames/haunts/console"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/game/actions"
	"github.com/MobRulesGames/haunts/game/ai"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/mrgnet"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/sound"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/MobRulesGames/memory"
	"github.com/caffeine-storm/gl"
	glopdebug "github.com/runningwild/glop/debug"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gos"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/system"
)

//go:generate go run github.com/MobRulesGames/haunts/tools/genversion/cmd ../.git/HEAD ./gen/version.go

var (
	key_map     base.KeyMap
	editors     map[string]house.Editor
	editor      house.Editor
	editor_name string
	anchor      *gui.AnchorBox
	chooser     *gui.FileChooser
	game_box    *lowerLeftTable
	game_panel  *game.GamePanel
)

const (
	wdx = 1024
	wdy = 750
)

func ensureDirectory(filePath string) error {
	return os.MkdirAll(filepath.Dir(filePath), 0o755)
}

func openLogFile(datadir string) (*os.File, error) {
	logFileName := filepath.Join(datadir, "logs", "haunts.log")

	err := ensureDirectory(logFileName)
	if err != nil {
		return nil, fmt.Errorf("couldn't create dir for %q: %w", logFileName, err)
	}

	f, err := os.Create(logFileName)
	if err != nil {
		return nil, fmt.Errorf("couldn't Os.Create %q: %w", logFileName, err)
	}
	return f, nil
}

type lowerLeftTable struct {
	*gui.AnchorBox
}

func (llt *lowerLeftTable) AddChild(w gui.Widget) {
	llt.AnchorBox.AddChild(w, gui.Anchor{
		Wx: 0,
		Wy: 0,
		Bx: 0,
		By: 0,
	})
}

func onHauntsPanic(recoveredValue interface{}) {
	stack := debug.Stack()
	logging.Error("PANIC", "val", recoveredValue, "stack", stack)
	fmt.Printf("PANIC: %v\n", recoveredValue)
	fmt.Printf("PANIC: %s\n", string(stack))
}

// TODO(tmckee): optimize things till we can reliably hit 144
const TargetFPS = 30

func WatchForSlowJobs() *render.JobTimingListener {
	return &render.JobTimingListener{
		OnNotify: func(info *render.JobTimingInfo, attribution string) {
			logging.Warn("slow render job", "runtime", info.RunTime, "queuetime", info.QueueTime, "location", attribution)
		},
		Threshold: time.Second / TargetFPS,
	}
}

func initializeDependencies() (system.System, io.Reader, func()) {
	gin.In().SetLogger(logging.InfoLogger())

	logging.SetLoggingLevel(slog.LevelInfo)
	sysret := system.Make(gos.NewSystemInterface(), gin.In())

	rand.Seed(100)
	base.SetDatadir("data")
	var err error
	logFile, err := openLogFile(base.GetDataDir())
	if err != nil {
		fmt.Printf("warning: couldn't open logfile in %q\nlogging to stdout instead\n", base.GetDataDir())
		logFile = os.Stdout
		err = nil
	}

	// Ignore the returned 'undo' func; it's only really for testing. We don't
	// want to _not_ log to the log file.
	_, logReader := logging.RedirectAndSpy(logFile)

	logging.Info("setting datadir", "datadir", base.GetDataDir())
	err = house.SetDatadir(base.GetDataDir())
	if err != nil {
		panic(err.Error())
	}

	actions.Init()
	ai.Init()

	return sysret, logReader, func() {
		logFile.Close()
	}
}

func Main(argv []string) {
	sys, logReader, cleanup := initializeDependencies()
	defer cleanup()

	var key_binds base.KeyBinds
	err := base.LoadJson(filepath.Join(base.GetDataDir(), "key_binds.json"), &key_binds)
	if err != nil {
		panic(fmt.Errorf("couldn't load key binds: %w", err))
	}
	key_map = key_binds.MakeKeyMap()
	base.SetDefaultKeyMap(key_map)

	defer func() {
		if r := recover(); r != nil {
			onHauntsPanic(r)
			panic(r)
		}
	}()

	// If 'gen.Version' isn't found, try running 'go generate ./cmd'
	logging.Info("version", "version", gen.Version())
	sys.Startup()
	sound.Init()
	queue := render.MakeQueueWithTiming(func(queueState render.RenderQueueState) {
		globals.SetRenderQueueState(queueState)
		sys.CreateWindow(10, 10, wdx, wdy)
		sys.EnableVSync(true)
		err := gl.Init()
		// TODO(tmckee): 0 is from glew.h's GLEW_OK; we should expose a symbol to
		// check and/or a MustInit from the gl package.
		if err != 0 {
			panic(err)
		}
		gl.Enable(gl.BLEND)
		gl.BlendFunc(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA)
	}, WatchForSlowJobs())
	queue.AddErrorCallback(func(_ render.RenderQueueInterface, e error) {
		logging.Error("render-thread error", "err", e)
	})
	queue.StartProcessing()

	texture.Init(queue)

	base.InitShaders(queue)
	runtime.GOMAXPROCS(8)
	ui, err := gui.Make(gui.Dims{Dx: wdx, Dy: wdy}, gin.In())
	if err != nil {
		panic(err.Error())
	}
	base.InitDictionaries(ui)
	registry.LoadAllRegistries()

	// TODO: Might want to be able to reload stuff, but this is sensitive because it
	// is loading textures.  We should probably redo the sprite system so that this
	// is easier to safely handle.
	game.LoadAllEntities()

	// Set up editors
	editors = map[string]house.Editor{
		"room":  house.MakeRoomEditorPanel(),
		"house": house.MakeHouseEditorPanel(),
	}
	for name, editor := range editors {
		path := base.GetStoreVal(fmt.Sprintf("last %s path", name))
		path = filepath.Join(base.GetDataDir(), path)
		if path != "" {
			editor.Load(path)
		}
	}
	// TODO(tmckee): clean: using a string to pick between room editor and house
	// editor is unclear. For now, remember that we start in 'room editor'
	// 'editor mode'; can select room/house editor with 'os+1'/'os+2'.
	editor_name = "room"
	editor = editors[editor_name]

	game.Restart = func() {
		logging.Info("restarting")
		defer logging.Info("restarted")

		ui.RemoveChild(game_box)
		game_box = &lowerLeftTable{gui.MakeAnchorBox(gui.Dims{
			Dx: 1024,
			Dy: 768,
		})}
		if len(argv) > 1 && argv[1] == "lvl1" {
			lvl1scenario := game.Scenario{
				Script:    "Lvl01.lua",
				HouseName: "Lvl_01_Haunted_House",
			}
			var player *game.Player // nil for now
			nodata := map[string]string{}
			nogamekey := mrgnet.GameKey("")
			game_box.AddChild(game.MakeGamePanel(lvl1scenario, player, nodata, nogamekey))
		} else {
			layout, err := game.LoadStartLayoutFromDatadir(base.GetDataDir())
			if err != nil {
				panic(fmt.Errorf("loading start layout failed: %w", err))
			}
			err = game.InsertStartMenu(game_box, *layout)
			if err != nil {
				panic(fmt.Errorf("couldn't insert start menu: %w", err))
			}
		}
		ui.AddChild(game_box)
	}
	game.Restart()

	if base.IsDevel() {
		ui.AddChild(console.MakeConsole(logReader))
	}
	sys.Think()
	// Wait until now to create the dictionary because the render thread needs
	// to be running in advance.
	queue.Queue(func(render.RenderQueueState) {
		ui.Draw()
	})
	queue.Purge()

	runGameLoop(queue, ui, sys)
}

// TODO(tmckee): move everything below this to a game/game_loop.go file.

type applicationMode int

const (
	applicationStartupMode applicationMode = iota
	applicationGameMode
	applicationEditMode
)

func (mode applicationMode) String() string {
	switch mode {
	case applicationStartupMode:
		return "startup"
	case applicationGameMode:
		return "game"
	case applicationEditMode:
		return "edit"
	}
	panic(fmt.Errorf("unknown applicationMode: %v", int(mode)))
}

func gameMode(ui *gui.Gui, sys system.System) {
}

func editMode(ui *gui.Gui, sys system.System) {
	logging.TraceLogger().Trace("editMode entered")
	defer logging.TraceLogger().Trace("editMode returning")

	if ui.FocusWidget() != nil {
		return
	}

	// Did a keypress come in for "change the type of editor"?
	for name := range editors {
		if key_map[fmt.Sprintf("%s editor", name)].FramePressCount() > 0 && ui.FocusWidget() == nil {
			ui.RemoveChild(editor)
			editor_name = name
			editor = editors[editor_name]
			registry.LoadAllRegistries()
			editor.Reload()
			ui.AddChild(editor)
		}
	}

	// Did a keypress come in for "save"?
	if key_map["save"].FramePressCount() > 0 && chooser == nil {
		path, err := editor.Save()
		if err != nil {
			logging.Warn("Failed to save", "error", err.Error)
		}
		if path != "" && err == nil {
			base.SetStoreVal(fmt.Sprintf("last %s path", editor_name), base.TryRelative(base.GetDataDir(), path))
		}
	}

	if key_map["load"].FramePressCount() > 0 && chooser == nil {
		callback := func(path string, err error) {
			ui.DropFocus()
			ui.RemoveChild(anchor)
			chooser = nil
			anchor = nil
			err = editor.Load(path)
			if err != nil {
				logging.Warn("Failed to load", "error", err.Error)
			} else {
				base.SetStoreVal(fmt.Sprintf("last %s path", editor_name), base.TryRelative(base.GetDataDir(), path))
			}
		}
		chooser = gui.MakeFileChooser(filepath.Join(base.GetDataDir(), fmt.Sprintf("%ss", editor_name)), callback, gui.MakeFileFilter(fmt.Sprintf(".%s", editor_name)))
		anchor = gui.MakeAnchorBox(gui.Dims{
			Dx: wdx,
			Dy: wdy,
		})
		anchor.AddChild(chooser, gui.Anchor{
			Wx: 0.5,
			Wy: 0.5,
			Bx: 0.5,
			By: 0.5,
		})
		ui.AddChild(anchor)
		ui.TakeFocus(chooser)
	}

	// Don't select tabs in an editor if we're doing some other sort of command.
	// TODO(tmckee): there's got to be a better way than to poll every key on
	// every frame to find out if any key was pressed? Also, what if someone adds
	// an entry to key_map for, say, numpad3? Wouldn't we just ignore all numpad
	// input at that point???
	for _, v := range key_map {
		if v.FramePressCount() > 0 {
			return
		}
	}

	numericKeyId := gin.AnyKeyPad0
	// Select the tab corresponding to a pressed keypad key.
	for i := 1; i <= 9; i++ {
		idx := int(gin.AnyKeyPad0.Index) + i
		numericKeyId.Index = gin.KeyIndex(idx)
		if gin.In().GetKeyById(numericKeyId).FramePressCount() > 0 {
			editor.SelectTab(i - 1)
		}
	}
}

func runGameLoop(queue render.RenderQueueInterface, ui *gui.Gui, sys system.System) {
	currentMode := applicationStartupMode
	var profile_output *os.File
	heap_prof_count := 0

	var tickCount int64
	for {
		if key_map["quit"].FramePressCount() != 0 {
			break
		}

		renderStart := time.Now()
		sys.Think()
		tickCount += 1
		queue.Queue(func(render.RenderQueueState) {
			gl.Finish()
		})
		queue.Queue(func(render.RenderQueueState) {
			sys.SwapBuffers()
			ui.Draw()
		})
		queue.Purge()
		renderEnd := time.Now()
		logging.Trace("renderwork", "duration", renderEnd.Sub(renderStart), "tick", tickCount)

		for _, child := range game_box.GetChildren() {
			if gp, ok := child.(*game.GamePanel); ok {
				game_panel = gp
			}
		}

		if base.IsDevel() {
			if key_map["cpu profile"].FramePressCount() > 0 {
				if profile_output == nil {
					var err error
					profile_output, err = os.Create(filepath.Join(base.GetDataDir(), "cpu.prof"))
					if err == nil {
						err = pprof.StartCPUProfile(profile_output)
						if err != nil {
							logging.Error("cpu profile", "fail to start", err)
							profile_output.Close()
							profile_output = nil
						}
						logging.Info("profile", "outputfile", profile_output)
					} else {
						logging.Error("cpu profile", "file creation failed", err)
					}
				} else {
					pprof.StopCPUProfile()
					profile_output.Close()
					profile_output = nil
				}
			}

			if key_map["heap profile"].FramePressCount() > 0 {
				out, err := os.Create(filepath.Join(base.GetDataDir(), fmt.Sprintf("heap-%d.prof", heap_prof_count)))
				heap_prof_count++
				if err == nil {
					err = pprof.WriteHeapProfile(out)
					out.Close()
					if err != nil {
						logging.Error("heap profile", "unable to write", err)
					}
				} else {
					logging.Error("heap profile", "unable to create file", err)
				}
			}

			if key_map["manual mem"].FramePressCount() > 0 {
				logging.Info("memory", "allocations", memory.TotalAllocations())
			}

			if key_map["screenshot"].FramePressCount() > 0 {
				// Use gl.ReadPixels to dump a 'screen shot' to screen.png
				fname := filepath.Join(base.GetDataDir(), "screen.png")
				f, err := os.Create(fname)
				if err != nil {
					panic(fmt.Errorf("couldn't os.Create %q: %w", fname, err))
				}
				defer f.Close()

				queue.Queue(func(render.RenderQueueState) {
					glopdebug.ScreenShot(wdx, wdy, f)
				})
			}

			if key_map["game mode"].FramePressCount() > 0 {
				switch currentMode {
				case applicationStartupMode:
					currentMode = applicationGameMode
					fallthrough
				case applicationGameMode:
					ui.RemoveChild(game_box)
					ui.AddChild(editor)
					currentMode = applicationEditMode
				case applicationEditMode:
					ui.RemoveChild(editor)
					ui.AddChild(game_box)
					currentMode = applicationGameMode
				default:
					panic(fmt.Errorf("bad applicationMode: %+v", currentMode))
				}

				if key_map["row up"].FramePressCount() > 0 {
					house.Num_rows += 25
				}
				if key_map["row down"].FramePressCount() > 0 {
					house.Num_rows -= 25
				}
				if key_map["steps up"].FramePressCount() > 0 {
					house.Num_steps++
				}
				if key_map["steps down"].FramePressCount() > 0 {
					house.Num_steps--
				}
				if key_map["noise up"].FramePressCount() > 0 {
					house.Noise_rate += 10
				}
				if key_map["noise down"].FramePressCount() > 0 {
					house.Noise_rate -= 10
				}
				if key_map["foo"].FramePressCount() > 0 {
					house.Foo = (house.Foo + 1) % 2
				}
			}

			switch currentMode {
			case applicationStartupMode:
				currentMode = applicationGameMode
				fallthrough
			case applicationGameMode:
				gameMode(ui, sys)
			case applicationEditMode:
				editMode(ui, sys)
			}
		}
	}
}
