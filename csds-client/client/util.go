package client

import (
	"encoding/json"
	"fmt"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_config_filter_http_router_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/router/v2"
	envoy_config_filter_network_http_connection_manager_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	envoy_service_status_v2 "github.com/envoyproxy/go-control-plane/envoy/service/status/v2"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"github.com/ghodss/yaml"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"io/ioutil"
	"os"
	"path/filepath"
)

func ParseYaml(path string, nms *[]*envoy_type_matcher.NodeMatcher) error {
	filename, _ := filepath.Abs(path)
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("%v", err)
	}
	js, err := yaml.YAMLToJSON(yamlFile)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	var data map[string]interface{}
	err = json.Unmarshal(js, &data)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	for _, n := range data["node_matchers"].([]interface{}) {
		x := &envoy_type_matcher.NodeMatcher{}

		jsonString, err := json.Marshal(n)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		err = protojson.Unmarshal(jsonString, x)
		if err != nil {
			return fmt.Errorf("%v", err)
		}
		*nms = append(*nms, x)
	}
	return nil
}

func ParseGCPProject(nms []*envoy_type_matcher.NodeMatcher) string {
	for _, nm := range nms {
		for _, mt := range nm.NodeMetadatas {
			for _, path := range mt.Path {
				if path.GetKey() == "TRAFFICDIRECTOR_GCP_PROJECT_NUMBER" {
					return mt.Value.GetStringMatch().GetExact()
				}
			}
		}
	}
	return ""
}

// implement protoregistry.ExtensionTypeResolver and protoregistry.MessageTypeResolver to resolve google.protobuf.Any types
type TypeResolver struct{}

func (r *TypeResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	return nil, protoregistry.NotFound
}

func (r *TypeResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	if url == "type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager" {
		httpConnectionManager := envoy_config_filter_network_http_connection_manager_v2.HttpConnectionManager{}
		return httpConnectionManager.ProtoReflect().Type(), nil
	}
	if url == "type.googleapis.com/envoy.api.v2.Cluster" {
		cluster := envoy_api_v2.Cluster{}
		return cluster.ProtoReflect().Type(), nil
	}
	if url == "type.googleapis.com/envoy.api.v2.Listener" {
		listener := envoy_api_v2.Listener{}
		return listener.ProtoReflect().Type(), nil
	}
	if url == "type.googleapis.com/envoy.config.filter.http.router.v2.Router" {
		router := envoy_config_filter_http_router_v2.Router{}
		return router.ProtoReflect().Type(), nil
	}
	if url == "type.googleapis.com/envoy.api.v2.RouteConfiguration" {
		routeConfiguration := envoy_api_v2.RouteConfiguration{}
		return routeConfiguration.ProtoReflect().Type(), nil
	}
	return nil, protoregistry.NotFound
}

func (r *TypeResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}

func (r *TypeResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}

func ParseResponse(response *envoy_service_status_v2.ClientStatusResponse, fileName string) {
	fmt.Printf("%-50s %-30s %-30s \n", "Client ID", "xDS stream type", "Config")
	for _, config := range response.Config {
		id := config.Node.GetId()
		metadata := config.Node.GetMetadata().AsMap()
		xdsType := metadata["TRAFFIC_DIRECTOR_XDS_STREAM_TYPE"]
		if xdsType == nil {
			xdsType = ""
		}

		var configFile string
		if config.GetXdsConfig() == nil {
			configFile = "N/A"
		} else {
			if fileName == "" {
				configFile = id + "_" + xdsType.(string) + "_config.json"
			} else {
				configFile = fileName
			}

			// parse response to json and write to the file
			f, err := os.Create(configFile)
			if err != nil {
				fmt.Println(err)
			}
			defer f.Close()

			m := protojson.MarshalOptions{Multiline: true, Indent: "  ", Resolver: &TypeResolver{}}
			out, err := m.Marshal(response)
			if err != nil {
				fmt.Printf("%v", err)
			}
			_, err = f.Write(out)
			if err != nil {
				fmt.Printf("%v", err)
			}
		}

		fmt.Printf("%-50s %-30s %-30s \n", id, xdsType, configFile)
	}
}
