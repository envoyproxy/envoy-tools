package main

import (
	"context"
	"crypto/x509"
	"fmt"
	csdspb "github.com/envoyproxy/go-control-plane/envoy/service/status/v2"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
)

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
