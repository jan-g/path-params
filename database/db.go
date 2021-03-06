package database

import (
	"github.com/jan-g/path-params/model"
	"fmt"
	"strings"
	"sync"
	"sort"
	"reflect"
)

type DatabaseReader interface {
	LookupApp(string)			(*model.PathPart, error)
	LookupPart(string, string)	(*model.PathPart, error)
}

type DatabaseWriter interface {
	AddApp(string)  error
	DelApp(string)  error
	SetRoute(string, string, model.RouteData)  error
	DelRoute(string, string)  error
}

type Database interface {
	DatabaseReader
	DatabaseWriter
	Print()
}

type inMemDb struct {
	mu sync.RWMutex
	paths map[string]*model.PathPart
}

func (db *inMemDb) LookupApp(app string) (*model.PathPart, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	pp, ok := db.paths[app]
	if !ok {
		// a negative result is still okay, a valid response
		return nil, nil
	}
	return pp, nil
}

func (db *inMemDb) LookupPart(app string, prefix string) (*model.PathPart, error) {
	db.mu.RLock()
	defer db.mu.RUnlock()

	pp, ok := db.paths[app + prefix]
	if !ok {
		return nil, fmt.Errorf("app %v route %v not found", app, prefix)
	}
	return pp, nil
}

func (db *inMemDb) AddApp(app string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, ok := db.paths[app]
	if ok {
		return fmt.Errorf("app %v already exists", app)
	}
	db.paths[app] = &model.PathPart{
		Path: app,
		Generation: 0,
		Children: map[string]*model.PathPart_ChildNode{},
		Route: nil,
	}
	return nil
}


func (db *inMemDb) DelApp(app string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	_, ok := db.paths[app]
	if !ok {
		return fmt.Errorf("app %v does not exist", app)
	}
	for k := range db.paths {
		if strings.HasPrefix(k, app + "/") {
			delete(db.paths, k)
		}
	}
	delete(db.paths, app)
	return nil
}

func (db *inMemDb) SetRoute(app string, path string, data model.RouteData) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	node, ok := db.paths[app]
	if !ok {
		return fmt.Errorf("app %v does not exist", app)
	}

	nextGen := node.Generation + 1

	for _, piece := range splitPath(path) {
		// Update the generation of the parent node
		node, ok = db.paths[app + piece.prefix]
		if ok {
			node.Generation = nextGen
		} else {
			node = &model.PathPart{
				Path: app + piece.prefix,
				Generation: nextGen,
				Children: map[string]*model.PathPart_ChildNode{},
			}
			db.paths[app + piece.prefix] = node
		}
		if piece.next != "" {
			child, ok := node.Children[piece.next]
			if ok {
				child.Generation = nextGen
			} else {
				node.Children[piece.next] = &model.PathPart_ChildNode{piece.next, nextGen}
			}
		} else {
			node.Route = &data
		}
	}

	return nil
}

// Split a path into a series of prefixes and 'next parts'
// "/"      -> (prefix, part) ("", -)  [as a special case]
// ""       ->                ("", -)
// "/a"     ->                ("", a), ("/a", -)
// "/a/b/c" -> (prefix, part) ("", a), ("/a", b), ("/a/b", c), ("/a/b/c", -)
type prefixPath struct {
	prefix string
	next string
}

func splitPath(path string) []prefixPath {
	parts := []prefixPath{}

	if path == "/" {
		path = ""
	}
	bits := strings.Split(path, "/")[1:]
	prefix := ""
	for _, piece := range bits {
		parts = append(parts, prefixPath{prefix, piece})
		prefix = prefix + "/" + piece
	}
	return append(parts, prefixPath{prefix, ""})
}

func (db *inMemDb) DelRoute(app string, path string) error {
	db.mu.Lock()
	defer db.mu.Unlock()

	node, ok := db.paths[app]
	if !ok {
		return fmt.Errorf("app %v does not exist", app)
	}

	nextGen := node.Generation + 1

	node, ok = db.paths[app + path]
	if !ok || node.Route == nil {
		return fmt.Errorf("app %v does not have a route for %v", app, path)
	}

	erase := len(node.Children) == 0

	splits := splitPath(path)
	for i := len(splits) - 1; i >= 0; i-- {
		prefix, next := splits[i].prefix, splits[i].next
		node = db.paths[app + prefix]
		if next != "" {
			if erase {
				delete(node.Children, next)
			} else {
				node.Children[next].Generation = nextGen
			}
		} else {
			node.Route = nil
		}
		if len(node.Children) == 0 && node.Route == nil && prefix != "" {
			// This can only be true if erase was already true.
			delete(db.paths, app + prefix)
		} else {
			node.Generation = nextGen
			erase = false
		}
	}
	return nil
}

func NewDatabase(config interface{}) Database {
	return &inMemDb{
		paths: map[string]*model.PathPart{},
		mu: sync.RWMutex{},
	}
}

func (db *inMemDb) Print() {
	for _, k := range sortedKeys(db.paths) {
		v := db.paths[k]
		fmt.Printf("Prefix: %v gen %v ", k, v.GetGeneration())
		if len(v.Children) != 0 {
			fmt.Printf("    [")
			for n, c := range v.Children {
				fmt.Printf("%v#%v ", n, c.GetGeneration())
			}
			fmt.Printf("]")
		}
		if v.Route != nil {
			fmt.Printf(" route data: %v", v.GetRoute())
		}
		fmt.Println()
	}
}

func sortedKeys(m interface{}) []string {
	v := reflect.ValueOf(m)
	if v.Kind() != reflect.Map {
		panic("not a map!")
	}

	vs := v.MapKeys()
	ks := make([]string, 0, len(vs))
	for _, k := range vs {
		ks = append(ks, k.Interface().(string))  // sigh
	}
	sort.Strings(ks)
	return ks
}