package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"
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

type EchoMessage struct {
	Message string
}

type EchoResponse struct {
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
	// Nodes cannot log anything but turmoil messages to STDOUT.
	log := slog.New(slog.NewTextHandler(os.Stderr, nil))
	slog.SetDefault(log)
	stdin := bufio.NewScanner(os.Stdin)
	nodeName := "unknown"
	var err error
	log.Debug("waiting for init message")
	nodeName, err = processesInitMessage(stdin)
	if err != nil {
		panic(err)
	}
	log.Debug("processed init message", "name", nodeName)
	go func() {
		for stdin.Scan() {
			line := stdin.Bytes()
			msg := TurmoilMessage{}
			err := json.Unmarshal(line, &msg)
			if err != nil {
				log.Warn("unable to unmarshal turmoil message", "raw_line", line)
				continue
			}
			switch msg.Type {
			case "echo_ok":
				echoBody := EchoMessage{}
				json.Unmarshal(msg.Body, &echoBody)
				log.Info("got echo_ok message", "msg", msg)
			default:
				log.Info("got unexpected message", "msg", "msg")
			}

		}
	}()
	encoder := json.NewEncoder(os.Stdout)
	id := 0
	for {
		id++
		time.Sleep(time.Second / 2)
		msgBody, err := json.Marshal(EchoMessage{
			Message: "hello world",
		})
		if err != nil {
			panic(err)
		}
		baseMsg := TurmoilMessage{
			Source:      nodeName,
			Destination: "n1",
			Type:        "echo",
			Id:          fmt.Sprintf("%d", id),
			Body:        []byte(msgBody),
		}
		log.Info("sending message", "message", baseMsg)
		err = encoder.Encode(baseMsg)
		if err != nil {
			panic(err)
		}
	}
}
