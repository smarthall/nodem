package modem

import (
	"bytes"
	"fmt"
	"io"
)

type State int

const (
	Command State = iota
	OnLine
)

type Modem struct {
	client io.ReadWriteCloser
	state  State
}

func New(client io.ReadWriteCloser) *Modem {
	return &Modem{
		client: client,
		state:  Command,
	}
}

func (m *Modem) Run() {
	var b bytes.Buffer
	c := make([]byte, 1)
	for {
		// Build a buffer from the incoming bytes
		_, err := m.client.Read(c)
		if err == io.EOF {
			break
		}
		b.Write(c)

		// Check if the buffer contains a command
		if m.state == Command && c[0] == '\r' {
			m.processCommand(b.String())
			b.Reset()
		}
	}
}

func (m *Modem) processCommand(command string) {
	fmt.Printf("Got Command: %s\n", command)
}
