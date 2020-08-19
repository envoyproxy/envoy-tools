package main

import (
	"flag"
	"log"
	"time"

	"envoy-tools/csds-client/client"
	client_v2 "envoy-tools/csds-client/client/v2"
	client_v3 "envoy-tools/csds-client/client/v3"
)

// flag vars
var uri string
var platform string
var authnMode string
var apiVersion int
var requestFile string
var requestYaml string
var jwt string
var configFile string
var monitorInterval time.Duration
var visualization bool

// const default values for flag vars
const (
	uriDefault             string        = "trafficdirector.googleapis.com:443"
	platformDefault        string        = "gcp"
	authnModeDefault       string        = "auto"
	apiVersionDefault      int           = 2
	requestFileDefault     string        = ""
	requestYamlDefault     string        = ""
	jwtDefault             string        = ""
	configFileDefault      string        = ""
	monitorIntervalDefault time.Duration = 0
	visualizationDefault   bool          = false
)

// init binds flags with variables
func init() {
	flag.StringVar(&uri, "service_uri", uriDefault, "the uri of the service to connect to")
	flag.StringVar(&platform, "platform", platformDefault, "the platform (e.g. gcp, aws,  ...)")
	flag.StringVar(&authnMode, "authn_mode", authnModeDefault, "the method to use for authentication (e.g. auto, jwt, ...)")
	flag.IntVar(&apiVersion, "api_version", apiVersionDefault, "which xds api major version to use (e.g. 2, 3 ...)")
	flag.StringVar(&requestFile, "request_file", requestFileDefault, "yaml file that defines the csds request")
	flag.StringVar(&requestYaml, "request_yaml", requestYamlDefault, "yaml string that defines the csds request")
	flag.StringVar(&jwt, "jwt_file", jwtDefault, "path of the -jwt_file")
	flag.StringVar(&configFile, "output_file", configFileDefault, "file name to save configs returned by csds response")
	flag.DurationVar(&monitorInterval, "monitor_interval", monitorIntervalDefault, "the interval of sending request in monitor mode (e.g. 500ms, 2s, 1m ...)")
	flag.BoolVar(&visualization, "visualization", visualizationDefault, "option to visualize the relationship between xDS")
}

func main() {
	flag.Parse()

	clientOpts := client.ClientOptions{
		Uri:             uri,
		Platform:        platform,
		AuthnMode:       authnMode,
		RequestFile:     requestFile,
		RequestYaml:     requestYaml,
		Jwt:             jwt,
		ConfigFile:      configFile,
		MonitorInterval: monitorInterval,
		Visualization:   visualization,
	}

	var c client.Client
	var err error
	if apiVersion == 2 {
		c, err = client_v2.New(clientOpts)
	} else if apiVersion == 3 {
		c, err = client_v3.New(clientOpts)
	} else {
		log.Fatalf("Unsupported xDS API version: %v", apiVersion)
	}

	if err != nil {
		log.Fatal(err)
	}

	if err := c.Run(); err != nil {
		log.Fatal(err)
	}
}
