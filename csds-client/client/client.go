package client

import (
	csdspb_v2 "github.com/envoyproxy/go-control-plane/envoy/service/status/v2"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"

	"context"
	"crypto/x509"
	"errors"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
	"time"
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
	cc         *grpc.ClientConn
	csdsClient csdspb_v2.ClientStatusDiscoveryServiceClient

	nm   []*envoy_type_matcher.NodeMatcher
	md   metadata.MD
	info ClientOptions
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

	c.nm = nodematchers

	// check if required fields exist in nodematcher
	switch c.info.Platform {
	case "gcp":
		keys := []string{"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER", "TRAFFICDIRECTOR_NETWORK_NAME"}
		for _, key := range keys {
			if value := getValueByKeyFromNodeMatcher(c.nm, key); value == "" {
				return fmt.Errorf("missing field %v in NodeMatcher", key)
			}
		}
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
			creds := credentials.NewClientTLSFromCert(pool, "")
			perRPC, err := oauth.NewServiceAccountFromFile(c.info.Jwt, scope)
			if err != nil {
				return err
			}

			c.cc, err = grpc.Dial(c.info.Uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
			if err != nil {
				return err
			}
			return nil
		default:
			return nil
		}
	case "auto":
		switch c.info.Platform {
		case "gcp":
			scope = "https://www.googleapis.com/auth/cloud-platform"
			pool, err := x509.SystemCertPool()
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
			if projectNum := getValueByKeyFromNodeMatcher(c.nm, key); projectNum != "" {
				c.md = metadata.Pairs("x-goog-user-project", projectNum)
			}

			c.cc, err = grpc.Dial(c.info.Uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
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
	defer c.cc.Close()

	c.csdsClient = csdspb_v2.NewClientStatusDiscoveryServiceClient(c.cc)
	var ctx context.Context
	if c.md != nil {
		ctx = metadata.NewOutgoingContext(context.Background(), c.md)
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
			return err
		}
		if c.info.MonitorInterval != 0 {
			time.Sleep(c.info.MonitorInterval)
		} else {
			return nil
		}
	}
}

// doRequest sends request and print out the parsed response
func (c *Client) doRequest(streamClientStatus csdspb_v2.ClientStatusDiscoveryService_StreamClientStatusClient) error {

	req := &csdspb_v2.ClientStatusRequest{NodeMatchers: c.nm}
	if err := streamClientStatus.Send(req); err != nil {
		return err
	}

	resp, err := streamClientStatus.Recv()
	if err != nil {
		return err
	}

	// post process response
	if err := printOutResponse(resp, c.info.ConfigFile, c.info.Visualization); err != nil {
		return err
	}

	return nil
}
