package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

type Store string

// SetupStore will create a store for use in a specific run.
// We will create all of the dictories and update the "latest" symlink to point at the latest store
func SetupStore(now time.Time) Store {
	base := "store"
	storeDir := filepath.Join(base, now.Format("20060102150405.000"))
	err := os.MkdirAll(storeDir, 0755)
	if err != nil {
		panic(err)
	}
	latestSymlink := filepath.Join(base, "latest")
	os.Remove(latestSymlink)
	err = os.Symlink(storeDir, latestSymlink)
	if err != nil {
		panic(err)
	}
	return Store(storeDir)
}

// createStoreFile creates or truncates a file and all parent the parent directories in the given store
// if filename contains a direcotry then the directory will be created before the file with mode 0755
func (s Store) createStoreFile(filename string) (*os.File, error) {
	err := os.MkdirAll(filepath.Join(string(s), filepath.Dir(filename)), 0755)
	if err != nil {
		return nil, err
	}
	return os.Create(filepath.Join(string(s), filename))
}

type Network struct {
	store          Store
	nodeQueuesLock sync.Mutex
	nodeQueues     map[string]chan string
}

func NewNetwork(s Store) *Network {
	return &Network{
		s,
		sync.Mutex{},
		make(map[string]chan string),
	}
}

func (n *Network) sendMsg(dst, msg string) {
	c, ok := n.nodeQueues[dst]
	if !ok {
		return
	}
	c <- msg
}

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

func (n *Network) StartNode(name string, executablePath string) error {
	log := slog.With("node", name)
	stderrFh, err := n.store.createStoreFile(filepath.Join("node_logs", fmt.Sprintf("%s.log", name)))
	if err != nil {
		return fmt.Errorf("error creating node log file: %w", err)
	}
	c := exec.Command(executablePath)
	c.Stderr = stderrFh
	nodeStdout, err := c.StdoutPipe()
	if err != nil {
		return fmt.Errorf("error creating StdoutPipe: %w", err)
	}
	nodeStdin, err := c.StdinPipe()
	if err != nil {
		return fmt.Errorf("error creating StdinPipe: %w", err)
	}
	inbox := make(chan string, 100)
	n.nodeQueuesLock.Lock()
	if _, ok := n.nodeQueues[name]; ok {
		panic(fmt.Sprintf("node already exists with name:%v", name))
	}
	n.nodeQueues[name] = inbox
	n.nodeQueuesLock.Unlock()
	go func() {
		stdout := bufio.NewScanner(nodeStdout)
		for stdout.Scan() {
			line := stdout.Bytes()
			log.Info("got line", "node", name, "line", line)
			// TODO: Queue the message in the destinations channel
			log.Debug("stdout", "line", line)
			msg := TurmoilMessage{}
			err := json.Unmarshal(line, &msg)
			if err != nil {
				panic(err)
			}
			n.nodeQueues[msg.Destination] <- string(line)
		}
		log.Debug("finished reading from stdout")
	}()

	go func() {
		writer := bufio.NewWriter(nodeStdin)
		for {
			select {
			case msg := <-inbox:
				log := log.With("msg", msg)
				log.Info("got msg from inbox", "msg", msg)
				_, err := writer.WriteString(msg + "\n")
				if err != nil {
					panic(err)
				}
				log.Info("flushing")
				err = writer.Flush()
				if err != nil {
					log.Error("error flushing", "command", c, "error", err)
				}
				log.Info("flushed")
			}
		}
	}()
	if err = c.Start(); err != nil {
		return fmt.Errorf("c.Start() error: %w", err)
	}

	body, err := json.Marshal(InitMessage{Name: name})
	if err != nil {
		return fmt.Errorf("error marshalling init message: %w", err)
	}
	messageBytes, err := json.Marshal(TurmoilMessage{
		Source:      "network",
		Destination: name,
		Type:        "init",
		Body:        body,
	})
	if err != nil {
		return fmt.Errorf("error marshalling base turmoil message: %w", err)
	}
	log.Info("sending init message", "message", string(messageBytes))
	inbox <- string(messageBytes)
	return nil
}

// TODO: Create a client that will send a specific QPS!
// TODO: It would be cool to have a HTTP interface for results, then maybe the runs could be done on a K8s cluster?
// TODO: I feel like it would be nicer to run docker containers, maybe way later.
func main() {
	start := time.Now()
	store := SetupStore(start)
	network := NewNetwork(store)

	serverExecPath := "./echo-server"
	err := network.StartNode("n1", serverExecPath)
	if err != nil {
		slog.Error("Start returned an error", "error", err)
	} else {
		slog.Debug("Start returned without error")
	}
	err = network.StartNode("c1", "./echo-client")
	if err != nil {
		slog.Error("Start returned an error", "error", err)
	} else {
		slog.Debug("Start returned without error")
	}

	slog.Info("Waiting")
	<-make(chan struct{})
}
