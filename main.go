package main

import (
	"github.com/docker/machine/libmachine/drivers/plugin"
	exoscale "github.com/francoismeulenberg/docker-machine-driver-kubiqo/drivers"
)

func main() {
	plugin.RegisterDriver(exoscale.NewDriver)
}
