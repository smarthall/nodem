package modem

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
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
		fmt.Printf("Read: %s:\"%s\"\n", hex.EncodeToString(c), string(c))
		b.Write(c)

		// TODO: Only if Echo is enabled
		// TODO: Only echo non-editing characters
		fmt.Printf("Write: %s:\"%s\"\n", hex.EncodeToString(c), string(c))
		m.client.Write(c)

		// Check if the buffer contains a command
		if m.state == Command && c[0] == 0x0d {
			m.processCommand(strings.TrimSpace(b.String()))
			b.Reset()
			continue
		}
	}
}

func (m *Modem) processCommand(command string) {
	fmt.Printf("Command: %s\n", command)
	m.client.Write([]byte("\r\nOK\r\n"))
}
