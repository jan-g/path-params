package main

import (
	"github.com/jan-g/path-params/database"
)

func main() {
	db := database.NewDatabase(nil)
	failIf(db.AddApp("testApp"))
	failIf(db.SetRoute("testApp", "/some/kind/of/path", "/some/kind/of/path data goes here"))
	failIf(db.SetRoute("testApp", "/some/other/kind/of/path", "/some/other/kind/of/path data goes here"))
	failIf(db.SetRoute("testApp", "/some/kind", "/some/kind is a short path"))
	failIf(db.SetRoute("testApp", "/:/foo", "/:param/foo has a path variable"))
	db.Print()
}

func failIf(err error) {
	if err != nil {
		panic(err)
	}
}
