package base

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/MobRulesGames/haunts/logging"
)

// Many things have the following format
//   type Foo struct {
//     Defname string
//     *fooDef
//     FooInst
//   }
// Such that a Foo is something for which there can be multiple instances (such
// as a hallway, or a couch), fooDef is the data that is constant between all
// such instances, and FooInst is the data that makes each instance unique
// (location, orientation, maybe textures, etc...)
//
// With things in this format it is convenient to have a registry structured
// like this:
//   foo_registry map[string]*fooDef
// so that a Foo can be made from a fooDef just by supplying the name of the
// fooDef. Given all of this the following functions are very common to all
// registries:
//
// GetAllFooNames() - Returns all keys in the foo_registry, in sorted order
//
// LoadAllFoosInDir(path string) - Finds every Foo that can be loaded in the
// specified directory and loads it into the registry.
//
// MakeFoo(name string) - Makes a Foo by finding the fooDef in the registry and
// embedding it in a Foo.
//
// Tags:
// The following tags can be used which will apply special processing to the
// objects when registered:
//
// `registry:"autoload"` - If an object is tagged with this and it has a
// method named Load() that takes zero inputs and zero outputs then its Load
// method will be called after all of its data has been loaded.

var (
	registry_registry map[string]reflect.Value
)

func init() {
	registry_registry = make(map[string]reflect.Value)
}

func RemoveRegistry(name string) {
	delete(registry_registry, name)
}

// Registers a registry which must be a map from string to
// pointer-to-something.
func RegisterRegistry(name string, registry interface{}) {
	if strings.Contains(name, " ") {
		logging.Error("Registry name cannot contain spaces", "name", name)
	}
	mr := reflect.ValueOf(registry)
	if mr.Kind() != reflect.Map {
		logging.Error("Registries must be map[string]*struct", "actualkind", mr.Kind())
	}
	if mr.Type().Key().Kind() != reflect.String {
		logging.Error("Registry must be a map that uses strings as keys", "actualtype", mr.Type().Key())
	}
	if mr.Type().Elem().Kind() != reflect.Pointer {
		logging.Error("Registry must be a map that uses pointers as values", "actualtype", mr.Type().Elem())
	}
	if field, ok := mr.Type().Elem().Elem().FieldByName("Name"); !ok || field.Type.Kind() != reflect.String {
		logging.Error("Registry must store values that have a Name field of type string")
	}
	if _, ok := registry_registry[name]; ok {
		logging.Error("Cannot register two registries with the same name", "name", name)
	}
	registry_registry[name] = mr
}

// Registers object in the named registry which must have already been
// registered through RegisterRegistry(). object must be a pointer of the type
// appropriate for the named registry.
func RegisterObject(registry_name string, object interface{}) {
	reg, ok := registry_registry[registry_name]
	if !ok {
		logging.Error("Tried to register an object into an unknown registry", "name", registry_name)
	}

	obj_val := reflect.ValueOf(object)
	if obj_val.Kind() != reflect.Pointer {
		logging.Error("Can only register objects as pointers", "actualkind", obj_val.Kind())
	}
	if obj_val.Elem().Type() != reg.Type().Elem().Elem() {
		logging.Error("Registry type mismatch", "objtype", obj_val.Elem(), "registry", registry_name, "requiredtype", reg.Type().Elem().Elem())
	}

	// At this point we know we have the right type, and since registries can
	// only exist that store values with a field called Name of type string we
	// don't need to check for validity, we can assume it.
	object_name := obj_val.Elem().FieldByName("Name").String()
	cur_val := reg.MapIndex(reflect.ValueOf(object_name))
	if cur_val.IsValid() {
		logging.Error("Registry entry name collision", "object_name", object_name, "registry_name", registry_name)
	}
	reg.SetMapIndex(reflect.ValueOf(object_name), obj_val)
}

// Loads an object using the specified registry. object should have a field
// called Defname of type string. This name will be used to find the def in the
// registry. The object should also embed a field of this type which the value
// in the registry will be assigned to.
func GetObject(registry_name string, object interface{}) {
	reg, ok := registry_registry[registry_name]
	if !ok {
		logging.Error("Load from an unknown registry", "registry_name", registry_name)
	}

	object_val := reflect.ValueOf(object)
	if object_val.Kind() != reflect.Pointer {
		logging.Error("Tried to load into a value that was not a pointer", "actualkind", object_val.Kind())
	}

	object_name := object_val.Elem().FieldByName("Defname")
	if !object_name.IsValid() || object_name.Kind() != reflect.String {
		logging.Error("Missing Defname field")
	}

	cur_val := reg.MapIndex(object_name)
	if !cur_val.IsValid() {
		logging.Error("No object with name", "object_name", object_name.String(), "registry_name", registry_name)
	}
	fieldName := cur_val.Elem().Type().Name()
	field := object_val.Elem().FieldByName(fieldName)
	if !field.IsValid() {
		logging.Error("Expected embedded field", "containertype", object_val.Elem().Type(), "missingtype", cur_val.Type())
	}
	if !field.CanSet() {
		panic(fmt.Errorf("can't set value through field named %q", fieldName))
	}
	field.Set(cur_val)
}

// Returns a sorted list of all names in the specified registry.
func GetAllNamesInRegistry(registry_name string) []string {
	reg, ok := registry_registry[registry_name]
	if !ok {
		logging.Error("Unknown registry", "registry_name", registry_name)
	}
	keys := reg.MapKeys()
	var names []string
	for _, key := range keys {
		names = append(names, key.String())
	}
	sort.Strings(names)
	return names
}

// Processes an object as it is normally processed when registered through
// RegisterAllObjectsInDir(). Does NOT register the object in any registry.
func LoadAndProcessObject(path, format string, target interface{}) error {
	logging.Info("LoadAndProcessObject", "path", path)
	var err error
	switch format {
	case "json":
		err = LoadJson(path, target)

	case "gob":
		err = LoadGob(path, target)

	default:
		panic(fmt.Errorf("Unknown format, %q", format))
	}
	if err != nil {
		return err
	}

	ProcessObject(reflect.ValueOf(target), "")
	return nil
}

// Recursively decends through a value's type hierarchy and applies processing
// according to any tags that have been set on those types.
func ProcessObject(val reflect.Value, tag string) {
	switch val.Type().Kind() {
	case reflect.Pointer:
		if val.IsNil() {
			break
		}
		// Any object marked with a tag of the form `registry:"loadfrom-foo"` will
		// be loaded from the specified registry ("foo", in this example) as long
		// as a Defname field of type string was in the same struct. If it was then
		// the value of that field will be used as the key when loading this object
		// from the registry.
		loadfrom_tag := "loadfrom-"
		if strings.HasPrefix(tag, loadfrom_tag) {
			source := tag[len(loadfrom_tag):]
			logging.Debug("ProcessObject calling GetObject", "registry", source)
			GetObject(source, val.Interface())
		}
		ProcessObject(val.Elem(), tag)
	case reflect.Struct:
		for i := 0; i < val.NumField(); i++ {
			ProcessObject(val.Field(i), val.Type().Field(i).Tag.Get("registry"))
		}

	case reflect.Array:
		fallthrough
	case reflect.Slice:
		for i := 0; i < val.Len(); i++ {
			ProcessObject(val.Index(i), tag)
		}
	}

	// Anything that is tagged with autoload has its Load() method called if it
	// exists and has zero inputs and outputs.
	if tag == "autoload" {
		load := val.MethodByName("Load")
		if !load.IsValid() && val.CanAddr() {
			load = val.Addr().MethodByName("Load")
		}
		if load.IsValid() && load.Type().NumIn() == 0 && load.Type().NumOut() == 0 {
			load.Call(nil)
		}
	}
}

// Walks recursively through the specified directory and loads all files with
// the specified suffix and loads them into the specified registry using
// RegisterObject(). format should either be "json" or "gob" Files begining
// with '.' are ignored in this process.
func RegisterAllObjectsInDir(registry_name, dir, suffix, format string) {
	logging.Info("Registering directory", "dir", dir)
	reg, ok := registry_registry[registry_name]
	if !ok {
		logging.Error("Tried to load objects into an unknown registry", "registry-name", registry_name)
	}
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		_, filename := filepath.Split(path)
		if err != nil {
			panic(fmt.Errorf("Error walking directory: %w", err))
		}
		if strings.HasPrefix(filename, ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !info.IsDir() {
			if strings.HasSuffix(info.Name(), suffix) {
				target := reflect.New(reg.Type().Elem().Elem())
				err = LoadAndProcessObject(path, format, target.Interface())
				if err == nil {
					RegisterObject(registry_name, target.Interface())
				} else {
					logging.Error("Error loading file", "path", path, "err", err)
				}
			}
		}
		return nil
	})
	logging.Info("Completed directory", "dir", dir)
}
