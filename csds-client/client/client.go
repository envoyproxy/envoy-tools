package client

import (
	csdspb "github.com/envoyproxy/go-control-plane/envoy/service/status/v2"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"

	"context"
	"crypto/x509"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/grpc/metadata"
	"os"
	"os/signal"
	"syscall"
	"time"
)

type Flag struct {
	uri         string
	platform    string
	authnMode   string
	apiVersion  string
	requestFile string
	requestYaml string
	jwt         string
	configFile  string
	monitorFreq string
}

type Client struct {
	cc         *grpc.ClientConn
	csdsClient csdspb.ClientStatusDiscoveryServiceClient

	nm   []*envoy_type_matcher.NodeMatcher
	md   metadata.MD
	info Flag
}

// ParseFLags parses flags to info
func ParseFlags() Flag {
	uriPtr := flag.String("service_uri", "trafficdirector.googleapis.com:443", "the uri of the service to connect to")
	platformPtr := flag.String("cloud_platform", "gcp", "the cloud platform (e.g. gcp, aws,  ...)")
	authnModePtr := flag.String("authn_mode", "auto", "the method to use for authentication (e.g. auto, jwt, ...)")
	apiVersionPtr := flag.String("api_version", "v2", "which xds api major version  to use (e.g. v2, v3 ...)")
	requestFilePtr := flag.String("request_file", "", "yaml file that defines the csds request")
	requestYamlPtr := flag.String("request_yaml", "", "yaml string that defines the csds request")
	jwtPtr := flag.String("jwt_file", "", "path of the -jwt_file")
	configFilePtr := flag.String("file_to_save_config", "", "the file name to save config")
	monitorFreqPtr := flag.String("monitor_freq", "", "the frequency of sending request in monitor mode (e.g. 500ms, 2s, 1m ...)")

	flag.Parse()

	f := Flag{
		uri:         *uriPtr,
		platform:    *platformPtr,
		authnMode:   *authnModePtr,
		apiVersion:  *apiVersionPtr,
		requestFile: *requestFilePtr,
		requestYaml: *requestYamlPtr,
		jwt:         *jwtPtr,
		configFile:  *configFilePtr,
		monitorFreq: *monitorFreqPtr,
	}

	return f
}

// parseNodeMatcher parses the csds request yaml to nodematcher
func (c *Client) parseNodeMatcher() error {
	if c.info.requestFile == "" && c.info.requestYaml == "" {
		return fmt.Errorf("missing request yaml")
	}

	var nodematchers []*envoy_type_matcher.NodeMatcher
	if err := parseYaml(c.info.requestFile, c.info.requestYaml, &nodematchers); err != nil {
		return fmt.Errorf("%v", err)
	}

	c.nm = nodematchers
	return nil
}

// connWithAuth connects to uri with authentication
func (c *Client) connWithAuth() error {
	var scope string
	switch c.info.authnMode {
	case "jwt":
		if c.info.jwt == "" {
			return fmt.Errorf("missing jwt file")
		}
		switch c.info.platform {
		case "gcp":
			scope = "https://www.googleapis.com/auth/cloud-platform"
			pool, err := x509.SystemCertPool()
			creds := credentials.NewClientTLSFromCert(pool, "")
			perRPC, err := oauth.NewServiceAccountFromFile(c.info.jwt, scope)
			if err != nil {
				return fmt.Errorf("%v", err)
			}

			c.cc, err = grpc.Dial(c.info.uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
			if err != nil {
				return fmt.Errorf("%v", err)
			} else {
				return nil
			}
		default:
			return nil
		}
	case "auto":
		switch c.info.platform {
		case "gcp":
			scope = "https://www.googleapis.com/auth/cloud-platform"
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
		default:
			return fmt.Errorf("Auto authentication mode for this platform is not supported. Please use jwt_file instead")
		}
	default:
		return fmt.Errorf("Invalid authn_mode")
	}
}

// New creates a new client
func New() (*Client, error) {
	c := &Client{
		info: ParseFlags(),
	}
	if c.info.platform != "gcp" {
		return c, fmt.Errorf("Can not support this platform now")
	}
	if c.info.apiVersion != "v2" {
		return c, fmt.Errorf("Can not suppoort this api version now")
	}

	if err := c.parseNodeMatcher(); err != nil {
		return c, err
	}

	return c, nil
}

// Run connects the client to the uri and call doRequest based on monitor_freq flag
func (c *Client) Run() error {
	if err := c.connWithAuth(); err != nil {
		return err
	}
	defer c.cc.Close()

	c.csdsClient = csdspb.NewClientStatusDiscoveryServiceClient(c.cc)
	var ctx context.Context
	if c.md != nil {
		ctx = metadata.NewOutgoingContext(context.Background(), c.md)
	} else {
		ctx = context.Background()
	}
	streamClientStatus, err := c.csdsClient.StreamClientStatus(ctx)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	// run once or run with monitor mode
	switch c.info.monitorFreq {
	case "":
		if err := c.doRequest(streamClientStatus); err != nil {
			return err
		}
		return nil
	default:
		freq, err := time.ParseDuration(c.info.monitorFreq)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		ticker := time.NewTicker(freq)

		// keep track of 'ctrl+c' to stop
		s := make(chan os.Signal)
		signal.Notify(s, os.Interrupt, syscall.SIGTERM)
		go func() {
			for {
				select {
				case <-s:
					fmt.Println("Client Stopped")
					os.Exit(0)
				case t := <-ticker.C:
					fmt.Printf("Sent request on %v\n", t)
					if err = c.doRequest(streamClientStatus); err != nil {
						return
					}
				}
			}
		}()
		time.Sleep(time.Minute)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		return nil
	}
}

// doRequest sends request and print out the parsed response
func (c *Client) doRequest(streamClientStatus csdspb.ClientStatusDiscoveryService_StreamClientStatusClient) error {

	req := &csdspb.ClientStatusRequest{NodeMatchers: c.nm}
	if err := streamClientStatus.Send(req); err != nil {
		return fmt.Errorf("%v", err)
	}

	resp, err := streamClientStatus.Recv()
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	// post process response
	printOutResponse(resp, c.info.configFile)

	return nil
}
