package proxy

import (
	"bufio"
	"bytes"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("frame codec", func() {
	It("writes then reads a content-length framed payload", func() {
		buffer := bytes.NewBuffer(nil)
		payload := []byte(`{"jsonrpc":"2.0","id":1}`)

		err := writeFrame(buffer, payload)
		Expect(err).NotTo(HaveOccurred())

		decoded, err := readFrame(bufio.NewReader(bytes.NewReader(buffer.Bytes())))
		Expect(err).NotTo(HaveOccurred())
		Expect(decoded).To(Equal(payload))
	})

	It("returns an error for missing content length", func() {
		_, err := readFrame(bufio.NewReader(bytes.NewBufferString("\r\n{}")))
		Expect(err).To(HaveOccurred())
	})
})
