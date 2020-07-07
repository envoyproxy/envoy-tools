package main

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
	requestYaml string
	jwt         string
	configFile  string
}

type Client struct {
	cc         *grpc.ClientConn
	csdsClient csdspb.ClientStatusDiscoveryServiceClient

	nm   []*envoy_type_matcher.NodeMatcher
	md   metadata.MD
	info Flag
}

func parseFlags() Flag {
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
		requestYaml: *requestYamlPtr,
		jwt:         *jwtPtr,
		configFile:  *configFilePtr,
	}

	return f
}

func (c *Client) parseNodeMatcher() error {
	if c.info.requestYaml == "" {
		return fmt.Errorf("missing request yaml")
	}

	var nodematchers []*envoy_type_matcher.NodeMatcher
	err := parseYaml(c.info.requestYaml, &nodematchers)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	c.nm = nodematchers
	return nil
}

func (c *Client) ConnWithAuth() error {
	scope := "https://www.googleapis.com/auth/cloud-platform"
	if c.info.authnMode == "jwt" {
		if c.info.jwt == "" {
			return fmt.Errorf("missing jwt file")
		} else {
			pool, err := x509.SystemCertPool()
			creds := credentials.NewClientTLSFromCert(pool, "")
			perRPC, err := oauth.NewServiceAccountFromFile(c.info.jwt, scope) //"/usr/local/google/home/yutongli/service_account_key.json"
			if err != nil {
				return fmt.Errorf("%v", err)
			}

			c.cc, err = grpc.Dial(c.info.uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
			if err != nil {
				return fmt.Errorf("%v", err)
			} else {
				return nil
			}
		}
	} else if c.info.authnMode == "auto" {
		if c.info.platform == "gcp" {
			pool, err := x509.SystemCertPool()
			creds := credentials.NewClientTLSFromCert(pool, "")
			perRPC, err := oauth.NewApplicationDefault(context.Background(), scope) // Application Default Credentials (ADC)
			if err != nil {
				return fmt.Errorf("%v", err)
			}

			// parse GCP project number as header for authentication
			if projectNum := parseGCPProject(c.nm); projectNum != "" {
				c.md = metadata.Pairs("x-goog-user-project", projectNum)
			}

			c.cc, err = grpc.Dial(c.info.uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
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
		info: parseFlags(),
	}
	if parseerr := c.parseNodeMatcher(); parseerr != nil {
		return c, parseerr
	}

	if connerr := c.ConnWithAuth(); connerr != nil {
		return c, connerr
	}
	defer c.cc.Close()

	c.csdsClient = csdspb.NewClientStatusDiscoveryServiceClient(c.cc)

	if runerr := c.Run(); runerr != nil {
		return c, runerr
	}
	return c, nil
}

func (c *Client) Run() error {
	var ctx context.Context
	if c.md != nil {
		ctx = metadata.NewOutgoingContext(context.Background(), c.md)
	} else {
		ctx = context.Background()
	}

	streamClientStatus, err := c.csdsClient.StreamClientStatus(ctx)
	if err != nil {
		return fmt.Errorf("stream client status error: %v", err)
	}
	req := &csdspb.ClientStatusRequest{NodeMatchers: c.nm}
	if err := streamClientStatus.Send(req); err != nil {
		return fmt.Errorf("%v", err)
	}

	resp, err := streamClientStatus.Recv()
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	parseResponse(resp, c.info.configFile)

	return nil
}

func main() {
	_, error := New()
	if error != nil {
		fmt.Println(fmt.Errorf("%v", error).Error())
	}
}
