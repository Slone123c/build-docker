package main

import (
	"fmt"

	"build-docker/container"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var runCommand = cli.Command{
	Name:  "run",
	Usage: "Create a new container with namespace: mydocker run -it [command]",
	Flags: []cli.Flag{
		cli.BoolFlag{
			Name:  "it",
			Usage: "enable stdin/stdout and interactive mode",
		},
	},
	Action: func(context *cli.Context) error {
		if context.NArg() < 1 {
			return fmt.Errorf("please provide a command to run")
		}
		tty := context.Bool("it")
		cmdArray := context.Args()
		Run(tty, cmdArray)
		return nil
	},
}

var initCommand = cli.Command{
	Name:  "init",
	Usage: "Init container process run user's process in container. Do not call it outside",
	Action: func(context *cli.Context) error {
		log.Infof("init come on")
		err := container.RunContainerInitProcess()
		return err
	},
}
