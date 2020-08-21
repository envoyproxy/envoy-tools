package util

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"reflect"
	"sync"
	"testing"
)

// CaptureOutput captures the stdout for testing.
func CaptureOutput(f func()) string {
	reader, writer, err := os.Pipe()
	if err != nil {
		panic(err)
	}
	stdout := os.Stdout
	stderr := os.Stderr
	defer func() {
		os.Stdout = stdout
		os.Stderr = stderr
	}()
	os.Stdout = writer
	os.Stderr = writer
	out := make(chan string)
	wg := new(sync.WaitGroup)
	wg.Add(1)
	go func() {
		var buf bytes.Buffer
		wg.Done()
		io.Copy(&buf, reader)
		out <- buf.String()
	}()
	wg.Wait()
	f()
	writer.Close()
	return <-out
}

// ShouldEqualJSON tests if json string s1 and s2 are equal
func ShouldEqualJSON(t *testing.T, s1, s2 string) bool {
	t.Helper()

	verdict, err := EqualJSONBytes([]byte(s1), []byte(s2))
	if err != nil {
		t.Errorf("failed to check since: %w", err)
		return false
	}

	return verdict
}

// EqualJSONBytes compares json bytes s1 and s2
func EqualJSONBytes(s1, s2 []byte) (bool, error) {
	var o1 interface{}
	var o2 interface{}

	var err error
	err = json.Unmarshal(s1, &o1)
	if err != nil {
		return false, fmt.Errorf("failed to marshal s1: %w", err)
	}
	err = json.Unmarshal(s2, &o2)
	if err != nil {
		return false, fmt.Errorf("failed to marshal s2: %w", err)
	}

	return reflect.DeepEqual(o1, o2), nil
}
