package proxy

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"
)

const contentLengthHeader = "content-length:"

func readFrame(reader *bufio.Reader) ([]byte, error) {
	contentLength := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return nil, err
		}

		trimmed := strings.TrimRight(line, "\r\n")
		if trimmed == "" {
			break
		}

		lower := strings.ToLower(trimmed)
		if strings.HasPrefix(lower, contentLengthHeader) {
			value := strings.TrimSpace(trimmed[len(contentLengthHeader):])
			parsed, parseErr := strconv.Atoi(value)
			if parseErr != nil {
				return nil, fmt.Errorf("parse content length: %w", parseErr)
			}

			contentLength = parsed
		}
	}

	if contentLength <= 0 {
		return nil, fmt.Errorf("missing content length")
	}

	payload := make([]byte, contentLength)
	if _, err := io.ReadFull(reader, payload); err != nil {
		return nil, fmt.Errorf("read payload: %w", err)
	}

	return payload, nil
}

func writeFrame(writer io.Writer, payload []byte) error {
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(payload))
	if _, err := io.Copy(writer, bytes.NewBufferString(header)); err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	if _, err := writer.Write(payload); err != nil {
		return fmt.Errorf("write payload: %w", err)
	}

	return nil
}
