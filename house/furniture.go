package house

import (
	"github.com/MobRulesGames/haunts/base"
	"github.com/MobRulesGames/haunts/texture"
	"github.com/caffeine-storm/mathgl"
)

func MakeFurniture(name string) *Furniture {
	f := Furniture{Defname: name}
	base.GetObject("furniture", &f)
	return &f
}

func GetAllFurnitureNames() []string {
	return base.GetAllNamesInRegistry("furniture")
}

func LoadAllFurnitureInDir(dir string) {
	base.RemoveRegistry("furniture")
	base.RegisterRegistry("furniture", make(map[string]*FurnitureDef))
	base.RegisterAllObjectsInDir("furniture", dir, ".json", "json")
}

type Furniture struct {
	Defname string
	*FurnitureDef

	// Position of this object in board coordinates.
	X, Y BoardSpaceUnit

	// Index into furnitureDef.Texture_paths
	Rotation int

	Flip bool

	// If this is currently being dragged around it will be marked as temporary
	// so that it will be drawn differently
	temporary bool

	// Used to determine how this is drawn as it is being moved in the editor
	invalid bool

	// If someone is walking behind this, and it blocks los, then we'll want to
	// make it transparent.
	alpha         float64
	alpha_enabled bool
}

func (f *Furniture) SetAlpha(a float64) {
	f.alpha = a
	f.alpha_enabled = true
}

func (f *Furniture) Alpha() float64 {
	return f.alpha
}

func (f *Furniture) FloorPos() (BoardSpaceUnit, BoardSpaceUnit) {
	return f.X, f.Y
}

func (f *Furniture) FPos() (float64, float64) {
	return float64(f.X), float64(f.Y)
}

func (f *Furniture) RotateLeft() {
	f.Rotation = (f.Rotation + 1) % len(f.Orientations)
}

func (f *Furniture) RotateRight() {
	f.Rotation = (f.Rotation - 1 + len(f.Orientations)) % len(f.Orientations)
}

type furnitureOrientation struct {
	Dx, Dy  BoardSpaceUnit
	Texture texture.Object `registry:"autoload"`
}

// All instances of the same piece of furniture have this data in common
type FurnitureDef struct {
	// Name of the object - should be unique among all furniture
	Name string

	// All available orientations for this piece of furniture
	Orientations []furnitureOrientation

	// Whether or not this piece of furniture blocks line-of-sight.  If a piece
	// of furniture blocks los, then the entire piece blocks los, regardless of
	// orientation.
	Blocks_los bool
}

func (f *Furniture) Dims() (BoardSpaceUnit, BoardSpaceUnit) {
	orientation := f.Orientations[f.Rotation]
	if f.Flip {
		return orientation.Dy, orientation.Dx
	}
	return orientation.Dx, orientation.Dy
}

func (f *Furniture) Color() (r, g, b, a byte) {
	if f.temporary {
		if f.invalid {
			return 255, 127, 127, 200
		} else {
			return 127, 127, 255, 200
		}
	}
	return 255, 255, 255, 255
}

func (f *Furniture) Render(pos mathgl.Vec2, width float32) {
	if !f.Blocks_los || !f.alpha_enabled {
		f.alpha = 1
	}
	orientation := f.Orientations[f.Rotation]
	dy := width * float32(orientation.Texture.Data().Dy()) / float32(orientation.Texture.Data().Dx())
	// orientation.Texture.Data().Render(float64(pos.X), float64(pos.Y), float64(width), float64(dy))
	orientation.Texture.Data().RenderAdvanced(float64(pos.X), float64(pos.Y), float64(width), float64(dy), 0, !f.Flip)
}
