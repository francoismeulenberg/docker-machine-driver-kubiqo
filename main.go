package main

import (
	"github.com/rancher/machine/libmachine/drivers/plugin"

	// change this import to match your go.mod module path
	"github.com/francoismeulenberg/docker-machine-driver-kubiqo/drivers"
)

func main() {
	plugin.RegisterDriver(drivers.NewDriver)
}
