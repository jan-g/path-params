package main

import (
	"fmt"

	"github.com/jan-g/path-params/cache"
	"github.com/jan-g/path-params/database"
	"time"
	"github.com/jan-g/path-params/model"
)

func main() {
	db := database.NewDatabase(nil)
	cache := cache.NewCache(db, time.Duration(5) * time.Millisecond, time.Duration(1) * time.Second)

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
	data, params, err := cache.GetRoute("testApp", "/some/kind/of/path")
	failIf(err)
	fmt.Println("Data: ", data, " and params=", params)

	fmt.Println("Getting route data for /some/other - should be nil")
	data, params, err = cache.GetRoute("testApp", "/some/other")
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
	data, params, err = cache.GetRoute("testApp", "/some/other")
	failIf(err)
	fmt.Println("Data: ", data, " and params=", params)

	fmt.Println("Getting route data for /blah/foo")
	data, params, err = cache.GetRoute("testApp", "/blah/foo")
	failIf(err)
	fmt.Println("Data: ", data, " and params=", params)

	fmt.Println("Getting route data for foo: /")
	data, params, err = cache.GetRoute("foo", "/")
	failIf(err)
	fmt.Println("Data: ", data, " and params=", params)
	fmt.Println()

	fmt.Println("setting up testApp /group/:gid/state/&state")
	db.SetRoute("testApp", "/group/:/state/&", model.RouteData{
		Params: []string{"gid", "state"},
	})
	db.Print()
	fmt.Println("trying with testApp /group/g1/state/s1")
	data, params, err = cache.GetRoute("testApp", "/group/g1/state/s1")
	failIf(err)
	fmt.Println("Data:", data, " and params=", params)
	fmt.Println("Pausing for positive cache time and trying again")
	time.Sleep(time.Duration(5) * time.Millisecond)
	data, params, err = cache.GetRoute("testApp", "/group/g1/state/s1")
	failIf(err)
	fmt.Println("Data:", data, " and params=", params)
	fmt.Println("trying with testApp /group/g1/state/s1/s2/s3")
	data, params, err = cache.GetRoute("testApp", "/group/g1/state/s1/s2/s3")
	failIf(err)
	fmt.Println("Data:", data, " and params=", params)
	fmt.Println("trying with testApp /group/g1/state")
	data, params, err = cache.GetRoute("testApp", "/group/g1/state")
	failIf(err)
	fmt.Println("Data:", data, " and params=", params)
	fmt.Println("trying with testApp /group/g1/state/")
	data, params, err = cache.GetRoute("testApp", "/group/g1/state/")
	failIf(err)
	fmt.Println("Data:", data, " and params=", params)
	fmt.Println()

	fmt.Println("Deleting the remaining routes")
	failIf(db.DelRoute("testApp", "/some/other"))
	failIf(db.DelRoute("testApp", "/:/foo"))
	db.Print()
	fmt.Println()

}

func failIf(err error) {
	if err != nil {
		panic(err)
	}
}
