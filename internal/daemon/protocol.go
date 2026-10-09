package daemon

import (
	"net"
	"strings"
)

// maxProtocolLine bounds control-line length so a bogus client cannot
// grow memory unbounded.
const maxProtocolLine = 512

// readLine reads a newline-terminated line byte by byte so no raw payload
// following the handshake line is consumed from the socket buffer.
func readLine(conn net.Conn) (string, error) {
	var b strings.Builder
	one := make([]byte, 1)
	for b.Len() < maxProtocolLine {
		_, err := conn.Read(one)
		if err != nil {
			return "", err
		}
		if one[0] == 10 {
			return b.String(), nil
		}
		if one[0] != 13 {
			b.WriteByte(one[0])
		}
	}
	return b.String(), nil
}
