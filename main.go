package main

import (
	"github.com/docker/machine/libmachine/drivers/plugin"
	"github.com/francoismeulenberg/docker-machine-driver-kubiqo/driver"
)

func main() {
	plugin.RegisterDriver(driver.NewDriver("", ""))
}
