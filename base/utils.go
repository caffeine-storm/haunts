package base

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"image/color"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"code.google.com/p/freetype-go/freetype/truetype"
	"github.com/MobRulesGames/opengl/gl"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
)

var datadir string
var logger *log.Logger
var log_reader io.Reader
var log_out *os.File
var log_console *bytes.Buffer

func SetDatadir(_datadir string) {
	datadir = _datadir
	setupLogger()
}
func GetDataDir() string {
	return datadir
}

func setupLogger() {
	// If an error happens when making this directory it might already exist,
	// all that really matters is making the log file in the directory.
	os.Mkdir(filepath.Join(datadir, "logs"), 0777)
	logger = nil
	var err error
	name := "haunts.log"
	log_out, err = os.Create(filepath.Join(datadir, "logs", name))
	if err != nil {
		fmt.Printf("Unable to open log file: %v\nLogging to stdout...\n", err.Error())
		log_out = os.Stdout
	}
	log_console = bytes.NewBuffer(nil)
	log_writer := io.MultiWriter(log_console, log_out)
	logger = log.New(log_writer, "> ", log.Ltime|log.Lshortfile)
}

// TODO: This probably isn't the best way to do things - different go-routines
// can call these and screw up prefixes for each other.
func Log() *log.Logger {
	logger.SetPrefix("LOG  > ")
	return logger
}

func Warn() *log.Logger {
	logger.SetPrefix("WARN > ")
	return logger
}

func Error() *log.Logger {
	logger.SetPrefix("ERROR> ")
	return logger
}

func CloseLog() {
	log_out.WriteString("END OF LOG\n\n\n\n")
	log_out.Close()
}

var font_dict map[int]*gui.Dictionary
var render_queue render.RenderQueueInterface
var dictionary_mutex sync.Mutex

func InitDictionaries(renderQueue render.RenderQueueInterface) {
	if font_dict != nil {
		panic("must not call InitDictionaries multiple times!")
	}
	font_dict = make(map[int]*gui.Dictionary)
	render_queue = renderQueue
}

func loadFont() (*truetype.Font, error) {
	f, err := os.Open(filepath.Join(datadir, "fonts", "tomnr.ttf"))
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return nil, err
	}
	font, err := truetype.Parse(data)
	if err != nil {
		return nil, err
	}
	return font, nil
}

func loadDictionaryFromFile(size int, renderQueue render.RenderQueueInterface) (*gui.Dictionary, error) {
	name := fmt.Sprintf("dict_%d.gob", size)
	f, err := os.Open(filepath.Join(datadir, "fonts", name))
	var d *gui.Dictionary
	if err == nil {
		d, err = gui.LoadDictionary(f, renderQueue, logger)
		f.Close()
	}
	return d, err
}

func saveDictionaryToFile(d *gui.Dictionary, size int) error {
	name := fmt.Sprintf("dict_%d.gob", size)
	f, err := os.Create(filepath.Join(datadir, "fonts", name))
	if err != nil {
		return err
	}
	defer f.Close()
	return d.Store(f)
}

func GetDictionary(size int) *gui.Dictionary {
	dictionary_mutex.Lock()
	defer dictionary_mutex.Unlock()
	if font_dict == nil {
		panic("need to call base.InitDictionaries first")
	}
	if _, ok := font_dict[size]; !ok {
		d, err := loadDictionaryFromFile(size, render_queue)
		if err == nil {
			font_dict[size] = d
		} else {
			Warn().Printf("Unable to load dictionary (%d) from file: %v", size, err)
			font, err := loadFont()
			if err != nil {
				Error().Fatalf("Unable to load font: %v", err)
			}
			d = gui.MakeDictionary(font, size)
			err = saveDictionaryToFile(d, size)
			if err != nil {
				Warn().Printf("Unable to save dictionary (%d) to file: %v", size, err)
			}
			font_dict[size] = d
		}
	}

	return font_dict[size]
}

// A Path is a string that is intended to store a path.  When it is encoded
// with gob or json it will convert itself to a relative path relative to
// datadir.  When it is decoded from gob or json it will convert itself to an
// absolute path based on datadir.
type Path string

func (p Path) String() string {
	return string(p)
}
func (p Path) GobEncode() ([]byte, error) {
	return []byte(TryRelative(datadir, string(p))), nil
}
func (p *Path) GobDecode(data []byte) error {
	*p = Path(filepath.Join(datadir, string(data)))
	return nil
}
func (p Path) MarshalJSON() ([]byte, error) {
	val := filepath.ToSlash(TryRelative(datadir, string(p)))
	return []byte("\"" + val + "\""), nil
}
func (p *Path) UnmarshalJSON(data []byte) error {
	rel := filepath.FromSlash(string(data[1 : len(data)-1]))
	*p = Path(filepath.Join(datadir, rel))
	return nil
}

func CheckPathCasing(path string) {
	if !IsDevel() {
		return
	}
	base := GetDataDir()
	rel, err := filepath.Rel(base, path)
	if err != nil {
		Error().Printf("Unable to CheckPathCasing(%s, %s): %v", base, path, err)
		return
	}
	parts := strings.Split(rel, string(filepath.Separator))
	running := filepath.Join(base, parts[0])
	parts = parts[1:]
	for _, part := range parts {
		f, err := os.Open(running)
		if err != nil {
			Error().Printf("Unable to open '%s': %v", running, err)
			return
		}
		names, err := f.Readdirnames(10000)
		f.Close()
		if err != nil {
			Error().Printf("Unable to Readdirnames on '%s': %v", running, err)
			return
		}
		found := false
		for _, name := range names {
			if name == part {
				found = true
				break
			}
		}
		if !found {
			final := filepath.Join(running, part)
			_, err := os.Stat(final)
			if err != nil {
				Error().Printf("Unable to stat '%s': %v", final, err)
				return
			}
			Error().Printf("Improper casing: '%s' should end with '%s'", running, part)
			return
		}
		running = filepath.Join(running, part)
	}
}

// Opens the file named by path, reads it all, decodes it as json into target,
// then closes the file.  Returns the first error found while doing this or nil.
func LoadJson(path string, target interface{}) error {
	CheckPathCasing(path)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	data, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	err = json.Unmarshal(data, target)
	return err
}

func SaveJson(path string, source interface{}) error {
	data, err := json.Marshal(source)
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(data)
	return err
}

func ToGobToBase64(src interface{}) (string, error) {
	buf := bytes.NewBuffer(nil)
	enc := gob.NewEncoder(buf)
	err := enc.Encode(src)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func FromBase64FromGob(dst interface{}, str string) error {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return err
	}
	dec := gob.NewDecoder(bytes.NewBuffer(data))
	return dec.Decode(dst)
}

// Opens the file named by path, reads it all, decodes it as gob into target,
// then closes the file.  Returns the first error found while doing this or nil.
func LoadGob(path string, target interface{}) error {
	CheckPathCasing(path)
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	dec := gob.NewDecoder(f)
	err = dec.Decode(target)
	return err
}

func SaveGob(path string, source interface{}) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := gob.NewEncoder(f)
	err = enc.Encode(source)
	return err
}

// Returns a path rel such that filepath.Join(a, rel) and b refer to the same
// file.  a and b must both be relative paths or both be absolute paths.  If
// they are not then b will be returned in either case.
func TryRelative(base, target string) string {
	rel, err := filepath.Rel(base, target)
	if err == nil {
		return rel
	}
	return target
}

func GetStoreVal(key string) string {
	var store map[string]string
	LoadJson(filepath.Join(datadir, "store"), &store)
	if store == nil {
		store = make(map[string]string)
	}
	val := store[key]
	return val
}

func SetStoreVal(key, val string) {
	var store map[string]string
	path := filepath.Join(datadir, "store")
	LoadJson(path, &store)
	if store == nil {
		store = make(map[string]string)
	}
	store[key] = val
	SaveJson(path, store)
}

type ColorStack struct {
	colors []color.NRGBA
}

func (cs *ColorStack) Push(r, g, b, a float64) {
	c := color.NRGBA{byte(255 * r), byte(255 * g), byte(255 * b), byte(255 * a)}
	cs.colors = append(cs.colors, c)
}
func (cs *ColorStack) Pop() {
	cs.colors = cs.colors[0 : len(cs.colors)-1]
}
func (cs *ColorStack) subApply(n int) (r, g, b, a float64) {
	if n < 0 {
		return 1, 1, 1, 0
	}
	dr, dg, db, da := cs.subApply(n - 1)
	a = float64(cs.colors[n].A) / 255
	r = float64(cs.colors[n].R)/255*a + dr*(1-a)
	g = float64(cs.colors[n].G)/255*a + dg*(1-a)
	b = float64(cs.colors[n].B)/255*a + db*(1-a)
	a = a + (1-a)*da
	return
}
func (cs *ColorStack) Apply() {
	gl.Color4d(cs.subApply(len(cs.colors) - 1))
}
func (cs *ColorStack) ApplyWithAlpha(alpha float64) {
	r, g, b, a := cs.subApply(len(cs.colors) - 1)
	gl.Color4d(r, g, b, a*alpha)
}
