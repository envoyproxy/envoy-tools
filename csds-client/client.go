package main

import (
	"context"
	"crypto/x509"
	"fmt"
	csdspb "github.com/envoyproxy/go-control-plane/envoy/service/status/v2"
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
		fmt.Println(error.Error())
	}
	resp, resperr := streamClient.Recv()
	if resperr != nil {
		error := fmt.Errorf("%v", resperr)
		fmt.Println(error.Error())
	} else {
		fmt.Println("success")
	}
	fmt.Println(resp.Config)
	fmt.Println(resp.String())
}
