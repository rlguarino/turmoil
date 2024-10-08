// A server that will echo the message is receives.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
)

type TurmoilMessage struct {
	Source       string
	Destination  string
	Type         string
	Id           string
	InResponseTo string
	Body         json.RawMessage
}

type InitMessage struct {
	Name string
}

type EchoBody struct {
	Message string
}

func processesInitMessage(stdin *bufio.Scanner) (string, error) {
	slog.Info("waiting for init message")
	ok := stdin.Scan()
	if !ok {
		return "", fmt.Errorf("scanning stdin for init message failed with error:%w", stdin.Err())
	}
	msg := TurmoilMessage{}
	line := stdin.Bytes()
	defer slog.Info("proccessed init message", "msg", string(line))
	err := json.Unmarshal(line, &msg)
	if err != nil {
		slog.Error("Unmarshal error", "error", err, "line", line)
	}
	if msg.Type != "init" {
		return "", fmt.Errorf("unexpected first message type, expected \"init\", got:%q", msg)
	}
	initBody := InitMessage{}
	err = json.Unmarshal(msg.Body, &initBody)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling init message: %w", err)
	}
	return initBody.Name, nil
}

func main() {
	// Servers cannot log anything but turmoil messages to STDOUT.
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(log)
	stdin := bufio.NewScanner(os.Stdin)
	output := json.NewEncoder(os.Stdout)

	nodeName := "unknown"
	var err error
	log.Debug("waiting for init message")
	nodeName, err = processesInitMessage(stdin)
	if err != nil {
		panic(err)
	}
	log.Debug("processed init message", "name", nodeName)
	for stdin.Scan() {
		line := stdin.Bytes()
		msg := TurmoilMessage{}
		err := json.Unmarshal(line, &msg)
		if err != nil {
			log.Error("Unmarshal error", "error", err, "line", line)
		}
		switch msg.Type {
		case "init":
			log.Info("got init message", "msg", msg)
		case "echo":
			log.Info("got echo message", "msg", msg)
			echo := EchoBody{}
			err := json.Unmarshal(msg.Body, &echo)
			if err != nil {
				log.Info("error unmashalling EchoBody", "error", err)
				continue
			}

			respMessage := TurmoilMessage{
				Source:       nodeName,
				Destination:  msg.Source,
				Type:         "echo_ok",
				InResponseTo: msg.Id,
				Body:         msg.Body,
			}
			err = output.Encode(respMessage)
			if err != nil {
				log.Error("error encoding response to echo", "error", err, "message", respMessage)
			}
		}
	}
}
