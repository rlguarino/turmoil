// A server that starts and then exists.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
)

//TODO Maybe I can do something to automatically hook up structured json logging to STDERR?
// Actually should probably just make STDERR reference a specific file automatically and save that file in the reports.

type message struct {
	Type string
}

func main() {
	f, err := os.CreateTemp("temp", "server.*.log")
	if err != nil {
		panic(err)
	}
	log := slog.New(slog.NewTextHandler(f, nil))
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
			log.Info("got init message!")
			fmt.Println("{\"type\":\"started\"}")
			<-time.After(time.Second)
			fmt.Println("{\"type\":\"lifecycle\", \"lifecycle\":\"shutdown\"}")
			os.Exit(0)
		}
	}
}
