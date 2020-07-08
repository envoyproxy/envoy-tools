package test

import (
	"bytes"
	"envoy-tools/csds-client/client"
	envoy_service_status_v2 "github.com/envoyproxy/go-control-plane/envoy/service/status/v2"
	"google.golang.org/protobuf/encoding/protojson"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestParseNodeMatcher(t *testing.T) {
	c := client.Client{
		Info: client.Flag{
			RequestYaml: "./test_request.yaml",
		},
	}
	err := c.ParseNodeMatcher()
	if err != nil {
		t.Errorf("Parse NodeMatcher Error: %v", err)
	}
	if c.Nm == nil {
		t.Errorf("Parse NodeMatcher Failure!")
	}
	want := "node_id:{exact:\"fake_node_id\"} node_metadatas:{path:{key:\"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER\"} value:{string_match:{exact:\"fake_project_number\"}}} node_metadatas:{path:{key:\"TRAFFICDIRECTOR_NETWORK_NAME\"} value:{string_match:{exact:\"fake_network_name\"}}}"
	if c.Nm[0].String() != want {
		t.Errorf("NodeMatcher = %v, want: %v", c.Nm[0].String(), want)
	}
}

// Capture the std out for testing
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

func TestParseResponseWithoutNodeId(t *testing.T) {
	filename, _ := filepath.Abs("./response_without_nodeid_test.json")
	responsejson, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	var response envoy_service_status_v2.ClientStatusResponse
	err = protojson.Unmarshal(responsejson, &response)
	if err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	out := CaptureOutput(func() {
		client.ParseResponse(&response, "")
	})
	want := "Client ID                                          xDS stream type                Config                         \ntest_node_1                                        test_stream_type1              N/A                            \ntest_node_2                                        test_stream_type2              N/A                            \ntest_node_3                                        test_stream_type3              N/A                            \n"
	if out != want {
		t.Errorf("want\n%vout\n%v", want, out)
	}
}

func TestParseResponseWithNodeId(t *testing.T) {
	filename, _ := filepath.Abs("./response_with_nodeid_test.json")
	responsejson, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	var response envoy_service_status_v2.ClientStatusResponse
	err = protojson.Unmarshal(responsejson, &response)
	if err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	out := CaptureOutput(func() {
		client.ParseResponse(&response, "test_config.json")
	})
	want := "Client ID                                          xDS stream type                Config                         \nSTALE                                              test_stream_type1              test_config.json               \n"
	if out != want {
		t.Errorf("want\n%vout\n%v", want, out)
	}

	outfile, _ := filepath.Abs("./test_config.json")
	outputjson, err := ioutil.ReadFile(outfile)
	if err != nil {
		t.Errorf("Write config to file failure: %v", err)
	}
	if string(outputjson) != string(responsejson) {
		t.Errorf("Output formatted error")
	}
}
