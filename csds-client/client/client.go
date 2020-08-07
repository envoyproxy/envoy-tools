package client

import (
	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	csdspb_v2 "github.com/envoyproxy/go-control-plane/envoy/service/status/v2"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
)

type ClientOptions struct {
	Uri             string
	Platform        string
	AuthnMode       string
	ApiVersion      string
	RequestFile     string
	RequestYaml     string
	Jwt             string
	ConfigFile      string
	MonitorInterval time.Duration
	Visualization   bool
}

type Client struct {
	clientConn *grpc.ClientConn
	csdsClient interface{}

	nodeMatcher []*envoy_type_matcher.NodeMatcher
	metadata    metadata.MD
	info        ClientOptions
}

// parseNodeMatcher parses the csds request yaml from -request_file and -request_yaml to nodematcher
// if -request_file and -request_yaml are both set, the values in this yaml string will override and
// merge with the request loaded from -request_file
func (c *Client) parseNodeMatcher() error {
	if c.info.RequestFile == "" && c.info.RequestYaml == "" {
		return errors.New("missing request yaml")
	}

	var nodematchers []*envoy_type_matcher.NodeMatcher
	if err := parseYaml(c.info.RequestFile, c.info.RequestYaml, &nodematchers); err != nil {
		return err
	}

	c.nodeMatcher = nodematchers

	// check if required fields exist in nodematcher
	switch c.info.Platform {
	case "gcp":
		keys := []string{"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER", "TRAFFICDIRECTOR_NETWORK_NAME"}
		for _, key := range keys {
			if value := getValueByKeyFromNodeMatcher(c.nodeMatcher, key); value == "" {
				return fmt.Errorf("missing field %v in NodeMatcher", key)
			}
		}
	default:
		return fmt.Errorf("%s platform is not supported, list of supported platforms: gcp", c.info.Platform)
	}

	return nil
}

// connWithAuth connects to uri with authentication
func (c *Client) connWithAuth() error {
	var scope string
	switch c.info.AuthnMode {
	case "jwt":
		if c.info.Jwt == "" {
			return errors.New("missing jwt file")
		}
		switch c.info.Platform {
		case "gcp":
			scope = "https://www.googleapis.com/auth/cloud-platform"
			pool, err := x509.SystemCertPool()
			if err != nil {
				return err
			}
			creds := credentials.NewClientTLSFromCert(pool, "")
			perRPC, err := oauth.NewServiceAccountFromFile(c.info.Jwt, scope)
			if err != nil {
				return err
			}

			c.clientConn, err = grpc.Dial(c.info.Uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
			if err != nil {
				return err
			}
			return nil
		default:
			return fmt.Errorf("%s platform is not supported, list of supported platforms: gcp", c.info.Platform)
		}
	case "auto":
		switch c.info.Platform {
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
			switch c.info.Uri {
			case "trafficdirector.googleapis.com:443":
				key = "TRAFFICDIRECTOR_GCP_PROJECT_NUMBER"
			}
			if projectNum := getValueByKeyFromNodeMatcher(c.nodeMatcher, key); projectNum != "" {
				c.metadata = metadata.Pairs("x-goog-user-project", projectNum)
			}

			c.clientConn, err = grpc.Dial(c.info.Uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
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

// New creates a new client
func New(option ClientOptions) (*Client, error) {
	c := &Client{
		info: option,
	}
	if c.info.Platform != "gcp" {
		return nil, fmt.Errorf("%s platform is not supported, list of supported platforms: gcp", c.info.Platform)
	}
	if c.info.ApiVersion != "v2" {
		return nil, fmt.Errorf("%s api version is not supported, list of supported api versions: v2", c.info.ApiVersion)
	}

	if err := c.parseNodeMatcher(); err != nil {
		return nil, err
	}

	return c, nil
}

// Run connects the client to the uri and calls doRequest
func (c *Client) Run() error {
	if err := c.connWithAuth(); err != nil {
		return err
	}
	defer c.clientConn.Close()

	var ctx context.Context
	if c.metadata != nil {
		ctx = metadata.NewOutgoingContext(context.Background(), c.metadata)
	} else {
		ctx = context.Background()
	}

	var streamClientStatus interface{}
	var err error
	switch c.info.ApiVersion {
	case "v2":
		c.csdsClient = csdspb_v2.NewClientStatusDiscoveryServiceClient(c.clientConn)
		streamClientStatus, err = c.csdsClient.(csdspb_v2.ClientStatusDiscoveryServiceClient).StreamClientStatus(ctx)
		if err != nil {
			return err
		}
	}

	// run once or run with monitor mode
	for {
		if err := c.doRequest(streamClientStatus); err != nil {
			// timeout error
			// retry to connect
			if strings.Contains(err.Error(), "RpcSecurityPolicy") {
				switch c.info.ApiVersion {
				case "v2":
					c.csdsClient = csdspb_v2.NewClientStatusDiscoveryServiceClient(c.clientConn)
					streamClientStatus, err = c.csdsClient.(csdspb_v2.ClientStatusDiscoveryServiceClient).StreamClientStatus(ctx)
					if err != nil {
						return err
					}
				}
				continue
			} else {
				return err
			}
		}
		if c.info.MonitorInterval != 0 {
			time.Sleep(c.info.MonitorInterval)
		} else {
			var err error
			switch c.info.ApiVersion {
			case "v2":
				if err = streamClientStatus.(csdspb_v2.ClientStatusDiscoveryService_StreamClientStatusClient).CloseSend(); err != nil {
					return err
				}
			}
			return err
		}
	}
}

// doRequest sends request and print out the parsed response
func (c *Client) doRequest(streamClientStatus interface{}) error {
	switch streamClientStatus.(type) {
	case csdspb_v2.ClientStatusDiscoveryService_StreamClientStatusClient:
		req := &csdspb_v2.ClientStatusRequest{NodeMatchers: c.nodeMatcher}
		streamclientstatusV2 := streamClientStatus.(csdspb_v2.ClientStatusDiscoveryService_StreamClientStatusClient)
		if err := streamclientstatusV2.Send(req); err != nil {
			return err
		}

		resp, err := streamclientstatusV2.Recv()
		if err != nil && err != io.EOF {
			return err
		}
		// post process response
		if err := printOutResponse_v2(resp, c.info); err != nil {
			return err
		}
	}
	return nil
}
