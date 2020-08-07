package client

import (
	"fmt"
	"google.golang.org/protobuf/encoding/protojson"
	"os"

	csdspb_v2 "github.com/envoyproxy/go-control-plane/envoy/service/status/v2"
)

// parseConfigStatus parses each xds config status to string
func parseConfigStatus_v2(xdsConfig []*csdspb_v2.PerXdsConfig) []string {
	var configStatus []string
	for _, perXdsConfig := range xdsConfig {
		status := perXdsConfig.GetStatus().String()
		var xds string
		if perXdsConfig.GetClusterConfig() != nil {
			xds = "CDS"
		} else if perXdsConfig.GetListenerConfig() != nil {
			xds = "LDS"
		} else if perXdsConfig.GetRouteConfig() != nil {
			xds = "RDS"
		} else if perXdsConfig.GetScopedRouteConfig() != nil {
			xds = "SRDS"
		}
		if status != "" && xds != "" {
			configStatus = append(configStatus, xds+"   "+status)
		}
	}
	return configStatus
}

// printOutResponse processes response and print
func printOutResponse_v2(response *csdspb_v2.ClientStatusResponse, info ClientOptions) error {
	if response.GetConfig() == nil || len(response.GetConfig()) == 0 {
		fmt.Printf("No xDS clients connected.\n")
		return nil
	} else {
		fmt.Printf("%-50s %-30s %-30s \n", "Client ID", "xDS stream type", "Config Status")
	}

	var hasXdsConfig bool

	for _, config := range response.GetConfig() {
		var id string
		var xdsType string
		if config.GetNode() != nil {
			id = config.GetNode().GetId()
			metadata := config.GetNode().GetMetadata().AsMap()

			// control plane is expected to use "XDS_STREAM_TYPE" to communicate
			// the stream type of the connected client in the response.
			if metadata["XDS_STREAM_TYPE"] != nil {
				xdsType = metadata["XDS_STREAM_TYPE"].(string)
			}
		}

		if config.GetXdsConfig() == nil {
			if config.GetNode() != nil {
				fmt.Printf("%-50s %-30s %-30s \n", id, xdsType, "N/A")
			}
		} else {
			hasXdsConfig = true

			// parse config status
			configStatus := parseConfigStatus_v2(config.GetXdsConfig())
			fmt.Printf("%-50s %-30s ", id, xdsType)

			for i := 0; i < len(configStatus); i++ {
				if i == 0 {
					fmt.Printf("%-30s \n", configStatus[i])
				} else {
					fmt.Printf("%-50s %-30s %-30s \n", "", "", configStatus[i])
				}
			}
			if len(configStatus) == 0 {
				fmt.Printf("\n")
			}
		}
	}

	if hasXdsConfig {
		// parse response to json
		// format the json and resolve google.protobuf.Any types
		m := protojson.MarshalOptions{Multiline: true, Indent: "  ", Resolver: &TypeResolver{}}
		out, err := m.Marshal(response)
		if err != nil {
			return err
		}

		if info.ConfigFile == "" {
			// output the configuration to stdout by default
			fmt.Println("Detailed Config:")
			fmt.Println(string(out))
		} else {
			// write the configuration to the file
			f, err := os.Create(info.ConfigFile)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = f.Write(out)
			if err != nil {
				return err
			}
			fmt.Printf("Config has been saved to %v\n", info.ConfigFile)
		}

		// call visualize to enable visualization
		if info.Visualization {
			if err := visualize(out, info.MonitorInterval != 0); err != nil {
				return err
			}
		}
	}
	return nil
}
