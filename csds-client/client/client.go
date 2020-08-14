package client

import (
	"time"
)

// ClientOptions are options that are common to use in all the version implementations of client
// TODO: If ClientOptions will no longer be common to use in all the version, it will need to be
//  implemented in version packages
type ClientOptions struct {
	Uri             string
	Platform        string
	AuthnMode       string
	RequestFile     string
	RequestYaml     string
	Jwt             string
	ConfigFile      string
	MonitorInterval time.Duration
	Visualization   bool
}

// Client is an interface of CSDS Client
// Packages which implement this interface in different api versions should have New() and Run() methods
type Client interface {
	Run() error
}
