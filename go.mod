module github.com/rancher/machine

go 1.24.0

toolchain go1.24.7

replace github.com/urfave/cli => github.com/urfave/cli v1.11.1-0.20151120215642-0302d3914d2a // newer versions of this will break the rpc binding code