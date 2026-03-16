package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strings"
)

func readNewlineFrame(reader *bufio.Reader) ([]byte, error) {
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return nil, err
		}

		trimmed := bytes.TrimSpace(line)
		if len(trimmed) == 0 {
			continue
		}

		return trimmed, nil
	}
}

func writeNewlineFrame(writer io.Writer, payload []byte) error {
	if _, err := writer.Write(payload); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	if !strings.HasSuffix(string(payload), "\n") {
		if _, err := writer.Write([]byte("\n")); err != nil {
			return fmt.Errorf("write newline: %w", err)
		}
	}

	return nil
}
