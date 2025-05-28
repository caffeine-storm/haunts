package texture

import (
	"context"
	"fmt"
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"maps"
	"os"
	"slices"
	"sync"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/logging"
	"github.com/MobRulesGames/mathgl"
	"github.com/MobRulesGames/memory"
	"github.com/go-gl-legacy/gl"
	"github.com/go-gl-legacy/glu"
	"github.com/runningwild/glop/imgmanip"
	"github.com/runningwild/glop/render"
)

type Object struct {
	// Only exported so that the registry stuff can load paths from disk. Use
	// 'GetPath()' if you want to read, ResetPath() if you want to write.
	Path base.Path
	data *Data
}

func (o *Object) GetPath() string {
	return getPath(o)
}

func (o *Object) ResetPath(newpath base.Path) {
	resetPath(o, newpath)
}

func (o *Object) Data() *Data {
	if o.data == nil {
		var err error
		o.data, err = LoadFromPath(string(o.Path))
		if err != nil {
			panic(fmt.Errorf("texture manager LoadFromPath failed: path: %q: %w", o.Path, err))
		}
	}

	onAccess(o.data)
	return o.data
}

type Data struct {
	dx, dy   int
	texture  gl.Texture // only accessed on the render thread so no mutex needed
	accessed int        // protected from concurrent access by the manager's mutex
}

func (d *Data) Dx() int {
	return d.dx
}
func (d *Data) Dy() int {
	return d.dy
}

func (d *Data) isLoaded() bool {
	render.MustBeOnRenderThread()
	return d.texture != 0
}

var textureList uint

func setupTextureList(queue render.RenderQueueInterface) {
	queue.Queue(func(render.RenderQueueState) {
		textureList = gl.GenLists(1)
		gl.NewList(textureList, gl.COMPILE)

		gl.PushAttrib(gl.COLOR_BUFFER_BIT)
		gl.Enable(gl.BLEND)
		gl.BlendFuncSeparate(gl.SRC_ALPHA, gl.ONE_MINUS_SRC_ALPHA, gl.ZERO, gl.ONE)
		gl.Begin(gl.QUADS)
		// bottom-left
		gl.TexCoord2d(0, 0)
		gl.Vertex2i(0, 0)

		// top-left
		gl.TexCoord2d(0, 1)
		gl.Vertex2i(0, 1)

		// top-right
		gl.TexCoord2d(1, 1)
		gl.Vertex2i(1, 1)

		// bottom-right
		gl.TexCoord2d(1, 0)
		gl.Vertex2i(1, 0)
		gl.End()
		gl.PopAttrib()

		gl.EndList()
	})
}

// Renders the texture on a quad at the texture's natural size.
func (d *Data) RenderNatural(x, y int) {
	logging.Trace("render natural", "x", x, "y", y, "dx", d.dx, "dy", d.dy)
	d.Render(float64(x), float64(y), float64(d.dx), float64(d.dy))
}

func Render(x, y, dx, dy float64) {
	var run, op mathgl.Mat4
	run.Identity()
	op.Translation(float32(x), float32(y), 0)
	run.Multiply(&op)
	op.Scaling(float32(dx), float32(dy), 1)
	run.Multiply(&op)

	render.WithMultMatrixInMode(&run, render.MatrixModeModelView, func() {
		gl.Enable(gl.TEXTURE_2D)
		gl.CallList(textureList)
	})
}

func (d *Data) Render(x, y, dx, dy float64) {
	if textureList == 0 {
		logging.Warn("Data.Render called before textureList setup!")
		return
	}
	d.Bind()
	defer d.Unbind()
	Render(x, y, dx, dy)
}

func (d *Data) RenderAdvanced(x, y, dx, dy, rot float64, flip bool) {
	if textureList == 0 {
		logging.Warn("Data.RenderAdvanced called before textureList setup!")
		return
	}
	d.Bind()
	defer d.Unbind()
	RenderAdvanced(x, y, dx, dy, rot, flip)
}

func RenderAdvanced(x, y, dx, dy, rot float64, flip bool) {
	var run, op mathgl.Mat4
	run.Identity()
	op.Translation(float32(x), float32(y), 0)
	run.Multiply(&op)
	op.Translation(float32(dx/2), float32(dy/2), 0)
	run.Multiply(&op)
	op.RotationZ(float32(rot))
	run.Multiply(&op)
	if flip {
		op.Translation(float32(dx/2), float32(-dy/2), 0)
		run.Multiply(&op)
		op.Scaling(float32(-dx), float32(dy), 1)
		run.Multiply(&op)
	} else {
		op.Translation(float32(-dx/2), float32(-dy/2), 0)
		run.Multiply(&op)
		op.Scaling(float32(dx), float32(dy), 1)
		run.Multiply(&op)
	}

	render.WithMultMatrixInMode(&run, render.MatrixModeModelView, func() {
		gl.Enable(gl.TEXTURE_2D)
		gl.CallList(textureList)
	})
}

func (d *Data) Bind() {
	if d.isLoaded() {
		d.texture.Bind(gl.TEXTURE_2D)
	} else {
		if error_texture == 0 {
			makeErrorTexture()
		}
		error_texture.Bind(gl.TEXTURE_2D)
	}
}

func (d *Data) Unbind() {
	gl.Texture(0).Unbind(gl.TEXTURE_2D)
}

// Launch this in its own go-routine if you want to occassionally
// delete textures that haven't been used in a while.
func (m *Manager) Scavenger() {
	for {
		time.Sleep(time.Minute)
		unused := []string{}
		m.mutex.RLock()
		for s, d := range m.registry {
			if m.generation-d.accessed >= 2 {
				unused = append(unused, s)
			}
		}
		m.mutex.RUnlock()

		m.mutex.Lock()
		m.generation++
		if len(unused) == 0 {
			m.mutex.Unlock()
			continue
		}

		var unused_data []*Data
		for _, s := range unused {
			unused_data = append(unused_data, m.registry[s])
			m.deleted[s] = m.registry[s]
			delete(m.registry, s)
		}
		manager.renderQueue.Queue(func(render.RenderQueueState) {
			for _, d := range unused_data {
				d.texture.Delete()
				d.texture = 0
			}
		})
		m.mutex.Unlock()
	}
}

func LoadFromPath(path string) (*Data, error) {
	if manager == nil {
		panic("need to call texure.Init before texture.LoadFromPath")
	}

	return manager.LoadFromPath(path)
}

func getPath(o *Object) string {
	if manager == nil {
		panic("need to call texure.Init before texture.getPath")
	}

	return manager.getPath(o)
}

func resetPath(o *Object, newpath base.Path) {
	if manager == nil {
		panic("need to call texure.Init before texture.resetPath")
	}

	manager.resetPath(o, newpath)
}

func onAccess(d *Data) {
	if manager == nil {
		panic("need to call texure.Init before texture.onAccess")
	}

	manager.onAccess(d)
}

type loadRequest struct {
	path string
	data *Data
}

type Manager struct {
	// Currently loaded/loading textures are in the registry
	registry map[string]*Data

	// If a texture has been deleted it is moved here so that if it gets
	// reloaded it will be loaded into the same texture object it was in
	// before.
	deleted map[string]*Data

	// If a texture is in the process of being loaded, there will be a
	// corresponding entry in 'inFlight'.
	inFlight map[string]bool

	// Rendering queue/context that will be used for all gl operations.
	renderQueue render.RenderQueueInterface

	// Clients can request to block until a given texture path has been loaded.
	// This map tracks a set of channels needed for signalling when a texture
	// loads.
	loadWaiters map[string]chan bool

	// Instead of keeping track of access time, we just keep track of how many
	// times the scavenger has seen something without it being accessed.
	// generation is incremented every time the scavenger loop runs, and any
	// time a texture is accessed it is updated with the current generation.
	generation int

	// Protects all fields above this.
	mutex sync.RWMutex

	load struct {
		requests chan loadRequest
		count    int
		// Protects other members of the 'load' substruct.
		mutex sync.Mutex
	}
}

// TODO(tmckee:#28): package level state is causing problems w.r.t. data races
// during tests. We should return a manager instance from Init instead of
// relying on this variable.
var manager *Manager

func Init(renderQueue render.RenderQueueInterface) {
	manager = &Manager{
		registry:    make(map[string]*Data),
		deleted:     make(map[string]*Data),
		inFlight:    make(map[string]bool),
		renderQueue: renderQueue,
		generation:  0,
		loadWaiters: make(map[string]chan bool),
	}
	manager.load.requests = make(chan loadRequest, 10)

	go manager.Scavenger()

	pipe := make(chan loadRequest, 10)
	// We want to be able to handle any number of incoming load requests, so
	// we have one go-routine collect them all and send them along pipe any
	// time someone is ready to receive one.
	go func() {
		var rs []loadRequest
		var send chan loadRequest
		var hold loadRequest
		for {
			select {
			case r := <-manager.load.requests:
				rs = append(rs, r)
			case send <- hold:
				rs = rs[1:]
			}
			if len(rs) > 0 {
				send = pipe
				hold = rs[0]
			} else {
				// If send is nil then it will effectively be excluded from the select
				// statement above.  This is good since we won't have anything to send
				// until we get more requests.
				rs = nil
				send = nil
			}
		}
	}()
	for i := 0; i < 4; i++ {
		go loadTextureRoutine(pipe)
	}

	setupTextureList(manager.renderQueue)
}

// This routine waits for a filename and a data object, then loads the texture
// in that file into that object.  This is so that only one texture is being
// loaded at a time, it prevents us from hammering the filesystem and also
// makes sure we aren't using up a ton of memory all at once.
func loadTextureRoutine(pipe chan loadRequest) {
	for req := range pipe {
		handleLoadRequest(req)
	}
}

func BlockUntilLoaded(ctx context.Context, paths ...string) error {
	if manager == nil {
		panic("need to call texture.Init before texture.BlockUntilLoaded")
	}

	return manager.BlockUntilLoaded(ctx, paths...)
}

func BlockUntilIdle(ctx context.Context) error {
	if manager == nil {
		panic("need to call texture.Init before texture.BlockUntilIdle")
	}

	return manager.BlockUntilIdle(ctx)
}

// Returns a slice of the texture paths that are currently loading.
func GetInFlightRequests() []string {
	if manager == nil {
		panic("need to call texture.Init before texture.GetInFlightRequests")
	}

	return manager.GetInFlightRequests()
}

const load_threshold = 1000 * 1000

func handleLoadRequest(req loadRequest) {
	logging.Trace("texture manager: handleLoadRequest", "path", req.path)
	f, _ := os.Open(req.path)
	im, _, err := image.Decode(f)
	f.Close()
	if err != nil {
		manager.signalLoad(req.path, false)
		return
	}
	gray := true
	dx := im.Bounds().Dx()
	dy := im.Bounds().Dy()
	for i := 0; i < dx; i++ {
		for j := 0; j < dy; j++ {
			r, g, b, _ := im.At(i, j).RGBA()
			if r != g || g != b {
				gray = false
				break
			}
		}
		if !gray {
			break
		}
	}
	// Use an inverted image helper thing b/c OpenGL reads rows bottom-up but
	// golang stores rows top-down.
	var canvas *imgmanip.InvertedCanvas
	var pix []byte
	if gray {
		pix = memory.GetBlock(2 * req.data.dx * req.data.dy)
		ga := imgmanip.NewGrayAlpha(im.Bounds())
		ga.Pix = pix
		canvas = imgmanip.NewInvertedCanvas(ga)
	} else {
		pix = memory.GetBlock(4 * dx * dy)
		rgbaImage := &image.RGBA{
			Pix:    pix,
			Stride: 4 * dx,
			Rect:   im.Bounds(),
		}
		canvas = imgmanip.NewInvertedCanvas(rgbaImage)
	}
	draw.Draw(canvas, im.Bounds(), im, image.Point{}, draw.Src)
	manager.load.mutex.Lock()
	manager.load.count += len(pix)
	manual_unlock := false
	// This prevents us from trying to send too much to opengl in a single
	// frame.  If we go over the threshold then we hold the lock until we're
	// done sending data to opengl, then other requests will be free to
	// queue up and they will run on the next frame.
	if manager.load.count < load_threshold {
		manager.load.mutex.Unlock()
	} else {
		manual_unlock = true
	}
	manager.renderQueue.Queue(func(render.RenderQueueState) {
		{
			gl.Enable(gl.TEXTURE_2D)
			req.data.texture = gl.GenTexture()
			req.data.texture.Bind(gl.TEXTURE_2D)
			defer req.data.texture.Unbind(gl.TEXTURE_2D)
			gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
			gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
			gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
			gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
			gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		}
		if gray {
			glu.Build2DMipmaps(gl.TEXTURE_2D, gl.LUMINANCE_ALPHA, req.data.dx, req.data.dy, gl.LUMINANCE_ALPHA, gl.UNSIGNED_BYTE, pix)
		} else {
			glu.Build2DMipmaps(gl.TEXTURE_2D, gl.RGBA, req.data.dx, req.data.dy, gl.RGBA, gl.UNSIGNED_BYTE, pix)
		}
		memory.FreeBlock(pix)
		if manual_unlock {
			manager.load.count = 0
			manager.load.mutex.Unlock()
		}

		manager.signalLoad(req.path, true)
	})
}

func (m *Manager) LoadFromPath(path string) (*Data, error) {
	m.mutex.RLock()
	var data *Data
	var ok bool
	if data, ok = m.registry[path]; ok {
		m.mutex.RUnlock()
		m.onAccess(data)
		return data, nil
	}
	m.mutex.RUnlock()
	m.mutex.Lock()
	if data, ok = m.deleted[path]; ok {
		delete(m.deleted, path)
	} else {
		data = &Data{}
	}
	data.accessed = m.generation
	m.registry[path] = data
	m.inFlight[path] = true
	m.mutex.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("couldn't open path %q: %w", path, err)
	}
	config, _, err := image.DecodeConfig(f)
	f.Close()
	data.dx = config.Width
	data.dy = config.Height

	logging.Trace("texture manager: sending load request", "path", path)
	m.load.requests <- loadRequest{path, data}
	return data, nil
}

func (m *Manager) resetPath(o *Object, newpath base.Path) error {
	newData, err := m.LoadFromPath(string(newpath))
	if err != nil {
		return fmt.Errorf("couldn't m.LoadFromPath(%q): %w", newpath, err)
	}

	m.mutex.Lock()
	oldData := o.data
	o.data = newData
	m.mutex.Unlock()

	if oldData != nil {
		m.renderQueue.Queue(func(render.RenderQueueState) {
			oldData.texture.Delete()
		})
	}

	return nil
}

func (m *Manager) onAccess(d *Data) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	d.accessed = m.generation
}

func (m *Manager) getPath(o *Object) string {
	manager.mutex.RLock()
	defer manager.mutex.RUnlock()

	return string(o.Path)
}

func (m *Manager) BlockUntilLoaded(ctx context.Context, paths ...string) error {
	logging.Trace("block until loaded called", "paths", paths)
	pathset := make(map[string]bool)
	for _, path := range paths {
		pathset[path] = true
	}

	waitChannels := []chan bool{}

	func() {
		m.mutex.Lock()
		defer m.mutex.Unlock()

		for path := range pathset {
			if waitChan, found := m.loadWaiters[path]; found {
				waitChannels = append(waitChannels, waitChan)
				continue
			}

			_, found := m.registry[path]
			if !found {
				// Need a new channel to wait on
				newchan := make(chan bool, 1)
				m.loadWaiters[path] = newchan
				waitChannels = append(waitChannels, newchan)
				continue
			}

			// There's an entry in the registry but the texture might not be loaded
			// yet. If so, the path will be in 'inFlight'.
			if m.inFlight[path] {
				// Need a new channel to wait on
				newchan := make(chan bool, 1)
				m.loadWaiters[path] = newchan
				waitChannels = append(waitChannels, newchan)
				continue
			}

			// ... must be loaded; don't wait for it.
		}
	}()

	loadOk := true
	for _, c := range waitChannels {
		select {
		case loadResult := <-c:
			loadOk = loadOk && loadResult
		case <-ctx.Done():
			return fmt.Errorf("deadline exceeded")
		}
	}

	logging.Trace("done waiting", "times-waited", len(waitChannels))

	if !loadOk {
		return fmt.Errorf("texture load failure")
	}

	return nil
}

func (m *Manager) BlockUntilIdle(ctx context.Context) error {
	for inFlight := m.GetInFlightRequests(); len(inFlight) > 0; inFlight = m.GetInFlightRequests() {
		err := m.BlockUntilLoaded(ctx, inFlight...)
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *Manager) GetInFlightRequests() []string {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	return slices.Collect(maps.Keys(m.inFlight))
}

func (m *Manager) signalLoad(path string, success bool) {
	logging.Trace("signalling load", "path", path, "success", success)
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.inFlight, path)

	waitChan, found := m.loadWaiters[path]
	if !found {
		return
	}

	waitChan <- success
	close(waitChan)
	delete(m.loadWaiters, path)
}

// TODO(tmckee): this is horrible; not as horrible as exposing the
// module-global directly but still, pretty bad.
func GetRenderQueue() render.RenderQueueInterface {
	if manager == nil {
		panic("need to call texture.Init before texture.GetRenderQueue")
	}

	return manager.renderQueue
}
