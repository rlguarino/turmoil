package main

import (
	"bufio"
	"log/slog"
	"os/exec"
	"sync"
)

// TODO: Create a dir for run results, save stdERR to that file, save messages to that file, save some results to that directory as well?
// TODO: Create a client that will send a specific QPS!
// TODO: It would be cool to have a HTTP interface for results, then maybe the runs could be done on a K8s cluster?
// TODO: I feel like it would be nicer to run docker containers, maybe way later.
func main() {
	serverExecPath := "./exit-server"
	c := exec.Command(serverExecPath)
	serverStdout, err := c.StdoutPipe()
	if err != nil {
		panic(err)
	}
	serverStderr, err := c.StderrPipe()
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
		stderr := bufio.NewScanner(serverStderr)
		for stderr.Scan() {
			line := stderr.Text()
			slog.Info("stderr", "line", line)
		}
		slog.Info("finish reading from stderr")
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
