package mockhelpers

import (
	"bytes"
	"errors"
	"io"
	"os/exec"
	"sync"
)

type MockPTY struct {
	PtsPath      string
	Cmd          *exec.Cmd
	OutputBuffer bytes.Buffer
	InputBuffer  bytes.Buffer
	ShouldFail   bool
	FailOnWrite  bool
	Closed       bool
	mu           sync.Mutex
}

func NewMockPTY() *MockPTY {
	return &MockPTY{
		OutputBuffer: bytes.Buffer{},
		InputBuffer:  bytes.Buffer{},
		ShouldFail:   false,
		FailOnWrite:  false,
		Closed:       false,
	}
}

func (m *MockPTY) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.ShouldFail {
		return errors.New("pty mock start failure")
	}
	m.PtsPath = "/dev/pts/mock"
	return nil
}

func (m *MockPTY) Write(data []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Closed {
		return 0, io.EOF
	}
	if m.FailOnWrite {
		return 0, errors.New("pty mock write failure")
	}
	return m.InputBuffer.Write(data)
}

func (m *MockPTY) Read(p []byte) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.Closed {
		return 0, io.EOF
	}
	return m.OutputBuffer.Read(p)
}

func (m *MockPTY) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Closed = true
	return nil
}

func (m *MockPTY) SetShouldFail(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ShouldFail = fail
}

func (m *MockPTY) SetFailOnWrite(fail bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.FailOnWrite = fail
}

func (m *MockPTY) SetOutputData(data string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.OutputBuffer.Reset()
	m.OutputBuffer.WriteString(data)
}

func (m *MockPTY) GetInputData() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.InputBuffer.String()
}
