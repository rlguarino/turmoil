// A server that will echo the message is receives.
package main

import (
	"bufio"
	"encoding/json"
	"log/slog"
	"os"
)

type message struct {
	Type string
}

func main() {
	// Servers cannot log anything but turmoil messages to STDOUT.
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(log)
	stdin := bufio.NewScanner(os.Stdin)
	for stdin.Scan() {
		line := stdin.Bytes()
		msg := message{}
		err := json.Unmarshal(line, &msg)
		if err != nil {
			log.Error("Unmarshal error", "error", err, "line", line)
		}
		switch msg.Type {
		case "init":
			log.Info("got init message", "msg", msg)
		case "echo":
			log.Info("got echo message", "msg", msg)
		}
	}
}
