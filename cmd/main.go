package main

import (
    "github.com/khyallin/shardkv-dashboard/internal/app"
)

func main() {
	app := app.New()
	app.Run()
}
