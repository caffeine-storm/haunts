package base

import (
	"os"
	"path/filepath"

	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/render"
)

type Shader struct {
	Defname string
	*ShaderDef
}

type ShaderDef struct {
	// Name of this texture as it appears in the editor, should be unique among
	// all WallTextures
	Name string

	// Paths to the vertex and fragment shaders
	Vertex_path   string
	Fragment_path string
}

// Mappings from vertex shader name, fragment shader name, and shader program
// name to their respective opengl objects.
var vertex_shaders map[string]gl.Shader
var fragment_shaders map[string]gl.Shader
var shader_progs map[string]gl.Program

var warned_names map[string]bool

func EnableShader(name string) {
	prog_obj, ok := shader_progs[name]
	if ok {
		gl.Program(gl.GLuint(prog_obj)).Use()
	} else {
		gl.Program(0).Use()
		if name != "" && !warned_names[name] {
			Log().Warn("Tried to use unknown shader", "name", name)
			warned_names[name] = true
		}
	}
}

func SetUniformI(shader, variable string, n int) {
	prog, ok := shader_progs[shader]
	if !ok {
		if !warned_names[shader] {
			Log().Warn("Tried to set a uniform in an unknown shader", "shader", shader)
			warned_names[shader] = true
		}
		return
	}
	loc := gl.Program(prog).GetUniformLocation(variable)
	gl.UniformLocation(loc).Uniform1i(n)
}

func SetUniformF(shader, variable string, f float32) {
	prog, ok := shader_progs[shader]
	if !ok {
		if !warned_names[shader] {
			Log().Warn("Tried to set a uniform in an unknown shader", "shader", shader)
			warned_names[shader] = true
		}
		return
	}
	loc := prog.GetUniformLocation(variable)
	loc.Uniform1f(f)
}

func InitShaders(queue render.RenderQueueInterface) {
	queue.Queue(func(render.RenderQueueState) {
		vertex_shaders = make(map[string]gl.Shader)
		fragment_shaders = make(map[string]gl.Shader)
		shader_progs = make(map[string]gl.Program)
		warned_names = make(map[string]bool)
		RemoveRegistry("shaders")
		RegisterRegistry("shaders", make(map[string]*ShaderDef))
		RegisterAllObjectsInDir("shaders", filepath.Join(GetDataDir(), "shaders"), ".json", "json")
		names := GetAllNamesInRegistry("shaders")
		for _, name := range names {
			// Load the shader files
			shader := Shader{Defname: name}
			GetObject("shaders", &shader)
			vdata, err := os.ReadFile(filepath.Join(GetDataDir(), shader.Vertex_path))
			if err != nil {
				Log().Error("Unable to load vertex shader", "path", shader.Vertex_path, "err", err)
				continue
			}
			fdata, err := os.ReadFile(filepath.Join(GetDataDir(), shader.Fragment_path))
			if err != nil {
				Log().Error("Unable to load fragment shader", "path", shader.Fragment_path, "err", err)
				continue
			}

			// TODO(tmckee): we should defer to glop's shader loading
			// Create the vertex shader
			glVertexShader, ok := vertex_shaders[shader.Vertex_path]
			if !ok {
				glVertexShader = gl.CreateShader(gl.VERTEX_SHADER)

				glVertexShader.Source(string(vdata))
				glVertexShader.Compile()
				status := glVertexShader.Get(gl.COMPILE_STATUS)
				if status == 0 {
					Log().Error("Failed to compile vertex shader", "path", shader.Vertex_path, "compile-status", status)
					continue
				}
			}

			// Create the fragment shader
			glFragmentShader, ok := fragment_shaders[shader.Fragment_path]
			if !ok {
				glFragmentShader = gl.CreateShader(gl.FRAGMENT_SHADER)
				glFragmentShader.Source(string(fdata))
				glFragmentShader.Compile()
				status := glFragmentShader.Get(gl.COMPILE_STATUS)
				if status == 0 {
					Log().Error("Failed to compile fragment shader", "path", shader.Fragment_path, "compile-status", status)
					continue
				}
			}

			// shader successfully compiled - now link
			glProgram := gl.CreateProgram()
			glProgram.AttachShader(glVertexShader)
			glProgram.AttachShader(glFragmentShader)
			glProgram.Link()
			status := glProgram.Get(gl.LINK_STATUS)
			if status == 0 {
				Log().Error("Failed to link shader", "shader-name", shader.Name, "compile-status", status)
				continue
			}

			vertex_shaders[shader.Vertex_path] = glVertexShader
			fragment_shaders[shader.Fragment_path] = glFragmentShader
			shader_progs[shader.Name] = glProgram
		}
	})
	queue.Purge()
}
