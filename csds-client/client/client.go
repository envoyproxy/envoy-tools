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

// Client implements CSDS Client of a particular version. Upon creation of the new client it is
// expected that connection to the CSDS server is established.
type Client interface {
	// Run must send an CSDS request to the server and output the response according to the
	// options provided during Client creation.
	Run() error
}
