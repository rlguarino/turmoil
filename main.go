package main

import (
	"bufio"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"sync"
	"time"
)

func testTempDir(t time.Time) string {
	return fmt.Sprintf("store/%s", t.Format("20060102150405.000"))
}

// TODO: Create a dir for run results, save stdERR to that file, save messages to that file, save some results to that directory as well?
// TODO: Create a client that will send a specific QPS!
// TODO: It would be cool to have a HTTP interface for results, then maybe the runs could be done on a K8s cluster?
// TODO: I feel like it would be nicer to run docker containers, maybe way later.
func main() {
	now := time.Now()
	tempDir := testTempDir(now)
	// err := os.Mkdir("store", 0755)
	// if err != nil {
	// 	panic(err)
	// }
	err := os.Mkdir(tempDir, 0755)
	if err != nil {
		panic(err)
	}
	n1StderrFile, err := os.Create(fmt.Sprintf("%s/%s", tempDir, "n1.log"))
	if err != nil {
		panic(err)
	}

	serverExecPath := "./echo-server"
	c := exec.Command(serverExecPath)
	c.Stderr = n1StderrFile
	serverStdout, err := c.StdoutPipe()
	if err != nil {
		panic(err)
	}
	serverStdin, err := c.StdinPipe()
	if err != nil {
		panic(err)
	}

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		stdout := bufio.NewScanner(serverStdout)
		for stdout.Scan() {
			line := stdout.Text()
			slog.Info("stdout", "line", line)
		}
		slog.Info("finished reading from stdout")
	}()
	wg.Add(1)
	go func() {
		defer wg.Done()
		writer := bufio.NewWriter(serverStdin)
		writer.WriteString("{\"type\":\"init\"}\n")
		err := writer.Flush()
		if err != nil {
			slog.Error("error flushing", "command", c, "error", err)
		}
		slog.Info("finish writing to stdin")
	}()

	err = c.Start()
	if err != nil {
		slog.Error("Start returned an error", "error", err)
	} else {
		slog.Info("Start returned without error")
	}

	slog.Info("Waiting")
	wg.Wait()
	slog.Info("Wait finished")
}
