package main

import (
	"github.com/docker/machine/libmachine/drivers/plugin"
	kubiqo "github.com/francoismeulenberg/docker-machine-driver-kubiqo/driver"
)

func main() {
	plugin.RegisterDriver(kubiqo.NewDriver("", ""))
}
