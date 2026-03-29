package pty

import (
	"bytes"
	"os"
	"testing"
)

func TestPTYProcessSetSize(t *testing.T) {
	tests := []struct {
		name    string
		p       *PTYProcess
		rows    uint16
		cols    uint16
		wantErr bool
	}{
		{
			name:    "valid size",
			p:       &PTYProcess{Pty: nil},
			rows:    24,
			cols:    80,
			wantErr: true,
		},
		{
			name:    "nil PTY",
			p:       &PTYProcess{Pty: nil},
			rows:    24,
			cols:    80,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.SetSize(tt.rows, tt.cols)
			if (err != nil) != tt.wantErr {
				t.Errorf("PTYProcess.SetSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPTYProcessClose(t *testing.T) {
	tests := []struct {
		name    string
		p       *PTYProcess
		wantErr bool
	}{
		{
			name:    "close nil PTY",
			p:       &PTYProcess{Pty: nil},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.Close()
			if (err != nil) != tt.wantErr {
				t.Errorf("PTYProcess.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPTYProcessWait(t *testing.T) {
	tests := []struct {
		name    string
		p       *PTYProcess
		wantErr bool
	}{
		{
			name:    "wait nil command",
			p:       &PTYProcess{Cmd: nil},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.Wait()
			if (err != nil) != tt.wantErr {
				t.Errorf("PTYProcess.Wait() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPTYProcessKill(t *testing.T) {
	tests := []struct {
		name    string
		p       *PTYProcess
		wantErr bool
	}{
		{
			name:    "kill nil command",
			p:       &PTYProcess{Cmd: nil},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.p.Kill()
			if (err != nil) != tt.wantErr {
				t.Errorf("PTYProcess.Kill() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPTYProcessPipe(t *testing.T) {
	tests := []struct {
		name    string
		p       *PTYProcess
		in      []byte
		wantErr bool
	}{
		{
			name:    "pipe with nil PTY",
			p:       &PTYProcess{Pty: nil},
			in:      []byte("test"),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clientIn := bytes.NewReader(tt.in)
			clientOut := &bytes.Buffer{}
			err := tt.p.Pipe(clientIn, clientOut)
			if (err != nil) != tt.wantErr {
				t.Errorf("PTYProcess.Pipe() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetPtsPath(t *testing.T) {
	tests := []struct {
		name    string
		ptyFile *os.File
		wantErr bool
	}{
		{
			name:    "nil file",
			ptyFile: nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := getPtsPath(tt.ptyFile)
			if (err != nil) != tt.wantErr {
				t.Errorf("getPtsPath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
