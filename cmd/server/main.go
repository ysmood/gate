package main

import (
	"flag"

	"github.com/ysmood/gate/lib/conf"
	"github.com/ysmood/gate/lib/server"
)

var path = flag.String("c", "config.json", "the config json file path")

func main() {
	flag.Parse()

	server.New(conf.New(*path)).Serve()
}
