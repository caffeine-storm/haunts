package base

import (
	"os"
	"path/filepath"

	"github.com/go-gl-legacy/gl"
	"github.com/runningwild/glop/render"
)

type Shader struct {
	Defname string
	*shaderDef
}

type shaderDef struct {
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
			Warn().Printf("Tried to use unknown shader '%s'", name)
			warned_names[name] = true
		}
	}
}

func SetUniformI(shader, variable string, n int) {
	prog, ok := shader_progs[shader]
	if !ok {
		if !warned_names[shader] {
			Warn().Printf("Tried to set a uniform in an unknown shader '%s'", shader)
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
			Warn().Printf("Tried to set a uniform in an unknown shader '%s'", shader)
			warned_names[shader] = true
		}
		return
	}
	loc := prog.GetUniformLocation(variable)
	loc.Uniform1f(f)
}

func InitShaders() {
	render.Queue(func() {
		vertex_shaders = make(map[string]gl.Shader)
		fragment_shaders = make(map[string]gl.Shader)
		shader_progs = make(map[string]gl.Program)
		warned_names = make(map[string]bool)
		RemoveRegistry("shaders")
		RegisterRegistry("shaders", make(map[string]*shaderDef))
		RegisterAllObjectsInDir("shaders", filepath.Join(GetDataDir(), "shaders"), ".json", "json")
		names := GetAllNamesInRegistry("shaders")
		for _, name := range names {
			// Load the shader files
			shader := Shader{Defname: name}
			GetObject("shaders", &shader)
			vdata, err := os.ReadFile(filepath.Join(GetDataDir(), shader.Vertex_path))
			if err != nil {
				Error().Printf("Unable to load vertex shader '%s': %v", shader.Vertex_path, err)
				continue
			}
			fdata, err := os.ReadFile(filepath.Join(GetDataDir(), shader.Fragment_path))
			if err != nil {
				Error().Printf("Unable to load fragment shader '%s': %v", shader.Fragment_path, err)
				continue
			}

			// Create the vertex shader
			glVertexShader, ok := vertex_shaders[shader.Vertex_path]
			if !ok {
				glVertexShader = gl.CreateShader(gl.VERTEX_SHADER)

				glVertexShader.Source(string(vdata))
				glVertexShader.Compile()
				param := glVertexShader.Get(gl.COMPILE_STATUS)
				if param == 0 {
					Error().Printf("Failed to compile vertex shader '%s': %v", shader.Vertex_path, param)
					continue
				}
			}

			// Create the fragment shader
			glFragmentShader, ok := fragment_shaders[shader.Fragment_path]
			if !ok {
				glFragmentShader = gl.CreateShader(gl.FRAGMENT_SHADER)
				glFragmentShader.Source(string(fdata))
				glFragmentShader.Compile()
				param := glFragmentShader.Get(gl.COMPILE_STATUS)
				if param == 0 {
					Error().Printf("Failed to compile fragment shader '%s': %v", shader.Fragment_path, param)
					continue
				}
			}

			// shader successfully compiled - now link
			glProgram := gl.CreateProgram()
			glProgram.AttachShader(glVertexShader)
			glProgram.AttachShader(glFragmentShader)
			glProgram.Link()
			param := glProgram.Get(gl.LINK_STATUS)
			if param == 0 {
				Error().Printf("Failed to link shader '%s': %v", shader.Name, param)
				continue
			}

			vertex_shaders[shader.Vertex_path] = glVertexShader
			fragment_shaders[shader.Fragment_path] = glFragmentShader
			shader_progs[shader.Name] = glProgram
		}
	})
}
