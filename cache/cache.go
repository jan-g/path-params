package cache

import (
	"strings"
	"sync"
	"time"

	"github.com/jan-g/path-params/model"
	"github.com/jan-g/path-params/database"
	"fmt"
)

type Cache interface {
	GetRoute(string, string) (data *model.RouteData, pathVars map[string]string, err error)
}

type gen uint64

func NewCache(db database.DatabaseReader, positiveTTL time.Duration, negativeTTL time.Duration) Cache {
	return &cacheImpl{
		db: db,
		apps: map[string]appTuple{},
		positiveTTL: positiveTTL,
		negativeTTL: negativeTTL,
	}
}


type cacheImpl struct {
	db				database.DatabaseReader
	mu				sync.RWMutex
	apps			map[string]appTuple
	negativeExpire	time.Time
	positiveTTL		time.Duration
	negativeTTL		time.Duration
}

type appTuple struct {
	gen				gen
	pathNode		*pathNode	// This will be nil for negative caching
	expire			time.Time
}

type pathTuple struct {
	gen				gen
	pathNode		*pathNode
}

type pathNode struct {
	mu				sync.RWMutex
	gen				gen
	paths			map[string]pathTuple
	routeData		*model.RouteData
}


func (c *cacheImpl) GetRoute(app string, path string) (*model.RouteData, map[string]string, error) {
	gen, part, err := c.findApp(app)
	if part == nil || err != nil {
		return nil, nil, err
	}

	if path == "/" {
		path = ""
	}
	params := []string{}
	pieces := strings.Split(path, "/")[1:]
	prefix := ""
	fmt.Println("** app =", app, "; part =", part)
	search: for i, piece := range pieces {
		fmt.Println("**   searching for", piece, "; prefix =", prefix)
		var item string
		item, gen, part, err = c.nextPart(app, prefix, gen, part, piece, ":", "&")
		fmt.Println("**   -> item =", item, "; gen =", gen, "; part =", part, "; err =", err)
		if err != nil {
			return nil, nil, err
		}
		if part == nil {
			// No such configured route
			return nil, nil, nil
		}
		prefix = prefix + "/" + item
		switch item {
		case ":":
			// A single path element
			params = append(params, piece)
		case "&":
			// A "rest" parameter matches whatever's left
			params = append(params, strings.Join(pieces[i:], "/"))
			break search
		}
	}
	// Force a load of a leaf node, if required
	_, _, _, err = c.nextPart(app, prefix, gen, part)
	if err != nil {
		return nil, nil, err
	}
	paramMap := map[string]string{}
	for i, paramName := range part.routeData.GetParams() {
		paramMap[paramName] = params[i]
	}
	return part.routeData, paramMap, nil
}

func (c *cacheImpl) findApp(app string) (gen, *pathNode, error) {
	begin:
	c.mu.RLock()

	at, ok := c.apps[app]
	if !ok {
		// Total cache miss.
		c.mu.RUnlock()
		c.mu.Lock()		// Upgradable locks would be nice here and avoid having to recheck
		at, ok = c.apps[app]
		if !ok {
			// It's still missing; hit the DB
			pathPart, err := c.db.LookupApp(app)
			if err != nil {
				// Come kind of communication problem; percolate it
				c.mu.Unlock()
				return 0, nil, err
			}
			if pathPart == nil {
				// app doesn't exist; cache this fact negatively
				at = appTuple{
					gen: 0,
					expire: time.Now().Add(c.negativeTTL),
					pathNode: nil,
				}
			} else {
				// A hit; cache positively
				at = appTuple{
					gen: gen(pathPart.Generation),
					expire: time.Now().Add(c.positiveTTL),
					pathNode: &pathNode{
						gen: 0,
						paths: map[string]pathTuple{},
						routeData: pathPart.GetRoute(),
					},
				}
			}
			c.apps[app] = at
			c.mu.Unlock()
			return at.gen, at.pathNode, nil
		} else {
			// Someone else has spoken to the DB in the interim; loop and check again
			c.mu.Unlock()
			goto begin
		}
	} else {
		// We have some data - is it fresh?
		if time.Now().After(at.expire) {
			// We need to contact the DB for a refresh
			c.mu.RUnlock()
			c.mu.Lock()			// an upgradable lock would be preferable - we have to recheck
			at, ok = c.apps[app]
			if !ok {
				// It's been dropped from the cache. Loop and check again
				c.mu.Unlock()
				goto begin
			} else {
				if time.Now().After(at.expire) {
					// Still out-of-date, hit the DB
					pathPart, err := c.db.LookupApp(app)
					if err != nil {
						// Some kind of DB error
						c.mu.Unlock()
						return 0, nil, err
					}
					if pathPart == nil {
						// app doesn't exist
						at = appTuple{
							gen: 0,
							expire: time.Now().Add(c.negativeTTL),
							pathNode: nil,
						}
					} else {
						// We have an updated value
						at = appTuple{
							gen: gen(pathPart.GetGeneration()),
							expire: time.Now().Add(c.positiveTTL),
							pathNode: &pathNode{
								gen: 0,
								paths: map[string]pathTuple{},
								routeData: pathPart.GetRoute(),
							},
						}
					}
					c.apps[app] = at
					c.mu.Unlock()
					return at.gen, at.pathNode, nil
				} else {
					// Someone got here before us. Use this value.
					c.mu.Unlock()
					return at.gen, at.pathNode, nil
				}
			}
		} else {
			// The information is still within its use-by period
			c.mu.RUnlock()
			return at.gen, at.pathNode, nil
		}
	}
}

// This one is simpler: we don't have timeouts to consider
func (c *cacheImpl) nextPart(app string, prefix string, genSought gen, part *pathNode, items... string) (string, gen, *pathNode, error) {
begin:
	c.mu.RLock()

	if part.gen < genSought {
		// This node is out-of-date. We need to update it
		c.mu.RUnlock()
		c.mu.Lock()
		if part.gen < genSought {
			// Update the part with new data
			newPart, err := c.db.LookupPart(app, prefix)
			if err != nil {
				// Some kind of error
				c.mu.Unlock()
				return "", 0, nil, err
			}
			if newPart == nil {
				// This route's been deleted out from under us
				c.mu.Unlock()
				return "", 0, nil, nil
			}
			// Update the pathNode with the new information
			part.gen = gen(newPart.GetGeneration())
			part.routeData = newPart.GetRoute()
			newMap := map[string]pathTuple{}
			for hop, child := range newPart.GetChildren() {
				curChild, ok := part.paths[hop]
				if ok {
					newMap[hop] = pathTuple{
						gen: gen(child.GetGeneration()),
						pathNode: curChild.pathNode,
					}
				} else {
					newMap[hop] = pathTuple{
						gen: gen(child.GetGeneration()),
						pathNode: &pathNode{
							gen: 0,
							paths: map[string]pathTuple{},
							routeData: nil,
						},
					}
				}
			}
			part.paths = newMap
			// Loop and try again
			c.mu.Unlock()
			goto begin
		} else {
			// Someone else has gotten here before us; loo around to carry on
			c.mu.Unlock()
			goto begin
		}
	} else {
		// We've got an up-to-date node. Go looking for a matching item
		fmt.Println("****    items =", items, "; part.paths =", part.paths)
		for _, item := range items {
			fmt.Println("****      item =", item)
			child, ok := part.paths[item]
			if ok {
				c.mu.RUnlock()
				return item, child.gen, child.pathNode, nil
			}
		}
		c.mu.RUnlock()
		return "", 0, nil, nil
	}
}