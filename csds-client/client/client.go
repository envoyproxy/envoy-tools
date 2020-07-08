package client

import (
	"context"
	"crypto/x509"
	"flag"
	"fmt"
	csdspb "github.com/envoyproxy/go-control-plane/envoy/service/status/v2"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
)

type Flag struct {
	uri         string
	platform    string
	authnMode   string
	apiVersion  string
	RequestYaml string
	jwt         string
	configFile  string
}

type Client struct {
	Cc         *grpc.ClientConn
	CsdsClient csdspb.ClientStatusDiscoveryServiceClient

	Nm   []*envoy_type_matcher.NodeMatcher
	Md   metadata.MD
	Info Flag
}

func ParseFlags() Flag {
	uriPtr := flag.String("service_uri", "trafficdirector.googleapis.com:443", "the uri of the service to connect to")
	platformPtr := flag.String("cloud_platform", "gcp", "the cloud platform (e.g. gcp, aws,  ...)")
	authnModePtr := flag.String("authn_mode", "auto", "the method to use for authentication (e.g. auto, jwt, ...)")
	apiVersionPtr := flag.String("api_version", "v2", "which xds api major version  to use (e.g. v2, v3 ...)")
	requestYamlPtr := flag.String("csds_request_yaml", "", "yaml file that defines the csds request")
	jwtPtr := flag.String("jwt_file", "", "path of the -jwt_file")
	configFilePtr := flag.String("file_to_save_config", "", "the file name to save config")

	flag.Parse()

	f := Flag{
		uri:         *uriPtr,
		platform:    *platformPtr,
		authnMode:   *authnModePtr,
		apiVersion:  *apiVersionPtr,
		RequestYaml: *requestYamlPtr,
		jwt:         *jwtPtr,
		configFile:  *configFilePtr,
	}

	return f
}

func (c *Client) ParseNodeMatcher() error {
	if c.Info.RequestYaml == "" {
		return fmt.Errorf("missing request yaml")
	}

	var nodematchers []*envoy_type_matcher.NodeMatcher
	err := ParseYaml(c.Info.RequestYaml, &nodematchers)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	c.Nm = nodematchers
	return nil
}

func (c *Client) ConnWithAuth() error {
	scope := "https://www.googleapis.com/auth/cloud-platform"
	if c.Info.authnMode == "jwt" {
		if c.Info.jwt == "" {
			return fmt.Errorf("missing jwt file")
		} else {
			pool, err := x509.SystemCertPool()
			creds := credentials.NewClientTLSFromCert(pool, "")
			perRPC, err := oauth.NewServiceAccountFromFile(c.Info.jwt, scope) //"/usr/local/google/home/yutongli/service_account_key.json"
			if err != nil {
				return fmt.Errorf("%v", err)
			}

			c.Cc, err = grpc.Dial(c.Info.uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
			if err != nil {
				return fmt.Errorf("%v", err)
			} else {
				return nil
			}
		}
	} else if c.Info.authnMode == "auto" {
		if c.Info.platform == "gcp" {
			pool, err := x509.SystemCertPool()
			creds := credentials.NewClientTLSFromCert(pool, "")
			perRPC, err := oauth.NewApplicationDefault(context.Background(), scope) // Application Default Credentials (ADC)
			if err != nil {
				return fmt.Errorf("%v", err)
			}

			// parse GCP project number as header for authentication
			if projectNum := ParseGCPProject(c.Nm); projectNum != "" {
				c.Md = metadata.Pairs("x-goog-user-project", projectNum)
			}

			c.Cc, err = grpc.Dial(c.Info.uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
			if err != nil {
				return fmt.Errorf("connect error: %v", err)
			}
			return nil
		} else {
			return fmt.Errorf("Auto authentication mode for this platform is not supported. Please use jwt_file instead")
		}
	} else {
		return fmt.Errorf("Invalid authn_mode")
	}
}

func New() (*Client, error) {
	c := &Client{
		Info: ParseFlags(),
	}
	if parseerr := c.ParseNodeMatcher(); parseerr != nil {
		return c, parseerr
	}

	if connerr := c.ConnWithAuth(); connerr != nil {
		return c, connerr
	}
	defer c.Cc.Close()

	c.CsdsClient = csdspb.NewClientStatusDiscoveryServiceClient(c.Cc)

	if runerr := c.Run(); runerr != nil {
		return c, runerr
	}
	return c, nil
}

func (c *Client) Run() error {
	var ctx context.Context
	if c.Md != nil {
		ctx = metadata.NewOutgoingContext(context.Background(), c.Md)
	} else {
		ctx = context.Background()
	}

	streamClientStatus, err := c.CsdsClient.StreamClientStatus(ctx)
	if err != nil {
		return fmt.Errorf("stream client status error: %v", err)
	}
	req := &csdspb.ClientStatusRequest{NodeMatchers: c.Nm}
	if err := streamClientStatus.Send(req); err != nil {
		return fmt.Errorf("%v", err)
	}

	resp, err := streamClientStatus.Recv()
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	ParseResponse(resp, c.Info.configFile)

	return nil
}
