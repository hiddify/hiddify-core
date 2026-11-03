package hcore

import (
	"bytes"
	"io"
	"testing"
)

func TestUploadBodySendsTheSizeFromOneBlock(t *testing.T) {
	block := uploadBlock()
	for _, size := range []int64{0, 1, int64(len(block)) - 1, int64(len(block)), 3*int64(len(block)) + 17} {
		sent, err := io.ReadAll(&uploadBody{left: size})
		if err != nil {
			t.Fatal(err)
		}
		if int64(len(sent)) != size {
			t.Fatalf("size %d: sent %d", size, len(sent))
		}
		for at := 0; at < len(sent); at += len(block) {
			end := min(at+len(block), len(sent))
			if !bytes.Equal(sent[at:end], block[:end-at]) {
				t.Fatalf("size %d: bytes at %d are not the block's", size, at)
			}
		}
	}
}
