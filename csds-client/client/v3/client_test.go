// Unit Tests for client/v3
package client

import (
	"envoy-tools/csds-client/client"
	clientUtil "envoy-tools/csds-client/client/util"
	"io/ioutil"
	"path/filepath"
	"testing"
	"strings"

	csdspb_v3 "github.com/envoyproxy/go-control-plane/envoy/service/status/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"github.com/google/uuid"
)

// TestParseNodeMatcherWithFile tests parsing -request_file to nodematcher.
func TestParseNodeMatcherWithFile(t *testing.T) {
	c := ClientV3{
		opts: client.ClientOptions{
			Platform:    "gcp",
			RequestFile: "./test_request.yaml",
		},
	}
	if err := c.parseNodeMatcher(); err != nil {
		t.Errorf("Parse NodeMatcher Error: %v", err)
	}
	if c.nodeMatcher == nil {
		t.Errorf("Parse NodeMatcher Failure!")
	}
	want := "{\"nodeId\":{\"exact\":\"fake_node_id\"},\"nodeMetadatas\":[{\"path\":[{\"key\":\"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER\"}],\"value\":{\"stringMatch\":{\"exact\":\"fake_project_number\"}}},{\"path\":[{\"key\":\"TRAFFICDIRECTOR_NETWORK_NAME\"}],\"value\":{\"stringMatch\":{\"exact\":\"fake_network_name\"}}}]}"
	get, err := protojson.Marshal(c.nodeMatcher[0])
	if err != nil {
		t.Errorf("Parse NodeMatcher Error: %v", err)
	}

	if !clientUtil.ShouldEqualJSON(t, string(get), want) {
		t.Errorf("NodeMatcher = \n%v\n, want: \n%v\n", string(get), want)
	}

	wantNodeIdPrefix := "projects/fake_project_number/networks/fake_network_name/nodes"
	nodeSlice := strings.Split(c.node.Id, "/")
	gotNodeIdPrefix := strings.Join(nodeSlice[:len(nodeSlice)-1], "/")
	if wantNodeIdPrefix != gotNodeIdPrefix {
		t.Errorf("node.id prefix = %v, want = %v", gotNodeIdPrefix, wantNodeIdPrefix)
	}

	if _, err := uuid.Parse(nodeSlice[len(nodeSlice)-1]); err != nil {
		t.Errorf("node.id postfix isn't a valid UUID:  %v", err)
	}
}

// TestParseNodeMatcherWithString tests parsing -request_yaml to nodematcher.
func TestParseNodeMatcherWithString(t *testing.T) {
	c := ClientV3{
		opts: client.ClientOptions{
			Platform:    "gcp",
			RequestYaml: "{\"node_matchers\": [{\"node_id\": {\"exact\": \"fake_node_id\"}, \"node_metadatas\": [{\"path\": [{\"key\": \"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER\"}], \"value\": {\"string_match\": {\"exact\": \"fake_project_number\"}}}, {\"path\": [{\"key\": \"TRAFFICDIRECTOR_NETWORK_NAME\"}], \"value\": {\"string_match\": {\"exact\": \"fake_network_name\"}}}]}]}",
		},
	}
	err := c.parseNodeMatcher()
	if err != nil {
		t.Errorf("Parse NodeMatcher Error: %v", err)
	}
	if c.nodeMatcher == nil {
		t.Errorf("Parse NodeMatcher Failure!")
	}
	want := "{\"nodeId\":{\"exact\":\"fake_node_id\"}, \"nodeMetadatas\":[{\"path\":[{\"key\":\"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER\"}], \"value\":{\"stringMatch\":{\"exact\":\"fake_project_number\"}}}, {\"path\":[{\"key\":\"TRAFFICDIRECTOR_NETWORK_NAME\"}], \"value\":{\"stringMatch\":{\"exact\":\"fake_network_name\"}}}]}"
	get, err := protojson.Marshal(c.nodeMatcher[0])
	if err != nil {
		t.Errorf("Parse NodeMatcher Error: %v", err)
	}
	if !clientUtil.ShouldEqualJSON(t, string(get), want) {
		t.Errorf("NodeMatcher = \n%v\n, want: \n%v\n", string(get), want)
	}
}

// TestParseNodeMatcherWithFileAndString tests parsing -request_file and -request_yaml to nodematcher.
func TestParseNodeMatcherWithFileAndString(t *testing.T) {
	c := ClientV3{
		opts: client.ClientOptions{
			Platform:    "gcp",
			RequestFile: "./test_request.yaml",
			RequestYaml: "{\"node_matchers\": [{\"node_id\": {\"exact\": \"fake_node_id_from_cli\"}}]}",
		},
	}
	if err := c.parseNodeMatcher(); err != nil {
		t.Errorf("Parse NodeMatcher Error: %v", err)
	}
	if c.nodeMatcher == nil {
		t.Errorf("Parse NodeMatcher Failure!")
	}
	want := "{\"nodeId\":{\"exact\":\"fake_node_id_from_cli\"}, \"nodeMetadatas\":[{\"path\":[{\"key\":\"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER\"}], \"value\":{\"stringMatch\":{\"exact\":\"fake_project_number\"}}}, {\"path\":[{\"key\":\"TRAFFICDIRECTOR_NETWORK_NAME\"}], \"value\":{\"stringMatch\":{\"exact\":\"fake_network_name\"}}}]}"
	get, err := protojson.Marshal(c.nodeMatcher[0])
	if err != nil {
		t.Errorf("Parse NodeMatcher Error: %v", err)
	}
	if !clientUtil.ShouldEqualJSON(t, string(get), want) {
		t.Errorf("NodeMatcher = \n%v\n, want: \n%v\n", string(get), want)
	}
}

// TestParseResponseWithoutNodeId tests post processing response without node_id.
func TestParseResponseWithoutNodeId(t *testing.T) {
	c := ClientV3{
		opts: client.ClientOptions{
			Platform: "gcp",
		},
	}
	filename, _ := filepath.Abs("./response_without_nodeid_test.json")
	responsejson, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	var response csdspb_v3.ClientStatusResponse
	if err = protojson.Unmarshal(responsejson, &response); err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	out := clientUtil.CaptureOutput(func() {
		if err := printOutResponse(&response, c.opts); err != nil {
			t.Errorf("Print out response error: %v", err)
		}
	})
	want := `Client ID                                          xDS stream type                Config Status                  
test_node_1                                        test_stream_type1              N/A                            
test_node_2                                        test_stream_type2              N/A                            
test_node_3                                        test_stream_type3              N/A                            
`
	if out != want {
		t.Errorf("want\n%vout\n%v", want, out)
	}
}

// TestParseResponseWithNodeId tests post processing response with node_id
func TestParseResponseWithNodeId(t *testing.T) {
	c := ClientV3{
		opts: client.ClientOptions{
			Platform:   "gcp",
			ConfigFile: "test_config.json",
		},
	}
	filename, _ := filepath.Abs("./response_with_nodeid_test.json")
	responsejson, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	var response csdspb_v3.ClientStatusResponse
	if err = protojson.Unmarshal(responsejson, &response); err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	out := clientUtil.CaptureOutput(func() {
		if err := printOutResponse(&response, c.opts); err != nil {
			t.Errorf("Print out response error: %v", err)
		}
	})
	want := `Client ID                                          xDS stream type                Config Status                  
test_nodeid                                        test_stream_type1              RDS   STALE                    
                                                                                  CDS   STALE                    
Config has been saved to test_config.json
`
	if out != want {
		t.Errorf("want\n%vout\n%v", want, out)
	}

	outfile, _ := filepath.Abs("./test_config.json")
	outputjson, err := ioutil.ReadFile(outfile)
	if err != nil {
		t.Errorf("Write config to file failure: %v", err)
	}
	ok, err := clientUtil.EqualJSONBytes(outputjson, responsejson)
	if err != nil {
		t.Errorf("failed to parse json")
	}
	if !ok {
		t.Errorf("Output formatted error")
	}
}

// TestVisualization tests parsing xds relationship from config and generating .dot
func TestVisualization(t *testing.T) {
	filename, _ := filepath.Abs("./response_for_visualization.json")
	responsejson, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	if err := clientUtil.Visualize(responsejson, false); err != nil {
		t.Errorf("Visualization Failure: %v", err)
	}
	want := `digraph G {
rankdir=LR;
"test_lds_0"->"test_rds_0"[ arrowsize=0.3, penwidth=0.3 ];
"test_lds_0"->"test_rds_1"[ arrowsize=0.3, penwidth=0.3 ];
"test_rds_0"->"test_cds_0"[ arrowsize=0.3, penwidth=0.3 ];
"test_rds_0"->"test_cds_1"[ arrowsize=0.3, penwidth=0.3 ];
"test_rds_1"->"test_cds_1"[ arrowsize=0.3, penwidth=0.3 ];
"test_cds_0"->"EDS0"[ arrowsize=0.3, penwidth=0.3 ];
"EDS0" [ color="#34A853", fillcolor="#34A853", fontcolor=white, fontname=Roboto, label=EDS0, shape=box, style="filled,rounded" ];
"test_cds_0" [ color="#FBBC04", fillcolor="#FBBC04", fontcolor=white, fontname=Roboto, label=CDS0, shape=box, style="filled,rounded" ];
"test_cds_1" [ color="#FBBC04", fillcolor="#FBBC04", fontcolor=white, fontname=Roboto, label=CDS1, shape=box, style="filled,rounded" ];
"test_lds_0" [ color="#4285F4", fillcolor="#4285F4", fontcolor=white, fontname=Roboto, label=LDS0, shape=box, style="filled,rounded" ];
"test_rds_0" [ color="#EA4335", fillcolor="#EA4335", fontcolor=white, fontname=Roboto, label=RDS0, shape=box, style="filled,rounded" ];
"test_rds_1" [ color="#EA4335", fillcolor="#EA4335", fontcolor=white, fontname=Roboto, label=RDS1, shape=box, style="filled,rounded" ];

}
`
	if err := clientUtil.OpenBrowser("http://dreampuf.github.io/GraphvizOnline/#" + want); err != nil {
		t.Errorf("Open want graph failure: %v", err)
	}
}

// TestNodeIdPrefixFilter tests node_id prefix filter
func TestNodeIdPrefixFilter(t *testing.T) {
	c := ClientV3{
		opts: client.ClientOptions{
			Platform:      "gcp",
			FilterMode:    "prefix",
			FilterPattern: "test",
		},
	}
	filename, _ := filepath.Abs("./response_for_filter.json")
	responsejson, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	var response csdspb_v3.ClientStatusResponse
	if err = protojson.Unmarshal(responsejson, &response); err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	out := clientUtil.CaptureOutput(func() {
		if err := printOutResponse(&response, c.opts); err != nil {
			t.Errorf("Print out response error: %v", err)
		}
	})
	want := `Client ID                                          xDS stream type                Config Status                  
test_node_1                                        test_stream_type1              N/A                            
test_node_2                                        test_stream_type2              N/A                            
test_node_3                                        test_stream_type3              N/A                            
`
	if out != want {
		t.Errorf("want\n%vout\n%v", want, out)
	}
}

// TestNodeIdSuffixFilter tests node_id suffix filter
func TestNodeIdSuffixFilter(t *testing.T) {
	c := ClientV3{
		opts: client.ClientOptions{
			Platform:      "gcp",
			FilterMode:    "suffix",
			FilterPattern: "3",
		},
	}
	filename, _ := filepath.Abs("./response_for_filter.json")
	responsejson, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	var response csdspb_v3.ClientStatusResponse
	if err = protojson.Unmarshal(responsejson, &response); err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	out := clientUtil.CaptureOutput(func() {
		if err := printOutResponse(&response, c.opts); err != nil {
			t.Errorf("Print out response error: %v", err)
		}
	})
	want := `Client ID                                          xDS stream type                Config Status                  
test_node_3                                        test_stream_type3              N/A                            
node_3                                             test_stream_type4              N/A                            
`
	if out != want {
		t.Errorf("want\n%vout\n%v", want, out)
	}
}

// TestNodeIdRegexFilter tests node_id regex filter
func TestNodeIdRegexFilter(t *testing.T) {
	c := ClientV3{
		opts: client.ClientOptions{
			Platform:      "gcp",
			FilterMode:    "regex",
			FilterPattern: "test.*",
		},
	}
	filename, _ := filepath.Abs("./response_for_filter.json")
	responsejson, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	var response csdspb_v3.ClientStatusResponse
	if err = protojson.Unmarshal(responsejson, &response); err != nil {
		t.Errorf("Read From File Failure: %v", err)
	}
	out := clientUtil.CaptureOutput(func() {
		if err := printOutResponse(&response, c.opts); err != nil {
			t.Errorf("Print out response error: %v", err)
		}
	})
	want := `Client ID                                          xDS stream type                Config Status                  
test_node_1                                        test_stream_type1              N/A                            
test_node_2                                        test_stream_type2              N/A                            
test_node_3                                        test_stream_type3              N/A                            
`
	if out != want {
		t.Errorf("want\n%vout\n%v", want, out)
	}
}

func TestParseNodeMatcherMeshScopeKeyWithFile(t *testing.T) {
	c := ClientV3{
		opts: client.ClientOptions{
			Platform:    "gcp",
			RequestFile: "./test_request_mesh_scope.yaml",
		},
	}
	if err := c.parseNodeMatcher(); err != nil {
		t.Errorf("Parse NodeMatcher Error: %v", err)
	}
	if c.nodeMatcher == nil {
		t.Errorf("Parse NodeMatcher Failure!")
	}
	want := "{\"nodeId\":{\"exact\":\"fake_node_id\"},\"nodeMetadatas\":[{\"path\":[{\"key\":\"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER\"}],\"value\":{\"stringMatch\":{\"exact\":\"fake_project_number\"}}},{\"path\":[{\"key\":\"TRAFFICDIRECTOR_MESH_SCOPE_NAME\"}],\"value\":{\"stringMatch\":{\"exact\":\"fake_mesh_scope_name\"}}}]}"
	get, err := protojson.Marshal(c.nodeMatcher[0])
	if err != nil {
		t.Errorf("Parse NodeMatcher Error: %v", err)
	}

	if !clientUtil.ShouldEqualJSON(t, string(get), want) {
		t.Errorf("NodeMatcher = \n%v\n, want: \n%v\n", string(get), want)
	}
}

func TestParseNodeMatcherMeshScopeAndNetworkShouldFail(t *testing.T) {
	c := ClientV3{
		opts: client.ClientOptions{
			Platform:    "gcp",
			RequestFile: "./test_request_mesh_scope_network.yaml",
		},
	}
	if err := c.parseNodeMatcher(); err == nil {
		t.Errorf("Parse NodeMatcher should fail since network name and meshScope are provided.")
	}
}
