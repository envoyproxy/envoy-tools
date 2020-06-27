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
)

type Flag struct {
	uri         string
	platform    string
	authnMode   string
	apiVersion  string
	requestYaml string
	jwt         string
}

type Client struct {
	cc         *grpc.ClientConn
	csdsClient csdspb.ClientStatusDiscoveryServiceClient

	nm   *envoy_type_matcher.NodeMatcher
	info Flag
}

func parseFlags() Flag {
	uriPtr := flag.String("service_uri", "trafficdirector.googleapis.com:443", "the uri of the service to connect to")
	platformPtr := flag.String("cloud_platform", "gcp", "the cloud platform (e.g. gcp, aws,  ...)")
	authnModePtr := flag.String("authn_mode", "auto", "the method to use for authentication (e.g. auto, jwt, ...)")
	apiVersionPtr := flag.String("api_version", "v2", "which xds api major version  to use (e.g. v2, v3 ...)")
	requestYamlPtr := flag.String("csds_request_yaml", "", "yaml file that defines the csds request")
	jwtPtr := flag.String("jwt_file", "", "path of the -jwt_file")

	flag.Parse()

	f := Flag{
		uri:         *uriPtr,
		platform:    *platformPtr,
		authnMode:   *authnModePtr,
		apiVersion:  *apiVersionPtr,
		requestYaml: *requestYamlPtr,
		jwt:         *jwtPtr,
	}

	return f
}

func main() () {
	pool, _ := x509.SystemCertPool()
	scope := "https://www.googleapis.com/auth/cloud-platform"
	creds := credentials.NewClientTLSFromCert(pool, "")
	perRPC, _ := oauth.NewServiceAccountFromFile("/usr/local/google/home/yutongli/service_account_key.json", scope)
	conn, connerr := grpc.Dial("trafficdirector.googleapis.com:443", grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
	if connerr != nil {
		error := fmt.Errorf("%v", connerr)
		fmt.Println(error.Error())
	}
	defer conn.Close()

	client := csdspb.NewClientStatusDiscoveryServiceClient(conn)
	streamClient, streamerr := client.StreamClientStatus(context.Background())
	if streamerr != nil {
		error := fmt.Errorf("%v", streamerr)
		fmt.Println("StreamClientStatus Error:")
		fmt.Println(error.Error())
	}
	x := &envoy_type_matcher.NodeMatcher{
		NodeId: &envoy_type_matcher.StringMatcher{
			MatchPattern: &envoy_type_matcher.StringMatcher_Exact{
				Exact: "8576d4bf-8f10-40b2-920b-bb6a7cf9f34a~10.168.0.3",
			},
		},
		NodeMetadatas: []*envoy_type_matcher.StructMatcher{
			{
				Path: []*envoy_type_matcher.StructMatcher_PathSegment{
					{Segment: &envoy_type_matcher.StructMatcher_PathSegment_Key{Key: "TRAFFICDIRECTOR_GCP_PROJECT_NUMBER"}},
				},
				Value: &envoy_type_matcher.ValueMatcher{
					MatchPattern: &envoy_type_matcher.ValueMatcher_StringMatch{
						StringMatch: &envoy_type_matcher.StringMatcher{
							MatchPattern: &envoy_type_matcher.StringMatcher_Exact{
								Exact: "798832730858",
							},
						},
					},
				},
			},
			{
				Path: []*envoy_type_matcher.StructMatcher_PathSegment{
					{Segment: &envoy_type_matcher.StructMatcher_PathSegment_Key{Key: "TRAFFICDIRECTOR_NETWORK_NAME"}},
				},
				Value: &envoy_type_matcher.ValueMatcher{
					MatchPattern: &envoy_type_matcher.ValueMatcher_StringMatch{
						StringMatch: &envoy_type_matcher.StringMatcher{
							MatchPattern: &envoy_type_matcher.StringMatcher_Exact{
								Exact: "default",
							},
						},
					},
				},
			},
		},
	}
	req := &csdspb.ClientStatusRequest{
		NodeMatchers: []*envoy_type_matcher.NodeMatcher{x},
	}
	reqerr := streamClient.Send(req)
	if reqerr != nil {
		error := fmt.Errorf("%v", reqerr)
		fmt.Println("Send Error:")
		fmt.Println(error.Error())
	}

	resp, resperr := streamClient.Recv()
	if resperr != nil {
		error := fmt.Errorf("%v", resperr)
		fmt.Println("Recv Error:")
		fmt.Println(error.Error())
	} else {
		fmt.Println("success")
	}
	//fmt.Println(resp.Config)
	fmt.Println(resp.String())
}
