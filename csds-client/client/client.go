package client

import (
	csdspb "github.com/envoyproxy/go-control-plane/envoy/service/status/v2"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"

	"context"
	"crypto/x509"
	"errors"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
	"io"
	"time"
)

type Flag struct {
	uri             string
	platform        string
	authnMode       string
	apiVersion      string
	requestFile     string
	requestYaml     string
	jwt             string
	configFile      string
	monitorInterval time.Duration
	visualization   bool
}

type Client struct {
	clientConn *grpc.ClientConn
	csdsClient csdspb.ClientStatusDiscoveryServiceClient

	nodeMatcher []*envoy_type_matcher.NodeMatcher
	metadata    metadata.MD
	info        Flag
}

// ParseFlags parses flags to info
func ParseFlags() Flag {
	uriPtr := flag.String("service_uri", "trafficdirector.googleapis.com:443", "the uri of the service to connect to")
	platformPtr := flag.String("cloud_platform", "gcp", "the cloud platform (e.g. gcp, aws,  ...)")
	authnModePtr := flag.String("authn_mode", "auto", "the method to use for authentication (e.g. auto, jwt, ...)")
	apiVersionPtr := flag.String("api_version", "v2", "which xds api major version  to use (e.g. v2, v3 ...)")
	requestFilePtr := flag.String("request_file", "", "yaml file that defines the csds request")
	requestYamlPtr := flag.String("request_yaml", "", "yaml string that defines the csds request")
	jwtPtr := flag.String("jwt_file", "", "path of the -jwt_file")
	configFilePtr := flag.String("output_file", "", "file name to save configs returned by csds response")
	monitorIntervalPtr := flag.Duration("monitor_interval", 0, "the interval of sending request in monitor mode (e.g. 500ms, 2s, 1m ...)")
	visualizationPtr := flag.Bool("visualization", false, "option to visualize the relationship between xDS")

	flag.Parse()

	f := Flag{
		uri:             *uriPtr,
		platform:        *platformPtr,
		authnMode:       *authnModePtr,
		apiVersion:      *apiVersionPtr,
		requestFile:     *requestFilePtr,
		requestYaml:     *requestYamlPtr,
		jwt:             *jwtPtr,
		configFile:      *configFilePtr,
		monitorInterval: *monitorIntervalPtr,
		visualization:   *visualizationPtr,
	}

	return f
}

// parseNodeMatcher parses the csds request yaml from -request_file and -request_yaml to nodematcher
// if -request_file and -request_yaml are both set, the values in this yaml string will override and
// merge with the request loaded from -request_file
func (c *Client) parseNodeMatcher() error {
	if c.info.requestFile == "" && c.info.requestYaml == "" {
		return errors.New("missing request yaml")
	}

	var nodematchers []*envoy_type_matcher.NodeMatcher
	if err := parseYaml(c.info.requestFile, c.info.requestYaml, &nodematchers); err != nil {
		return err
	}

	c.nodeMatcher = nodematchers

	// check if required fields exist in nodematcher
	switch c.info.platform {
	case "gcp":
		keys := []string{"TRAFFICDIRECTOR_GCP_PROJECT_NUMBER", "TRAFFICDIRECTOR_NETWORK_NAME"}
		for _, key := range keys {
			if value := getValueByKeyFromNodeMatcher(c.nodeMatcher, key); value == "" {
				return fmt.Errorf("missing field %v in NodeMatcher", key)
			}
		}
	default:
		return fmt.Errorf("%s platform is not supported, list of supported platforms: gcp", c.info.platform)
	}

	return nil
}

// connWithAuth connects to uri with authentication
func (c *Client) connWithAuth() error {
	var scope string
	switch c.info.authnMode {
	case "jwt":
		if c.info.jwt == "" {
			return errors.New("missing jwt file")
		}
		switch c.info.platform {
		case "gcp":
			scope = "https://www.googleapis.com/auth/cloud-platform"
			pool, err := x509.SystemCertPool()
			if err != nil {
				return err
			}

			creds := credentials.NewClientTLSFromCert(pool, "")
			perRPC, err := oauth.NewServiceAccountFromFile(c.info.jwt, scope)
			if err != nil {
				return err
			}

			c.clientConn, err = grpc.Dial(c.info.uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
			if err != nil {
				return err
			}
			return nil
		default:
			return fmt.Errorf("%s platform is not supported, list of supported platforms: gcp", c.info.platform)
		}
	case "auto":
		switch c.info.platform {
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
			switch c.info.uri {
			case "trafficdirector.googleapis.com:443":
				key = "TRAFFICDIRECTOR_GCP_PROJECT_NUMBER"
			}

			if projectNum := getValueByKeyFromNodeMatcher(c.nodeMatcher, key); projectNum != "" {
				c.metadata = metadata.Pairs("x-goog-user-project", projectNum)
			}

			c.clientConn, err = grpc.Dial(c.info.uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
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
func New() (*Client, error) {
	c := &Client{
		info: ParseFlags(),
	}
	if c.info.platform != "gcp" {
		return nil, fmt.Errorf("%s platform is not supported, list of supported platforms: gcp", c.info.platform)
	}
	if c.info.apiVersion != "v2" {
		return nil, fmt.Errorf("%s api version is not supported, list of supported api versions: v2", c.info.apiVersion)
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

	c.csdsClient = csdspb.NewClientStatusDiscoveryServiceClient(c.clientConn)
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
			return err
		}
		if c.info.monitorInterval != 0 {
			time.Sleep(c.info.monitorInterval)
		} else {
			if err := streamClientStatus.CloseSend(); err != nil {
				return err
			}
			return nil
		}
	}
}

// doRequest sends request and print out the parsed response
func (c *Client) doRequest(streamClientStatus csdspb.ClientStatusDiscoveryService_StreamClientStatusClient) error {

	req := &csdspb.ClientStatusRequest{NodeMatchers: c.nodeMatcher}
	if err := streamClientStatus.Send(req); err != nil {
		return err
	}

	resp, err := streamClientStatus.Recv()
	if err != nil && err != io.EOF {
		return err
	}

	// post process response
	if err := printOutResponse(resp, c.info.configFile, c.info.visualization, c.info.monitorInterval != 0); err != nil {
		return err
	}

	return nil
}
