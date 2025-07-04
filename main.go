package main

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

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/console"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/haunts/registry"
	"github.com/MobRulesGames/haunts/sound"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/MobRulesGames/memory"
	"github.com/go-gl-legacy/gl"
	glopdebug "github.com/runningwild/glop/debug"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gos"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/system"

	// Need to pull in all of the actions we define here and not in
	// haunts/game because haunts/game/actions depends on it
	_ "github.com/MobRulesGames/haunts/game/actions"
	_ "github.com/MobRulesGames/haunts/game/ai"
)

var (
	sys                       system.System
	datadir                   string
	logFile                   *os.File
	logReader                 io.Reader
	key_map                   base.KeyMap
	editors                   map[string]house.Editor
	editor                    house.Editor
	editor_name               string
	ui                        *gui.Gui
	anchor                    *gui.AnchorBox
	chooser                   *gui.FileChooser
	wdx, wdy                  int
	game_box                  *lowerLeftTable
	game_panel                *game.GamePanel
	zooming, dragging, hiding bool
)

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

func ensureDirectory(filePath string) error {
	return os.MkdirAll(filepath.Dir(filePath), 0755)
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

func init() {
	// TODO(tmckee): uhhh... shouldn't we _not_ call this here?
	runtime.LockOSThread()
	gin.In().SetLogger(logging.InfoLogger())

	logging.SetLoggingLevel(slog.LevelInfo)
	sys = system.Make(gos.NewSystemInterface(), gin.In())

	rand.Seed(100)
	datadir = "data"
	base.SetDatadir(datadir)
	var err error
	logFile, err = openLogFile(base.GetDataDir())
	if err != nil {
		fmt.Printf("warning: couldn't open logfile in %q\nlogging to stdout instead\n", base.GetDataDir())
		logFile = os.Stdout
		err = nil
	}

	// Ignore the returned 'undo' func; it's only really for testing. We don't
	// want to _not_ log to the log file.
	_, logReader = logging.RedirectAndSpy(logFile)

	logging.Info("Setting datadir", "datadir", datadir)
	err = house.SetDatadir(datadir)
	if err != nil {
		panic(err.Error())
	}

	var key_binds base.KeyBinds
	base.LoadJson(filepath.Join(datadir, "key_binds.json"), &key_binds)
	key_map = key_binds.MakeKeyMap()
	base.SetDefaultKeyMap(key_map)

	wdx = 1024
	wdy = 750
}

type draggerZoomer interface {
	Drag(float64, float64)
	SetZoom(float32)
	GetZoom() float32
}

func draggingAndZooming(ui *gui.Gui, dz draggerZoomer) {
	if ui.FocusWidget() != nil {
		dragging = false
		zooming = false
		sys.HideCursor(false)
		return
	}

	// TODO(#29): figure out the scale/style that makes sense here
	var zoom float64 = float64(dz.GetZoom())
	delta := key_map["zoom"].FramePressSum()
	logging.Info("draggingAndZooming", "old zoom", zoom, "delta", delta)
	if delta != 0 {
		logging.Info("draggingAndZooming", "setting", zoom+delta)
		dz.SetZoom(float32(zoom + delta))
	}

	if key_map["drag"].IsDown() != dragging {
		dragging = !dragging
	}
	if dragging {
		mx := gin.In().GetKeyById(gin.AnyMouseXAxis).FramePressAmt()
		my := gin.In().GetKeyById(gin.AnyMouseYAxis).FramePressAmt()
		if mx != 0 || my != 0 {
			dz.Drag(-mx, my)
		}
	}

	if (dragging || zooming) != hiding {
		hiding = (dragging || zooming)
		sys.HideCursor(hiding)
	}
}

func gameMode(ui *gui.Gui) {
	if game_panel != nil && game_panel.Active() {
		draggingAndZooming(ui, game_panel.GetViewer())
	}
}

func editMode(ui *gui.Gui) {
	logging.TraceLogger().Trace("editMode entered")
	draggingAndZooming(ui, editor.GetViewer())
	if ui.FocusWidget() == nil {
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
				base.SetStoreVal(fmt.Sprintf("last %s path", editor_name), base.TryRelative(datadir, path))
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
					base.SetStoreVal(fmt.Sprintf("last %s path", editor_name), base.TryRelative(datadir, path))
				}
			}
			chooser = gui.MakeFileChooser(filepath.Join(datadir, fmt.Sprintf("%ss", editor_name)), callback, gui.MakeFileFilter(fmt.Sprintf(".%s", editor_name)))
			anchor = gui.MakeAnchorBox(gui.Dims{wdx, wdy})
			anchor.AddChild(chooser, gui.Anchor{0.5, 0.5, 0.5, 0.5})
			ui.AddChild(anchor)
			ui.TakeFocus(chooser)
		}

		// Don't select tabs in an editor if we're doing some other sort of command
		ok_to_select := true
		for _, v := range key_map {
			if v.FramePressCount() > 0 {
				ok_to_select = false
				break
			}
		}
		if ok_to_select {
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
	}

	logging.TraceLogger().Trace("editMode returning")
}

type lowerLeftTable struct {
	*gui.AnchorBox
}

func (llt *lowerLeftTable) AddChild(w gui.Widget) {
	llt.AnchorBox.AddChild(w, gui.Anchor{0, 0, 0, 0})
}

type callsSys struct {
	sys system.System
}

func (c *callsSys) Dims() gui.Dims {
	_, _, dx, dy := c.sys.GetWindowDims()
	return gui.Dims{
		Dx: dx,
		Dy: dy,
	}
}

func onHauntsPanic(recoveredValue interface{}) {
	stack := debug.Stack()
	logging.Error("PANIC", "val", recoveredValue, "stack", stack)
	logFile.Close()
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

func main() {
	defer func() {
		if r := recover(); r != nil {
			onHauntsPanic(r)
			panic(r)
		}
	}()

	// If 'Version' isn't found, try running 'go -C tools/ run version.go'
	logging.Info("version", "version", Version())
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
		path = filepath.Join(datadir, path)
		if path != "" {
			editor.Load(path)
		}
	}
	// TODO(tmckee): clean: using a string to pick between room editor and house
	// editor is unclear. For now, remember that we start in 'room editor'
	// 'editor mode'; can select room/house editor with 'os+1'/'os+2'.
	editor_name = "room"
	editor = editors[editor_name]

	currentMode := applicationStartupMode
	game.Restart = func() {
		logging.Info("Restarting...")
		ui.RemoveChild(game_box)
		game_box = &lowerLeftTable{gui.MakeAnchorBox(gui.Dims{1024, 768})}
		layout, err := game.LoadStartLayoutFromDatadir(datadir)
		if err != nil {
			panic(fmt.Errorf("loading start layout failed: %w", err))
		}
		err = game.InsertStartMenu(game_box, *layout)
		if err != nil {
			panic(fmt.Errorf("couldn't insert start menu: %w", err))
		}
		ui.AddChild(game_box)
		logging.Info("Restarted")
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

	var profile_output *os.File
	heap_prof_count := 0

	var tickCount int64
	for {
		glopdebug.LogAndClearGlErrors(logging.WarnLogger())

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
					profile_output, err = os.Create(filepath.Join(datadir, "cpu.prof"))
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
				out, err := os.Create(filepath.Join(datadir, fmt.Sprintf("heap-%d.prof", heap_prof_count)))
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
				fname := filepath.Join(datadir, "screen.png")
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
				gameMode(ui)
			case applicationEditMode:
				editMode(ui)
			}
		}
		// Draw a cursor at the cursor - for testing an osx bug in glop.
		// zx, zy := gin.In().GetCursor("Mouse").Point()
		// render.Queue(func(render.RenderQueueState) {
		//   gl.Color4ub(255, 0, 0, 255)
		//   gl.Begin(gl.LINES)
		//   {
		//     gl.Vertex2i(int32(zx-25), int32(zy))
		//     gl.Vertex2i(int32(zx+25), int32(zy))
		//     gl.Vertex2i(int32(zx), int32(zy-25))
		//     gl.Vertex2i(int32(zx), int32(zy+25))
		//   }
		//   gl.End()
		// })
	}
}
