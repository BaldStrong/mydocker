package main

import (
	"example/chap3/container"
	"os"

	log "github.com/sirupsen/logrus"
)

func Run(tty bool, command string) {
	parent := container.NewParentProcess(tty, command)
	if err := parent.Start(); err != nil {
		log.Fatal(err)
	}
	parent.Wait()
	os.Exit(-1)
}
