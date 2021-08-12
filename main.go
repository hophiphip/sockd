package main

import (
	"io"
	"log"
	"os"
	"os/exec"
)

func main() {
	cmd := exec.Command("ls", "-la")

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	go io.Copy(os.Stdout, stdout)

	if err = cmd.Start(); err != nil {
		log.Fatal(err)
	}

	if err = cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}
