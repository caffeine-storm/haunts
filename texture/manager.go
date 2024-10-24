package texture

import (
	"image"
	"image/draw"
	_ "image/jpeg"
	_ "image/png"
	"os"
	"sync"
	"time"

	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/mathgl"
	"github.com/MobRulesGames/memory"
	"github.com/MobRulesGames/opengl/gl"
	"github.com/MobRulesGames/opengl/glu"
	"github.com/runningwild/glop/render"
)

type Object struct {
	Path base.Path

	// this path is the last one that was loaded, so that if we change Path then
	// we know to reload the texture.
	path base.Path
	data *Data
}

func (o *Object) Data() *Data {
	if o.data == nil || o.path != o.Path || o.data.texture == 0 {
		o.data = LoadFromPath(string(o.Path))
		o.path = o.Path
	}
	o.data.accessed = generation
	return o.data
}

type Data struct {
	dx, dy   int
	texture  gl.Texture
	accessed int
}

func (d *Data) Dx() int {
	return d.dx
}
func (d *Data) Dy() int {
	return d.dy
}

var textureList uint
var textureListSync sync.Once

func setupTextureList() {
	textureListSync.Do(func() {
		manager.renderQueue.Queue(func() {
			textureList = gl.GenLists(1)
			gl.NewList(textureList, gl.COMPILE)
			gl.Begin(gl.QUADS)
			gl.TexCoord2d(0, 0)
			gl.Vertex2i(0, 0)

			gl.TexCoord2d(0, -1)
			gl.Vertex2i(0, 1)

			gl.TexCoord2d(1, -1)
			gl.Vertex2i(1, 1)

			gl.TexCoord2d(1, 0)
			gl.Vertex2i(1, 0)
			gl.End()
			gl.EndList()
		})
	})
}

// Renders the texture on a quad at the texture's natural size.
func (d *Data) RenderNatural(x, y int) {
	d.Render(float64(x), float64(y), float64(d.dx), float64(d.dy))
}

func Render(x, y, dx, dy float64) {
	var run, op mathgl.Mat4
	run.Identity()
	op.Translation(float32(x), float32(y), 0)
	run.Multiply(&op)
	op.Scaling(float32(dx), float32(dy), 1)
	run.Multiply(&op)

	gl.PushMatrix()
	gl.Enable(gl.TEXTURE_2D)
	gl.MultMatrixf(&run[0])
	gl.CallList(textureList)
	gl.PopMatrix()
}

func (d *Data) Render(x, y, dx, dy float64) {
	if textureList != 0 {
		d.Bind()
		Render(x, y, dx, dy)
	}
}

func (d *Data) RenderAdvanced(x, y, dx, dy, rot float64, flip bool) {
	d.Bind()
	RenderAdvanced(x, y, dx, dy, rot, flip)
}

func RenderAdvanced(x, y, dx, dy, rot float64, flip bool) {
	if textureList != 0 {
		var run, op mathgl.Mat4
		run.Identity()
		op.Translation(float32(x), float32(y), 0)
		run.Multiply(&op)
		op.Translation(float32(dx/2), float32(dy/2), 0)
		run.Multiply(&op)
		op.RotationZ(float32(rot))
		run.Multiply(&op)
		if flip {
			op.Translation(float32(-dx/2), float32(-dy/2), 0)
			run.Multiply(&op)
			op.Scaling(float32(dx), float32(dy), 1)
			run.Multiply(&op)
		} else {
			op.Translation(float32(dx/2), float32(-dy/2), 0)
			run.Multiply(&op)
			op.Scaling(float32(-dx), float32(dy), 1)
			run.Multiply(&op)
		}
		gl.PushMatrix()
		gl.MultMatrixf(&run[0])
		gl.Enable(gl.TEXTURE_2D)
		gl.CallList(textureList)
		gl.PopMatrix()
	}
}

func (d *Data) Bind() {
	if d.texture == 0 {
		if error_texture == 0 {
			makeErrorTexture()
		}
		error_texture.Bind(gl.TEXTURE_2D)
	} else {
		d.texture.Bind(gl.TEXTURE_2D)
	}
}

// Instead of keeping track of access time, we just keep track of how many
// times the scavenger has seen something without it being accessed.
// generation is incremented every time the scavenger loop runs, and any
// time a texture is accessed it is updated with the current generation.
var generation int

// Launch this in its own go-routine if you want to occassionally
// delete textures that haven't been used in a while.
func (m *Manager) Scavenger() {
	var unused []string
	for {
		time.Sleep(time.Minute)
		unused = unused[0:0]
		m.mutex.RLock()
		for s, d := range m.registry {
			if generation-d.accessed >= 2 {
				unused = append(unused, s)
			}
		}
		m.mutex.RUnlock()

		m.mutex.Lock()
		generation++
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
		manager.renderQueue.Queue(func() {
			for _, d := range unused_data {
				d.texture.Delete()
				d.texture = 0
			}
		})
		m.mutex.Unlock()
	}
}

func LoadFromPath(path string) *Data {
	if manager == nil {
		panic("need to call texure.Init before texture.LoadFromPath")
	}

	return manager.LoadFromPath(path)
}

type loadRequest struct {
	path string
	data *Data
}

var load_requests chan loadRequest
var load_count int
var load_mutex sync.Mutex

const load_threshold = 1000 * 1000

type Manager struct {
	// Currently loaded/loading textures are in the registry
	registry map[string]*Data

	// If a texture has been deleted it is moved here so that if it gets
	// reloaded it will be loaded into the same texture object it was in
	// before.
	deleted map[string]*Data

	// Rendering queue/context that will be used for all gl operations.
	renderQueue render.RenderQueueInterface

	mutex sync.RWMutex
}

var (
	manager *Manager
)

func Init(renderQueue render.RenderQueueInterface) {
	manager = &Manager{
		registry:    make(map[string]*Data),
		deleted:     make(map[string]*Data),
		renderQueue: renderQueue,
	}

	go manager.Scavenger()

	load_requests = make(chan loadRequest, 10)
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
			case r := <-load_requests:
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

func handleLoadRequest(req loadRequest) {
	f, _ := os.Open(req.path)
	im, _, err := image.Decode(f)
	f.Close()
	if err != nil {
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
	var canvas draw.Image
	var pix []byte
	if gray {
		ga := NewGrayAlpha(im.Bounds())
		pix = ga.Pix
		canvas = ga
	} else {
		pix = memory.GetBlock(4 * req.data.dx * req.data.dy)
		canvas = &image.RGBA{pix, 4 * req.data.dx, im.Bounds()}
	}
	draw.Draw(canvas, im.Bounds(), im, image.Point{}, draw.Src)
	load_mutex.Lock()
	load_count += len(pix)
	manual_unlock := false
	// This prevents us from trying to send too much to opengl in a single
	// frame.  If we go over the threshold then we hold the lock until we're
	// done sending data to opengl, then other requests will be free to
	// queue up and they will run on the next frame.
	if load_count < load_threshold {
		load_mutex.Unlock()
	} else {
		manual_unlock = true
	}
	manager.renderQueue.Queue(func() {
		{
			gl.Enable(gl.TEXTURE_2D)
			req.data.texture = gl.GenTexture()
			req.data.texture.Bind(gl.TEXTURE_2D)
			gl.TexEnvf(gl.TEXTURE_ENV, gl.TEXTURE_ENV_MODE, gl.MODULATE)
			gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.LINEAR_MIPMAP_LINEAR)
			gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.LINEAR)
			gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_S, gl.REPEAT)
			gl.TexParameterf(gl.TEXTURE_2D, gl.TEXTURE_WRAP_T, gl.REPEAT)
		}
		if gray {
			glu.Build2DMipmaps(gl.TEXTURE_2D, gl.LUMINANCE_ALPHA, req.data.dx, req.data.dy, gl.LUMINANCE_ALPHA, pix)
		} else {
			glu.Build2DMipmaps(gl.TEXTURE_2D, gl.RGBA, req.data.dx, req.data.dy, gl.RGBA, pix)
		}
		memory.FreeBlock(pix)
		if manual_unlock {
			load_count = 0
			load_mutex.Unlock()
		}
	})
}

func (m *Manager) LoadFromPath(path string) *Data {
	setupTextureList()
	m.mutex.RLock()
	var data *Data
	var ok bool
	if data, ok = m.registry[path]; ok {
		m.mutex.RUnlock()
		m.mutex.Lock()
		data.accessed = generation
		m.mutex.Unlock()
		return data
	}
	m.mutex.RUnlock()
	m.mutex.Lock()
	if data, ok = m.deleted[path]; ok {
		delete(m.deleted, path)
	} else {
		data = &Data{}
	}
	data.accessed = generation
	m.registry[path] = data
	m.mutex.Unlock()

	f, err := os.Open(path)
	if err != nil {
		return data
	}
	config, _, err := image.DecodeConfig(f)
	f.Close()
	data.dx = config.Width
	data.dy = config.Height

	load_requests <- loadRequest{path, data}
	return data
}

// TODO(tmckee): this is horrible; not as horrible as exposing the
// module-global directly but still, pretty bad.
func GetRenderQueue() render.RenderQueueInterface {
	if manager == nil {
		panic("need to call texture.Init before texture.GetRenderQueue")
	}

	return manager.renderQueue
}
