package client

import (
	"time"
)

type ClientOptions struct {
	Uri             string
	Platform        string
	AuthnMode       string
	ApiVersion      string
	RequestFile     string
	RequestYaml     string
	Jwt             string
	ConfigFile      string
	MonitorInterval time.Duration
	Visualization   bool
}

type Client interface {
	Run() error
}
