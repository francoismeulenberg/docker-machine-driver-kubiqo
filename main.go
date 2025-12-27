package main

import (
	"github.com/docker/machine/libmachine/drivers/plugin"
	exoscale "github.com/francoismeulenberg/docker-machine-driver-kubiqo/driver"
)

func main() {
	plugin.RegisterDriver(exoscale.NewDriver("", ""))
}
