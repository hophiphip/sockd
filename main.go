package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"time"
)

func main() {
	//cmd := exec.Command("/bin/sh", "-c", "while true; do echo 'loop'; sleep 5; done")
	cmd := exec.Command("cat")

	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}
	defer stdout.Close()

	stdoutScanner := bufio.NewScanner(stdout)

	if err = cmd.Start(); err != nil {
		log.Fatal(err)
	}

	go func() {
		for stdoutScanner.Scan() {
			fmt.Printf("[%s] >> %s\n", time.Now().Format(time.RFC850), stdoutScanner.Text())
		}
	}()

	go func() {
		consoleReader := bufio.NewReader(os.Stdin)

		// Indefinetly read from os.Stdin
		// NOTE: it is probably impossible to tell when cmd.Stdin requires input
		for {
			fmt.Printf("[%s] << ", time.Now().Format(time.RFC850))
			input, err := consoleReader.ReadString('\n')
			if err != nil {
				log.Fatal(err)
			}
			fmt.Print("\n")

			stdin.Write([]byte(input))
		}
	}()

	if err = cmd.Wait(); err != nil {
		log.Fatal(err)
	}
}
