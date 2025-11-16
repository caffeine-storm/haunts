package base

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/caffeine-storm/gl"
	"github.com/caffeine-storm/glop/render"
)

type Shader struct {
	Defname string
	*ShaderDef
}

type ShaderDef struct {
	// Name of this shader, should be unique amongst all shaders
	Name string

	// Paths to the vertex and fragment shaders
	Vertex_path   string
	Fragment_path string
}

// Mappings from vertex shader name, fragment shader name, and shader program
// name to their respective opengl objects.
var (
	vertex_shaders   map[string]gl.Shader
	fragment_shaders map[string]gl.Shader
	shader_progs     map[string]gl.Program
)

func EnableShader(name string) {
	render.MustBeOnRenderThread()
	if name == "" {
		gl.Program(0).Use()
		return
	}

	prog_obj, ok := shader_progs[name]
	if !ok {
		panic(fmt.Errorf("unknown shader %q", name))
	}

	prog_obj.Use()
}

func SetUniformI(shader, variable string, n int) {
	prog, ok := shader_progs[shader]
	if !ok {
		panic(fmt.Errorf("can't SetUniformI on an unknown shader %q", shader))
	}
	loc := gl.Program(prog).GetUniformLocation(variable)
	gl.UniformLocation(loc).Uniform1i(n)
}

func SetUniformF(shader, variable string, f float32) {
	prog, ok := shader_progs[shader]
	if !ok {
		panic(fmt.Errorf("can't SetUniformF on an unknown shader %q", shader))
	}
	loc := prog.GetUniformLocation(variable)
	loc.Uniform1f(f)
}

func InitShaders(queue render.RenderQueueInterface) {
	queue.Queue(func(render.RenderQueueState) {
		vertex_shaders = make(map[string]gl.Shader)
		fragment_shaders = make(map[string]gl.Shader)
		shader_progs = make(map[string]gl.Program)
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
				DeprecatedLog().Error("Unable to load vertex shader", "path", shader.Vertex_path, "err", err)
				continue
			}
			fdata, err := os.ReadFile(filepath.Join(GetDataDir(), shader.Fragment_path))
			if err != nil {
				DeprecatedLog().Error("Unable to load fragment shader", "path", shader.Fragment_path, "err", err)
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
					DeprecatedLog().Error("Failed to compile vertex shader", "path", shader.Vertex_path, "compile-status", status)
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
					DeprecatedLog().Error("Failed to compile fragment shader", "path", shader.Fragment_path, "compile-status", status)
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
				DeprecatedLog().Error("Failed to link shader", "shader-name", shader.Name, "compile-status", status)
				continue
			}

			vertex_shaders[shader.Vertex_path] = glVertexShader
			fragment_shaders[shader.Fragment_path] = glFragmentShader
			shader_progs[shader.Name] = glProgram
		}
	})
	queue.Purge()
}
