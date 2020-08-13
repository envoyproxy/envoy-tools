package main

import (
	"envoy-tools/csds-client/client"
	client_v2 "envoy-tools/csds-client/client/v2"
	"flag"
	"log"
	"time"
)

// GetClientOptionsFromFlags parses flags to ClientOptions
func GetClientOptionsFromFlags() client.ClientOptions {
	const uriDefault string = "trafficdirector.googleapis.com:443"
	const platformDefault string = "gcp"
	const authnModeDefault string = "auto"
	const apiVersionDefault string = "v2"
	const requestFileDefault string = ""
	const requestYamlDefault string = ""
	const jwtDefault string = ""
	const configFileDefault string = ""
	const monitorIntervalDefault time.Duration = 0
	const visualizationDefault bool = false

	uriPtr := flag.String("service_uri", uriDefault, "the uri of the service to connect to")
	platformPtr := flag.String("platform", platformDefault, "the platform (e.g. gcp, aws,  ...)")
	authnModePtr := flag.String("authn_mode", authnModeDefault, "the method to use for authentication (e.g. auto, jwt, ...)")
	apiVersionPtr := flag.String("api_version", apiVersionDefault, "which xds api major version  to use (e.g. v2, v3 ...)")
	requestFilePtr := flag.String("request_file", requestFileDefault, "yaml file that defines the csds request")
	requestYamlPtr := flag.String("request_yaml", requestYamlDefault, "yaml string that defines the csds request")
	jwtPtr := flag.String("jwt_file", jwtDefault, "path of the -jwt_file")
	configFilePtr := flag.String("output_file", configFileDefault, "file name to save configs returned by csds response")
	monitorIntervalPtr := flag.Duration("monitor_interval", monitorIntervalDefault, "the interval of sending request in monitor mode (e.g. 500ms, 2s, 1m ...)")
	visualizationPtr := flag.Bool("visualization", visualizationDefault, "option to visualize the relationship between xDS")

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
	var c client.Client
	var err error
	clientOpts := GetClientOptionsFromFlags()

	if clientOpts.ApiVersion == "v2" {
		c, err = client_v2.New(clientOpts)
	} else {
		log.Fatal("invalid api version")
	}

	if err != nil {
		log.Fatal(err)
	}

	if err := c.Run(); err != nil {
		log.Fatal(err)
	}
}
