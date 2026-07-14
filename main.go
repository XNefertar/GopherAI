package main

import (
	"GopherAI/app"
)

func main() {
	if err := app.New().Run(); err != nil {
		panic(err)
	}
}
