// Package client/v2 implements the client interface for v2 transport api version
package client

import (
	"context"
	"crypto/x509"
	"encoding/json"
	"envoy-tools/csds-client/client"
	clientUtil "envoy-tools/csds-client/client/util"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"path/filepath"
	"strings"
	"time"

	csdspb_v2 "github.com/envoyproxy/go-control-plane/envoy/service/status/v2"
	envoy_type_matcher_v2 "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"github.com/ghodss/yaml"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// ClientV2 implements the Client interface
type ClientV2 struct {
	clientConn *grpc.ClientConn
	csdsClient csdspb_v2.ClientStatusDiscoveryServiceClient

	nodeMatcher []*envoy_type_matcher_v2.NodeMatcher
	metadata    metadata.MD
	opts        client.ClientOptions
}

// parseNodeMatcher parses the csds request yaml from -request_file and -request_yaml to nodematcher
// if -request_file and -request_yaml are both set, the values in this yaml string will override and
// merge with the request loaded from -request_file
func (c *ClientV2) parseNodeMatcher() error {
	if c.opts.RequestFile == "" && c.opts.RequestYaml == "" {
		return errors.New("missing request yaml")
	}

	var nodematchers []*envoy_type_matcher_v2.NodeMatcher
	if err := parseYaml(c.opts.RequestFile, c.opts.RequestYaml, &nodematchers); err != nil {
		return err
	}

	c.nodeMatcher = nodematchers

	// check if required fields exist in nodematcher
	switch c.opts.Platform {
	case "gcp":
		keys := []string{"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER", "TRAFFICDIRECTOR_NETWORK_NAME"}
		for _, key := range keys {
			if value := getValueByKeyFromNodeMatcher(c.nodeMatcher, key); value == "" {
				return fmt.Errorf("missing field %v in NodeMatcher", key)
			}
		}
	default:
		return fmt.Errorf("%s platform is not supported, list of supported platforms: gcp", c.opts.Platform)
	}

	return nil
}

// connWithAuth connects to uri with authentication
func (c *ClientV2) connWithAuth() error {
	var scope string
	switch c.opts.AuthnMode {
	case "jwt":
		if c.opts.Jwt == "" {
			return errors.New("missing jwt file")
		}
		switch c.opts.Platform {
		case "gcp":
			scope = "https://www.googleapis.com/auth/cloud-platform"
			pool, err := x509.SystemCertPool()
			if err != nil {
				return err
			}
			creds := credentials.NewClientTLSFromCert(pool, "")
			perRPC, err := oauth.NewServiceAccountFromFile(c.opts.Jwt, scope)
			if err != nil {
				return err
			}

			c.clientConn, err = grpc.Dial(c.opts.Uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
			if err != nil {
				return err
			}
			return nil
		default:
			return fmt.Errorf("%s platform is not supported, list of supported platforms: gcp", c.opts.Platform)
		}
	case "auto":
		switch c.opts.Platform {
		case "gcp":
			scope = "https://www.googleapis.com/auth/cloud-platform"
			pool, err := x509.SystemCertPool()
			if err != nil {
				return err
			}
			creds := credentials.NewClientTLSFromCert(pool, "")
			perRPC, err := oauth.NewApplicationDefault(context.Background(), scope) // Application Default Credentials (ADC)
			if err != nil {
				return err
			}

			// parse GCP project number as header for authentication
			var key string
			switch c.opts.Uri {
			case "trafficdirector.googleapis.com:443":
				key = "TRAFFICDIRECTOR_GCP_PROJECT_NUMBER"
			}
			if projectNum := getValueByKeyFromNodeMatcher(c.nodeMatcher, key); projectNum != "" {
				c.metadata = metadata.Pairs("x-goog-user-project", projectNum)
			}

			c.clientConn, err = grpc.Dial(c.opts.Uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
			if err != nil {
				return err
			}
			return nil
		default:
			return errors.New("auto authentication mode for this platform is not supported. Please use jwt_file instead")
		}
	default:
		return errors.New("invalid authn_mode")
	}
}

// New creates a new client with v2 api version
func New(option client.ClientOptions) (*ClientV2, error) {
	c := &ClientV2{
		opts: option,
	}
	if c.opts.Platform != "gcp" {
		return nil, fmt.Errorf("%s platform is not supported, list of supported platforms: gcp", c.opts.Platform)
	}

	if err := c.parseNodeMatcher(); err != nil {
		return nil, err
	}

	return c, nil
}

// Run connects the client to the uri and calls doRequest
func (c *ClientV2) Run() error {
	if err := c.connWithAuth(); err != nil {
		return err
	}
	defer c.clientConn.Close()

	c.csdsClient = csdspb_v2.NewClientStatusDiscoveryServiceClient(c.clientConn)
	var ctx context.Context
	if c.metadata != nil {
		ctx = metadata.NewOutgoingContext(context.Background(), c.metadata)
	} else {
		ctx = context.Background()
	}

	streamClientStatus, err := c.csdsClient.StreamClientStatus(ctx)
	if err != nil {
		return err
	}

	// run once or run with monitor mode
	for {
		if err := c.doRequest(streamClientStatus); err != nil {
			// timeout error
			// retry to connect
			if strings.Contains(err.Error(), "RpcSecurityPolicy") {
				streamClientStatus, err = c.csdsClient.StreamClientStatus(ctx)
				if err != nil {
					return err
				}
				continue
			} else {
				return err
			}
		}
		if c.opts.MonitorInterval != 0 {
			time.Sleep(c.opts.MonitorInterval)
		} else {
			if err = streamClientStatus.CloseSend(); err != nil {
				return err
			}
			return nil
		}
	}
}

// doRequest sends request and prints out the parsed response
func (c *ClientV2) doRequest(streamClientStatus csdspb_v2.ClientStatusDiscoveryService_StreamClientStatusClient) error {

	req := &csdspb_v2.ClientStatusRequest{NodeMatchers: c.nodeMatcher}
	if err := streamClientStatus.Send(req); err != nil {
		return err
	}

	resp, err := streamClientStatus.Recv()
	if err != nil && err != io.EOF {
		return err
	}
	// post process response
	if err := printOutResponse(resp, c.opts); err != nil {
		return err
	}

	return nil
}

// parseConfigStatus parses each xds config status to string
func parseConfigStatus(xdsConfig []*csdspb_v2.PerXdsConfig) []string {
	var configStatus []string
	for _, perXdsConfig := range xdsConfig {
		status := perXdsConfig.GetStatus().String()
		var xds string
		if perXdsConfig.GetClusterConfig() != nil {
			xds = "CDS"
		} else if perXdsConfig.GetListenerConfig() != nil {
			xds = "LDS"
		} else if perXdsConfig.GetRouteConfig() != nil {
			xds = "RDS"
		} else if perXdsConfig.GetScopedRouteConfig() != nil {
			xds = "SRDS"
		}
		if status != "" && xds != "" {
			configStatus = append(configStatus, xds+"   "+status)
		}
	}
	return configStatus
}

// printOutResponse processes response and print
func printOutResponse(response *csdspb_v2.ClientStatusResponse, opts client.ClientOptions) error {
	if response.GetConfig() == nil || len(response.GetConfig()) == 0 {
		fmt.Printf("No xDS clients connected.\n")
		return nil
	} else {
		fmt.Printf("%-50s %-30s %-30s \n", "Client ID", "xDS stream type", "Config Status")
	}

	var hasXdsConfig bool

	for _, config := range response.GetConfig() {
		var id string
		var xdsType string
		if config.GetNode() != nil {
			id = config.GetNode().GetId()
			metadata := config.GetNode().GetMetadata().AsMap()

			// control plane is expected to use "XDS_STREAM_TYPE" to communicate
			// the stream type of the connected client in the response.
			if metadata["XDS_STREAM_TYPE"] != nil {
				xdsType = metadata["XDS_STREAM_TYPE"].(string)
			}
		}

		if config.GetXdsConfig() == nil {
			if config.GetNode() != nil {
				fmt.Printf("%-50s %-30s %-30s \n", id, xdsType, "N/A")
			}
		} else {
			hasXdsConfig = true

			// parse config status
			configStatus := parseConfigStatus(config.GetXdsConfig())
			fmt.Printf("%-50s %-30s ", id, xdsType)

			for i := 0; i < len(configStatus); i++ {
				if i == 0 {
					fmt.Printf("%-30s \n", configStatus[i])
				} else {
					fmt.Printf("%-50s %-30s %-30s \n", "", "", configStatus[i])
				}
			}
			if len(configStatus) == 0 {
				fmt.Printf("\n")
			}
		}
	}

	if hasXdsConfig {
		if err := clientUtil.PrintDetailedConfig(response, opts); err != nil {
			return err
		}
	}
	return nil
}

// parseYaml is a helper method for parsing csds request yaml to nodematchers
func parseYaml(path string, yamlStr string, nms *[]*envoy_type_matcher_v2.NodeMatcher) error {
	if path != "" {
		// parse yaml to json
		filename, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		yamlFile, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		js, err := yaml.YAMLToJSON(yamlFile)
		if err != nil {
			return err
		}

		// parse the json array to a map to iterate it
		var data map[string]interface{}
		if err = json.Unmarshal(js, &data); err != nil {
			return err
		}

		// parse each json object to proto
		for _, n := range data["node_matchers"].([]interface{}) {
			x := &envoy_type_matcher_v2.NodeMatcher{}

			jsonString, err := json.Marshal(n)
			if err != nil {
				return err
			}
			if err = protojson.Unmarshal(jsonString, x); err != nil {
				return err
			}
			*nms = append(*nms, x)
		}
	}
	if yamlStr != "" {
		var js []byte
		var err error
		// json input
		if clientUtil.IsJson(yamlStr) {
			js = []byte(yamlStr)
		} else {
			// parse the yaml input into json
			js, err = yaml.YAMLToJSON([]byte(yamlStr))
			if err != nil {
				return err
			}
		}

		// parse the json array to a map to iterate it
		var data map[string]interface{}
		if err = json.Unmarshal(js, &data); err != nil {
			return err
		}

		// parse each json object to proto
		for i, n := range data["node_matchers"].([]interface{}) {
			x := &envoy_type_matcher_v2.NodeMatcher{}

			jsonString, err := json.Marshal(n)
			if err != nil {
				return err
			}
			if err = protojson.Unmarshal(jsonString, x); err != nil {
				return err
			}

			// merge the proto with existing proto from request_file
			if i < len(*nms) {
				proto.Merge((*nms)[i], x)
			} else {
				*nms = append(*nms, x)
			}
		}
	}
	return nil
}

// getValueByKeyFromNodeMatcher gets the first value by key from the metadata of a set of NodeMatchers
func getValueByKeyFromNodeMatcher(nms []*envoy_type_matcher_v2.NodeMatcher, key string) string {
	for _, nm := range nms {
		for _, mt := range nm.NodeMetadatas {
			for _, path := range mt.Path {
				if path.GetKey() == key {
					return mt.Value.GetStringMatch().GetExact()
				}
			}
		}
	}
	return ""
}
