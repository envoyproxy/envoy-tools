package main

import (
	"envoy-tools/csds-client/client"
	"flag"
	"log"
)

// ParseFLags parses flags to ClientOptions
func ParseFlags() client.ClientOptions {
	uriPtr := flag.String("service_uri", "trafficdirector.googleapis.com:443", "the uri of the service to connect to")
	platformPtr := flag.String("cloud_platform", "gcp", "the cloud platform (e.g. gcp, aws,  ...)")
	authnModePtr := flag.String("authn_mode", "auto", "the method to use for authentication (e.g. auto, jwt, ...)")
	apiVersionPtr := flag.String("api_version", "v2", "which xds api major version  to use (e.g. v2, v3 ...)")
	requestFilePtr := flag.String("request_file", "", "yaml file that defines the csds request")
	requestYamlPtr := flag.String("request_yaml", "", "yaml string that defines the csds request")
	jwtPtr := flag.String("jwt_file", "", "path of the -jwt_file")
	configFilePtr := flag.String("file_to_save_config", "", "file name to save configs returned by csds response")
	monitorIntervalPtr := flag.Duration("monitor_interval", 0, "the interval of sending request in monitor mode (e.g. 500ms, 2s, 1m ...)")
	visualizationPtr := flag.Bool("visualization", false, "option to visualize the relationship between xDS")

	flag.Parse()

	f := client.ClientOptions{
		Uri:             *uriPtr,
		Platform:        *platformPtr,
		AuthnMode:       *authnModePtr,
		ApiVersion:      *apiVersionPtr,
		RequestFile:     *requestFilePtr,
		RequestYaml:     *requestYamlPtr,
		Jwt:             *jwtPtr,
		ConfigFile:      *configFilePtr,
		MonitorInterval: *monitorIntervalPtr,
		Visualization:   *visualizationPtr,
	}

	return f
}

func main() {
	c, err := client.New(ParseFlags())
	if err != nil {
		log.Fatal(err)
	}

	if err := c.Run(); err != nil {
		log.Fatal(err)
	}
}
