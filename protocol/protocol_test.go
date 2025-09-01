package protocol

import (
	"bufio"
	"bytes"
	"testing"
)

func TestSerializeDeserialize(t *testing.T) {
	original := []byte("hello test")

	serialized := SerializeMessage(original)

	reader := bufio.NewReader(bytes.NewReader(serialized))

	deserialized, err := DeserializeMessage(reader)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if string(deserialized) != string(original) {
		t.Errorf("expected %q, got %q", original, deserialized)
	}
}

func TestDeserializeWrongVersion(t *testing.T) {
	data := []byte{99, REQUEST}
	data = append(data, []byte("5\nhello")...)

	reader := bufio.NewReader(bytes.NewReader(data))

	_, err := DeserializeMessage(reader)
	if err == nil {
		t.Fatal("expected error for unsupported version, got nil")
	}
}
