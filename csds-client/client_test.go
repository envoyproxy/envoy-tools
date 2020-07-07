package main

import "testing"

func TestParseNodeMatcher(t *testing.T) {
	c := Client{
		info: Flag{
			requestYaml: "./test_request.yaml",
		},
	}
	err := c.parseNodeMatcher()
	if err != nil {
		t.Errorf("Parse NodeMatcher Error: %v", err)
	}
	if c.nm == nil {
		t.Errorf("Parse NodeMatcher Failure!")
	}
	want := "node_id:{exact:\"fake_node_id\"}  node_metadatas:{path:{key:\"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER\"}  value:{string_match:{exact:\"fake_project_number\"}}}  node_metadatas:{path:{key:\"TRAFFICDIRECTOR_NETWORK_NAME\"}  value:{string_match:{exact:\"fake_network_name\"}}}"
	if c.nm[0].String() != want {
		t.Errorf("NodeMatcher = %v, want: %v", c.nm[0].String(), want)
	}
}
