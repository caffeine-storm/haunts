package base

import (
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"io/fs"
	"log"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"code.google.com/p/freetype-go/freetype/truetype"
	"github.com/MobRulesGames/haunts/globals"
	"github.com/MobRulesGames/opengl/gl"
	"github.com/runningwild/glop/glog"
	"github.com/runningwild/glop/gui"
	"github.com/runningwild/glop/render"
)

var datadir string

// Until we migrate lots of old log.Logger calls, we'll keep a log.Logger
// around.
// TODO(tmckee): delegate logging from 'logger' to 'slogger' so that all logs
// are structured/leveled conveniently.
var logger *log.Logger
var glogger glog.Logger

var log_reader io.Reader
var log_out *os.File
var log_console *bytes.Buffer

func SetDatadir(_datadir string) io.Reader {
	if datadir == _datadir {
		return log_console
	}

	if datadir != "" {
		panic(fmt.Errorf("double-setting datadir! was %q, new %q", datadir, _datadir))
	}

	datadir = _datadir
	return SetupLogger(datadir)
}
func GetDataDir() string {
	return datadir
}

func SetupLogger(dir string) *bytes.Buffer {
	// If an error happens when making this directory it might already exist,
	// all that really matters is making the log file in the directory.
	os.Mkdir(filepath.Join(dir, "logs"), 0777)
	logger = nil
	var err error
	name := "haunts.log"
	log_out, err = os.Create(filepath.Join(dir, "logs", name))
	if err != nil {
		fmt.Printf("Unable to open log file: %v\nLogging to stdout...\n", err.Error())
		log_out = os.Stdout
	}
	log_console = &bytes.Buffer{}
	log_writer := io.MultiWriter(log_console, log_out)
	logger = log.New(log_writer, "> ", log.Ltime|log.Lshortfile)

	glogger = glog.New(&glog.Opts{
		Output: log_writer,
	})

	base_logger = baseLogger{
		logger,
		glogger,
	}

	return log_console
}

type sloggy = glog.Slogger
type baseLogger struct {
	*log.Logger
	sloggy
}

var base_logger baseLogger

// Equivalent to glog.ErrorLogger.Error
func (*baseLogger) Error(msg string, args ...interface{}) {
	glogger.Error(msg, args...)
}

// Equivalent to glog.WarningLogger.Warn
func (*baseLogger) Warn(msg string, args ...interface{}) {
	glogger.Warn(msg, args...)
}

// Equivalent to glog.InfoLogger.Info
func (*baseLogger) Info(msg string, args ...interface{}) {
	glogger.Info(msg, args...)
}

// TODO: This probably isn't the best way to do things - different go-routines
// can call these and screw up prefixes for each other.
func Log() *baseLogger {
	logger.SetPrefix("LOG  > ")
	return &base_logger
}

func Warn() *baseLogger {
	logger.SetPrefix("WARN > ")
	return &base_logger
}

func Error() *baseLogger {
	logger.SetPrefix("ERROR> ")
	return &base_logger
}

func CloseLog() {
	log_out.WriteString("END OF LOG\n\n\n\n")
	log_out.Close()
}

var drawing_context gui.UpdateableDrawingContext
var dictionary_mutex sync.Mutex

func InitDictionaries(ctx gui.UpdateableDrawingContext) {
	drawing_context = ctx
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

func loadDictionaryFromFile(input io.Reader, renderQueue render.RenderQueueInterface, logger glog.Logger) (*gui.Dictionary, error) {
	d, err := gui.LoadDictionary(input, renderQueue, logger)
	if err != nil {
		return nil, fmt.Errorf("gui.LoadDictionary failed: %w", err)
	}

	return d, nil
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

// TODO(tmckee): this kinda breaks the abstraction for RenderQueue. We need it
// so that jobs running on the render thread already can call subroutines that
// call Queue(). We ought to find a better way to do this.
type immediateQueue struct {}
var _ render.RenderQueueInterface = (*immediateQueue)(nil)

func (q *immediateQueue) Queue(f render.RenderJob) {
	queue_state := globals.RenderQueueState()
	f(queue_state)
}
func (q *immediateQueue) StartProcessing() {}
func (q *immediateQueue) Purge() {}
func (q *immediateQueue) IsPurging() bool {
	return true
}

func GetDictionary(size int) *gui.Dictionary {
	dictionary_mutex.Lock()
	defer dictionary_mutex.Unlock()
	if drawing_context == nil {
		panic("need to call base.InitDictionaries first")
	}

	return getDictionaryByProperties("standard", size)
}

func fontIdFromProperties(fontName string, size int) string {
	return fmt.Sprintf("%s_%d", fontName, size)
}

func fontCachePath(fontName string, size int) string {
	return fmt.Sprintf("dict_%s.gob", fontIdFromProperties(fontName, size))
}

func getDictionaryByProperties(fontName string, size int) *gui.Dictionary {
	var ret *gui.Dictionary
	fontId := fontIdFromProperties(fontName, size)
	func () {
		// TODO(tmckee): catching a panic is not as nice as supporting lookup-miss
		// in the API.
		defer func() {
			if e := recover(); e != nil {
				switch e.(type) {
				case gui.MissingFontError:
					// TODO(tmckee): BARF!! THIS IS UGGGLGY
					ret = loadDictionaryByProperties(fontName, size)
					drawing_context.SetDictionary(fontId, ret)
					bank := globals.RenderQueueState().Shaders()
					drawing_context.SetShaders("glop.font", bank)
				default:
					panic(e)
				}
			}
		}()
		ret = drawing_context.GetDictionary(fontId)
	}()

	return ret
}

func loadDictionaryByProperties(fontName string, size int) *gui.Dictionary {
	logger := glog.Relevel(glogger, slog.LevelDebug)
	// First, check our disk cache for a grid-of-glyphs.
	cachePath := fontCachePath(fontName, size)

	filename := filepath.Join(datadir, "fonts", cachePath)
	f, err := os.Open(filename)
	if err == nil {
		defer f.Close()
		Log().Info("font-cache-hit", "fontName", fontName, "size", size, "err", err)
		d, err := loadDictionaryFromFile(f, &immediateQueue{}, glog.WarningLogger())
		if err != nil {
			panic(fmt.Errorf("couldn't loadDictionaryFromFile for %q @%d: %w", fontName, size, err))
		}

		return d
	}

	// Make sure this is a cache miss (i.e. missing file) instead of something
	// more serious.
	if !errors.Is(err, fs.ErrNotExist) {
		panic(fmt.Errorf("couldn't open %q: %w", filename, err))
	}

	// We don't have an appropriate grid-of-glyphs on disk; make one!
	Log().Warn("font-cache-miss", "fontName", fontName, "size", size, "err", err)
	font, err := loadFont()
	if err != nil {
		panic(fmt.Errorf("unable to load font: size %d: err: %w", size, err))
	}

	d := gui.MakeDictionary(font, size, &immediateQueue{}, logger)
	err = saveDictionaryToFile(d, size)
	if err != nil {
		Log().Error("Unable to save dictionary", "size", size, "err", err)
	}
	return d
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
		Log().Error("filepath.Rel(base, path) failed", "base", base, "path", path, "err", err)
		return
	}
	parts := strings.Split(rel, string(filepath.Separator))
	running := filepath.Join(base, parts[0])
	parts = parts[1:]
	for _, part := range parts {
		f, err := os.Open(running)
		if err != nil {
			Log().Error("os.Open(path) failed", "path", running, "err", err)
			return
		}
		names, err := f.Readdirnames(10000)
		f.Close()
		if err != nil {
			Log().Error("f.Readdirnames(10000) failed", "path", running, "err", err)
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
				Log().Error("os.Stat(final) failed", "final", final, "err", err)
				return
			}
			Log().Error("bad casing", "given", running, "should-end-with", part)
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
