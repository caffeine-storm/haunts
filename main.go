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

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/game"
	"github.com/MobRulesGames/haunts/house"
	"github.com/MobRulesGames/haunts/sound"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/MobRulesGames/memory"
	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/gin"
	"github.com/runningwild/glop/gos"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
	"github.com/runningwild/glop/system"
	glopdebug "github.com/runningwild/glop/debug"

	// Need to pull in all of the actions we define here and not in
	// haunts/game because haunts/game/actions depends on it
	_ "github.com/MobRulesGames/haunts/game/actions"
	_ "github.com/MobRulesGames/haunts/game/ai"

	"github.com/MobRulesGames/haunts/game/status"
)

var (
	sys                       system.System
	datadir                   string
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

func loadAllRegistries() {
	house.LoadAllFurnitureInDir(filepath.Join(datadir, "furniture"))
	house.LoadAllWallTexturesInDir(filepath.Join(datadir, "textures"))
	house.LoadAllRoomsInDir(filepath.Join(datadir, "rooms"))
	house.LoadAllDoorsInDir(filepath.Join(datadir, "doors"))
	house.LoadAllHousesInDir(filepath.Join(datadir, "houses"))
	game.LoadAllGearInDir(filepath.Join(datadir, "gear"))
	game.RegisterActions()
	status.RegisterAllConditions()
}

func init() {
	runtime.LockOSThread()
	sys = system.Make(gos.GetSystemInterface())

	gin.In().SetLogger(slog.New(slog.NewTextHandler(io.Discard, nil)))

	rand.Seed(100)
	datadir = "data-runtime"
	base.SetDatadir(datadir)
	base.Log().Printf("Setting datadir: %s", datadir)
	err := house.SetDatadir(datadir)
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
	Zoom(float64)
}

func draggingAndZooming(ui *gui.Gui, dz draggerZoomer) {
	if ui.FocusWidget() != nil {
		dragging = false
		zooming = false
		sys.HideCursor(false)
		return
	}

	var zoom float64
	if gin.In().GetKey(gin.AnySpace).FramePressAmt() > 0 {
		zoom = gin.In().GetKey(gin.AnyMouseWheelVertical).FramePressAmt()
	}
	dz.Zoom(zoom / 100)

	dz.Zoom(key_map["zoom in"].FramePressAmt() / 20)
	dz.Zoom(-key_map["zoom out"].FramePressAmt() / 20)

	if key_map["drag"].IsDown() != dragging {
		dragging = !dragging
	}
	if dragging {
		mx := gin.In().GetKey(gin.AnyMouseXAxis).FramePressAmt()
		my := gin.In().GetKey(gin.AnyMouseYAxis).FramePressAmt()
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
	draggingAndZooming(ui, editor.GetViewer())
	if ui.FocusWidget() == nil {
		for name := range editors {
			if key_map[fmt.Sprintf("%s editor", name)].FramePressCount() > 0 && ui.FocusWidget() == nil {
				ui.RemoveChild(editor)
				editor_name = name
				editor = editors[editor_name]
				loadAllRegistries()
				editor.Reload()
				ui.AddChild(editor)
			}
		}

		if key_map["save"].FramePressCount() > 0 && chooser == nil {
			path, err := editor.Save()
			if err != nil {
				base.Warn().Printf("Failed to save: %v", err.Error())
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
					base.Warn().Printf("Failed to load: %v", err.Error())
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
			for i := 1; i <= 9; i++ {
				idx := int(gin.AnyKeyPad0.Index) + i
				numericKeyId.Index = gin.KeyIndex(idx)
				if gin.In().GetKey(numericKeyId).FramePressCount() > 0 {
					editor.SelectTab(i - 1)
				}
			}
		}
	}
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

func sysDimser(sys system.System) gui.Dimser {
	return &callsSys{sys: sys}
}

func onHauntsPanic(recoveredValue interface{}) {
	data := debug.Stack()
	base.Error().Printf("PANIC: %v\n", recoveredValue)
	base.Error().Printf("PANIC: %s\n", string(data))
	base.CloseLog()
	fmt.Printf("PANIC: %v\n", recoveredValue)
	fmt.Printf("PANIC: %s\n", string(data))
}

func main() {
	defer func() {
		if r := recover(); r != nil {
			onHauntsPanic(r)
			panic(r)
		}
	}()

	// If 'Version' isn't found, try running 'go -C tools/ run version.go'
	base.Log().Printf("Version %s", Version())
	sys.Startup()
	sound.Init()
	render := render.MakeQueue(func() {
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
	})
	render.StartProcessing()

	base.InitDictionaries(render, sysDimser(sys))
	texture.Init(render)

	base.InitShaders(render)
	runtime.GOMAXPROCS(8)
	ui, err := gui.Make(gin.In(), gui.Dims{Dx: wdx, Dy: wdy}, filepath.Join(datadir, "fonts", "skia.ttf"))
	if err != nil {
		panic(err.Error())
	}
	loadAllRegistries()

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
	editor_name = "room"
	editor = editors[editor_name]

	currentMode := applicationStartupMode
	game.Restart = func() {
		base.Log().Printf("Restarting...")
		ui.RemoveChild(game_box)
		game_box = &lowerLeftTable{gui.MakeAnchorBox(gui.Dims{1024, 768})}
		err = game.InsertStartMenu(game_box)
		if err != nil {
			panic(err)
		}
		ui.AddChild(game_box)
		base.Log().Printf("Restarted")
	}
	game.Restart()

	if base.IsDevel() {
		ui.AddChild(base.MakeConsole())
	}
	sys.Think()
	// Wait until now to create the dictionary because the render thread needs
	// to be running in advance.
	render.Queue(func() {
		ui.Draw()
	})
	render.Purge()

	var profile_output *os.File
	heap_prof_count := 0

	for {
		if key_map["quit"].FramePressCount() != 0 {
			break
		}
		sys.Think()
		render.Queue(func() {
			sys.SwapBuffers()
			ui.Draw()
		})
		render.Purge()

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
							base.Log().Printf("Unable to start CPU profile: %v\n", err)
							profile_output.Close()
							profile_output = nil
						}
						base.Log().Printf("profout: %v\n", profile_output)
					} else {
						base.Log().Printf("Unable to start CPU profile: %v\n", err)
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
						base.Warn().Printf("Unable to write heap profile: %v", err)
					}
				} else {
					base.Warn().Printf("Unable to create heap profile: %v", err)
				}
			}

			if key_map["manual mem"].FramePressCount() > 0 {
				base.Log().Printf(memory.TotalAllocations())
			}

			if key_map["screenshot"].FramePressCount() > 0 {
				// Use gl.ReadPixels to dump a 'screen shot' to screen.png
				fname := filepath.Join(datadir, "screen.png")
				f, err := os.Create(fname)
				if err != nil {
					panic(fmt.Errorf("couldn't os.Create %q: %w", fname, err))
				}
				defer f.Close()

				render.Queue(func() {
					glopdebug.ScreenShot(wdx, wdy, f)
				})
			}

			if key_map["game mode"].FramePressCount()%2 == 1 {
				base.Log().Info("Game mode change", "currentMode", currentMode)
				switch currentMode {
				case applicationStartupMode:
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
			case applicationEditMode:
				editMode(ui)
			case applicationGameMode:
				gameMode(ui)
			case applicationStartupMode:

			}
		}
		// Draw a cursor at the cursor - for testing an osx bug in glop.
		// zx, zy := gin.In().GetCursor("Mouse").Point()
		// render.Queue(func() {
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
