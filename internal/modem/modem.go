package modem

import (
	"bytes"
	"fmt"
	"io"
	"math"
	"strings"
)

type State int

const (
	Command State = iota
	OnLine
)

type ModemVolume int

const (
	ModemVolumeLow ModemVolume = iota
	ModemVolumeMedium
	ModemVolumeHigh
)

type SpeakerMode int

const (
	SpeakerAlwaysOff SpeakerMode = iota
	SpeakerOnUntilCarrierDetect
	SpeakerAlwaysOn
	SpeakerOnAfterDiallingUntilCarrierDetect
)

type ModemRegisters struct {
	ringToAnswerAfter int  // S0
	endOfLine         byte // S3
	lineFeed          byte // S4
	backSpace         byte // S5
	waitForCarrierSec int  // S7

	// Bits in S95
	connectWithCarrierSpeed bool
	connectAppendARQ        bool
	displayCarrier          bool
	displayProtocol         bool
	displayCompression      bool
}

type Modem struct {
	client      io.ReadWriteCloser
	state       State
	lastCommand string
	echo        bool
	volume      ModemVolume
	speaker     SpeakerMode

	register ModemRegisters
}

func New(client io.ReadWriteCloser) *Modem {
	m := Modem{
		client:      client,
		state:       Command,
		lastCommand: "",
		echo:        true,
		volume:      ModemVolumeLow,
		speaker:     SpeakerOnAfterDiallingUntilCarrierDetect,
	}

	m.FactoryReset()

	return &m
}

func (m *Modem) FactoryReset() {
	m.volume = ModemVolumeLow
	m.speaker = SpeakerOnAfterDiallingUntilCarrierDetect

	m.register = ModemRegisters{
		endOfLine:         0x0d,
		lineFeed:          0x0a,
		backSpace:         0x08,
		waitForCarrierSec: 60,

		connectWithCarrierSpeed: false,
		connectAppendARQ:        false,
		displayCarrier:          false,
		displayProtocol:         false,
		displayCompression:      false,
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

		if m.echo {
			// TODO: Only echo non-editing characters
			m.client.Write(c)
		}

		// Check if the buffer contains a command
		if m.state == Command && c[0] == m.register.endOfLine {
			m.processCommand(strings.TrimSpace(b.String()))
			b.Reset()
			continue
		}
	}
}

func (m *Modem) processCommand(command string) {
	command = strings.TrimSpace(command)

	if len(command) < 2 {
		fmt.Printf("Invalid Command %s\n", command)
		m.sendResponse("ERROR")
		return
	}

	if command[0:2] == "AT" {
		err := m.processAtCommand(command[2:])
		if err != nil {
			fmt.Printf("Error processing AT Command: %s\n", err)
			m.sendResponse("ERROR")
			return
		}

		// Save the last command
		m.lastCommand = command

		return
	}

	if command[0:2] == "A/" {
		err := m.processAtCommand(m.lastCommand)
		if err != nil {
			fmt.Printf("Error processing AT Command: %s\n", err)
			m.sendResponse("ERROR")
			return
		}
	}

	fmt.Printf("Unknown Command: %s\n", command)
	m.sendResponse("ERROR")
}

func getNumber(s string) (int, int, error) {
	if len(s) == 0 {
		return 0, 0, fmt.Errorf("empty string")
	}

	if s[0] < '0' || s[0] > '9' {
		return 0, 0, fmt.Errorf("invalid number")
	}

	num := 0
	processed := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			num = num*int(math.Round(math.Pow10(processed))) + int(c-'0')
			processed += 1
		} else {
			return num, processed, nil
		}
	}

	return num, processed, nil
}

func (m *Modem) processAtCommand(command string) error {
	if len(command) == 0 {
		m.sendResponse("OK")
		return nil
	}

	fmt.Printf("Processing AT Command: %s\n", command)

	switch command[0] {
	case '&':
		return m.processAtCommandExtended(command[1:])
	case 'E':
		val := command[1]
		if val == '0' {
			m.echo = false
			return m.processAtCommand(command[2:])
		} else if val == '1' {
			m.echo = true
			return m.processAtCommand(command[2:])
		} else {
			return fmt.Errorf("invalid echo value")
		}
	case 'L':
		vol := command[1]

		switch vol {
		case '0':
			m.volume = ModemVolumeLow
			return m.processAtCommand(command[2:])
		case '1':
			m.volume = ModemVolumeLow
			return m.processAtCommand(command[2:])
		case '2':
			m.volume = ModemVolumeMedium
			return m.processAtCommand(command[2:])
		case '3':
			m.volume = ModemVolumeHigh
			return m.processAtCommand(command[2:])
		}

		return fmt.Errorf("invalid volume: %c", vol)
	case 'M':
		mode := command[1]

		switch mode {
		case '0':
			m.speaker = SpeakerAlwaysOff
			return m.processAtCommand(command[2:])
		case '1':
			m.speaker = SpeakerOnUntilCarrierDetect
			return m.processAtCommand(command[2:])
		case '2':
			m.speaker = SpeakerAlwaysOn
			return m.processAtCommand(command[2:])
		case '3':
			m.speaker = SpeakerOnAfterDiallingUntilCarrierDetect
			return m.processAtCommand(command[2:])
		}

		return fmt.Errorf("invalid speaker mode: %c", mode)
	case 'N':
		// TODO: Implement handshake options
		return m.processAtCommand(command[2:])
	case 'S':
		rNum, rNumProcessed, err := getNumber(command[1:])
		if err != nil {
			return fmt.Errorf("invalid register number: %s", err)
		}

		if command[1+rNumProcessed] != '=' {
			return fmt.Errorf("missing '='")
		}

		val, valProcessed, err := getNumber(command[1+rNumProcessed+1:])
		if err != nil {
			return fmt.Errorf("invalid value: %s", err)
		}

		switch rNum {
		case 0:
			m.register.ringToAnswerAfter = val
			return m.processAtCommand(command[1+rNumProcessed+1+valProcessed:])
		case 7:
			m.register.waitForCarrierSec = val
			return m.processAtCommand(command[1+rNumProcessed+1+valProcessed:])
		case 95:
			m.register.connectWithCarrierSpeed = val&1 == 1
			m.register.connectAppendARQ = val&2 == 2
			m.register.displayCarrier = val&4 == 4
			m.register.displayProtocol = val&8 == 8
			m.register.displayCompression = val&32 == 32
			return m.processAtCommand(command[1+rNumProcessed+1+valProcessed:])
		}
		return fmt.Errorf("unknown register number: %d", rNum)
	case 'V':
		val := command[1]
		if val == '0' {
			// TODO: Implement numeric result codes
			return fmt.Errorf("numeric result codes not implemented")
		} else {
			return m.processAtCommand(command[2:])
		}
	case 'X':
		val := command[1]

		switch val {
		// TODO: Is dialtone detection important
		case '0':
			m.register.connectWithCarrierSpeed = false
			return m.processAtCommand(command[2:])
		case '1':
			m.register.connectWithCarrierSpeed = true
			return m.processAtCommand(command[2:])
		case '2':
			m.register.connectWithCarrierSpeed = true
			return m.processAtCommand(command[2:])
		case '3':
			m.register.connectWithCarrierSpeed = true
			return m.processAtCommand(command[2:])
		case '4':
			m.register.connectWithCarrierSpeed = true
			return m.processAtCommand(command[2:])
		}

		return fmt.Errorf("invalid X value: %c", val)
	}

	return fmt.Errorf("unknown at command: %s", command)
}

func (m *Modem) processAtCommandExtended(command string) error {
	switch command[0] {
	case 'C':
		// TODO: Is this enough?
		return m.processAtCommand(command[2:])
	case 'D':
		_, processed, _ := getNumber(command[1:])
		return m.processAtCommand(command[1+processed:])
	case 'F':
		m.FactoryReset()
		return m.processAtCommand(command[1:])
	case 'K':
		// TODO: Implement flow control
		return m.processAtCommand(command[2:])
	}

	return fmt.Errorf("unknown extended at command: %s", command)
}

func (m *Modem) sendResponse(response string) {
	m.client.Write([]byte{m.register.endOfLine})
	m.client.Write([]byte{m.register.lineFeed})
	m.client.Write([]byte(response))
	m.client.Write([]byte{m.register.endOfLine})
	m.client.Write([]byte{m.register.lineFeed})

	fmt.Printf("Response: %s\n", response)
}
