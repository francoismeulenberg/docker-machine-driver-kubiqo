package main

import (
	kubiqo "driver/driver" // update to the module path if you rename the module

	"github.com/docker/machine/libmachine/drivers/plugin"
)

func main() {
	plugin.RegisterDriver(kubiqo.NewDriver("", ""))
}
