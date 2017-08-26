package main

import (
	"fmt"

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
	fmt.Println()

	fmt.Println("Deleting /some/kind")
	failIf(db.DelRoute("testApp", "/some/kind"))
	db.Print()
	fmt.Println()


	fmt.Println("Deleting /some/kind/of/path")
	failIf(db.DelRoute("testApp", "/some/kind/of/path"))
	db.Print()
	fmt.Println()

	failIf(db.SetRoute("testApp", "/some/other", "/some/other is a prefix route"))
	db.Print()
	fmt.Println()

	fmt.Println("Deleting /some/other/kind/of/path - longer prefix first")
	failIf(db.DelRoute("testApp", "/some/other/kind/of/path"))
	db.Print()
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
