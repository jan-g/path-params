package main

import (
	"fmt"
	"time"

	"github.com/jan-g/path-params/cache"
	"github.com/jan-g/path-params/database"
	"github.com/jan-g/path-params/model"
)

func main() {
	db := database.NewDatabase(nil)
	c := cache.NewCache(db, time.Duration(5) * time.Millisecond, time.Duration(1) * time.Second)

	failIf(db.AddApp("testApp"))
	failIf(db.SetRoute("testApp", "/some/kind/of/path", model.RouteData{Path:"/some/kind/of/path data goes here"}))
	failIf(db.SetRoute("testApp", "/some/other/kind/of/path", model.RouteData{Path:"/some/other/kind/of/path data goes here"}))
	failIf(db.SetRoute("testApp", "/some/kind", model.RouteData{Path:"/some/kind is a short path"}))
	failIf(db.SetRoute("testApp", "/:/foo", model.RouteData{
		Path:"/:param/foo has a path variable",
		Params: []string{"param"},
	}))
	db.Print()
	fmt.Println()

	fmt.Println("Deleting /some/kind")
	failIf(db.DelRoute("testApp", "/some/kind"))
	db.Print()
	fmt.Println()

	fmt.Println("Getting route data for /some/kind/of/path")
	data, params, err := c.GetRoute("testApp", "/some/kind/of/path")
	failIf(err)
	fmt.Println("Data: ", data, " and params=", params)

	fmt.Println("Getting route data for /some/other - should be nil")
	data, params, err = c.GetRoute("testApp", "/some/other")
	failIf(err)
	fmt.Println("Data: ", data, " and params=", params)

	fmt.Println("Deleting /some/kind/of/path")
	failIf(db.DelRoute("testApp", "/some/kind/of/path"))
	db.Print()
	fmt.Println()

	fmt.Println("Adding /some/other")
	failIf(db.SetRoute("testApp", "/some/other", model.RouteData{Path:"/some/other is a prefix route"}))
	db.Print()
	fmt.Println()

	fmt.Println("Deleting /some/other/kind/of/path - longer prefix first")
	failIf(db.DelRoute("testApp", "/some/other/kind/of/path"))
	db.Print()
	fmt.Println()

	db.AddApp("foo")
	db.SetRoute("foo", "/", model.RouteData{Path:"top-level route for foo"})
	db.Print()
	fmt.Println()

	fmt.Println("Getting route data for /some/other")
	data, params, err = c.GetRoute("testApp", "/some/other")
	failIf(err)
	fmt.Println("Data: ", data, " and params=", params)

	fmt.Println("Getting route data for /blah/foo")
	data, params, err = c.GetRoute("testApp", "/blah/foo")
	failIf(err)
	fmt.Println("Data: ", data, " and params=", params)

	fmt.Println("Getting route data for foo: /")
	data, params, err = c.GetRoute("foo", "/")
	failIf(err)
	fmt.Println("Data: ", data, " and params=", params)
	fmt.Println()

	fmt.Println("setting up testApp /group/:gid/state/&state")
	db.SetRoute("testApp", "/group/:/state/&", model.RouteData{
		Params: []string{"gid", "state"},
	})
	db.Print()
	fmt.Println("trying with testApp /group/g1/state/s1")
	data, params, err = c.GetRoute("testApp", "/group/g1/state/s1")
	failIf(err)
	fmt.Println("Data:", data, " and params=", params)
	fmt.Println("Pausing for positive cache time and trying again")
	time.Sleep(time.Duration(5) * time.Millisecond)
	data, params, err = c.GetRoute("testApp", "/group/g1/state/s1")
	failIf(err)
	fmt.Println("Data:", data, " and params=", params)
	fmt.Println("trying with testApp /group/g1/state/s1/s2/s3")
	data, params, err = c.GetRoute("testApp", "/group/g1/state/s1/s2/s3")
	failIf(err)
	fmt.Println("Data:", data, " and params=", params)
	fmt.Println("trying with testApp /group/g1/state")
	data, params, err = c.GetRoute("testApp", "/group/g1/state")
	failIf(err)
	fmt.Println("Data:", data, " and params=", params)
	fmt.Println("trying with testApp /group/g1/state/")
	data, params, err = c.GetRoute("testApp", "/group/g1/state/")
	failIf(err)
	fmt.Println("Data:", data, " and params=", params)
	fmt.Println()

	fmt.Println("Deleting the remaining routes")
	failIf(db.DelRoute("testApp", "/some/other"))
	failIf(db.DelRoute("testApp", "/:/foo"))
	db.Print()
	fmt.Println()

	fmt.Println("Illustrative routes from the README follow")
	db = database.NewDatabase(nil)

	db.AddApp("test")
	db.SetRoute("test", "/graph", model.RouteData{Path: "1. /graph"})
	db.SetRoute("test", "/graph/view", model.RouteData{Path: "2. /graph/view"})
	db.SetRoute("test", "/graph/:/stage/:", model.RouteData{
		Path: "3. /graph/:/stage/:",
		Params: []string{"graphId", "stageId"},
	})
	db.SetRoute("test", "/graph/&", model.RouteData{Path: "ERROR: never matched"})
	db.SetRoute("test", "/graph/:/&", model.RouteData{
		Path: "5. /graph/:/&",
		Params: []string{"graphId", "rest"},
	})
	db.SetRoute("test", "/graph/:", model.RouteData{
		Path: "extra: /graph/:",
		Params: []string{"gId"},    // param names don't have to be identical
	})
	db.Print()
	fmt.Println()

	c = cache.NewCache(db, time.Duration(5) * time.Millisecond, time.Duration(1) * time.Second)
	// Run through the examples
	for _, eg := range []string{
		"/graph", 				// matches 1
		"/graph/view",			// matches 2
		"/graph/view/",			// unmatched
		"/graph/view/foo",		// unmatched
		"/graph/2934/stage/4372",	// matches 3
		"/graph/4234",			// extra pattern should match this
		"/graph/4234/",			// matches 5 (rest parameter is "")
		"/graph/4234/x/y/z",	// matches 5 (rest parameter is "x/y/z")
	} {
		fmt.Println("Looking up route:", eg)
		data, params, err = c.GetRoute("test", eg)
		failIf(err)
		fmt.Println("Found the following:", data, " with variables", params)
		fmt.Println()
	}
}


func failIf(err error) {
	if err != nil {
		panic(err)
	}
}
