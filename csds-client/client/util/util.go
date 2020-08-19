package util

import (
	"bytes"
	"context"
	"crypto/x509"
	"encoding/json"
	"envoy-tools/csds-client/client"
	"errors"
	"fmt"
	"github.com/ghodss/yaml"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"

	"github.com/awalterschulze/gographviz"
	"github.com/emirpasic/gods/sets/treeset"
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_config_cluster_v3 "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	envoy_config_endpoint_v3 "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	envoy_config_filter_http_router_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/router/v2"
	envoy_config_filter_network_http_connection_manager_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	envoy_config_listener_v3 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	envoy_config_route_v3 "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	envoy_extensions_filters_http_router_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	envoy_extensions_filters_network_http_connection_manager_v3 "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/oauth"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
)

// IsJson checks if str is a valid json format string
func IsJson(str string) bool {
	input := []byte(str)
	decoder := json.NewDecoder(bytes.NewReader(input))
	for {
		_, err := decoder.Token()
		if err == io.EOF { // end of string
			break
		}
		if err != nil {
			return false
		}
	}
	return true
}

// TypeResolver implements protoregistry.ExtensionTypeResolver and protoregistry.MessageTypeResolver to resolve google.protobuf.Any types
type TypeResolver struct{}

func (r *TypeResolver) FindMessageByName(message protoreflect.FullName) (protoreflect.MessageType, error) {
	return nil, protoregistry.NotFound
}

// FindMessageByURL links the message type url to the specific message type
// TODO: If there's other message type can be passed in google.protobuf.Any, the typeUrl and
//  messageType need to be added to this method to make sure it can be parsed and output correctly
func (r *TypeResolver) FindMessageByURL(url string) (protoreflect.MessageType, error) {
	switch url {
	case "type.googleapis.com/envoy.config.filter.network.http_connection_manager.v2.HttpConnectionManager":
		httpConnectionManager := envoy_config_filter_network_http_connection_manager_v2.HttpConnectionManager{}
		return httpConnectionManager.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager":
		httpConnectionManager := envoy_extensions_filters_network_http_connection_manager_v3.HttpConnectionManager{}
		return httpConnectionManager.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.api.v2.Cluster":
		cluster := envoy_api_v2.Cluster{}
		return cluster.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.config.cluster.v3.Cluster":
		cluster := envoy_config_cluster_v3.Cluster{}
		return cluster.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.api.v2.Listener":
		listener := envoy_api_v2.Listener{}
		return listener.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.config.listener.v3.Listener":
		listener := envoy_config_listener_v3.Listener{}
		return listener.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.config.filter.http.router.v2.Router":
		router := envoy_config_filter_http_router_v2.Router{}
		return router.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.extensions.filters.http.router.v3.Router":
		router := envoy_extensions_filters_http_router_v3.Router{}
		return router.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.api.v2.RouteConfiguration":
		routeConfiguration := envoy_api_v2.RouteConfiguration{}
		return routeConfiguration.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.config.route.v3.RouteConfiguration":
		routeConfiguration := envoy_config_route_v3.RouteConfiguration{}
		return routeConfiguration.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.config.endpoint.v3.ClusterLoadAssignment":
		clusterLoadAssignment := envoy_config_endpoint_v3.ClusterLoadAssignment{}
		return clusterLoadAssignment.ProtoReflect().Type(), nil
	default:
		return nil, protoregistry.NotFound
	}
}

func (r *TypeResolver) FindExtensionByName(field protoreflect.FullName) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}

func (r *TypeResolver) FindExtensionByNumber(message protoreflect.FullName, field protoreflect.FieldNumber) (protoreflect.ExtensionType, error) {
	return nil, protoregistry.NotFound
}

// Visualize calls ParseXdsRelationship and use the result to Visualize
func Visualize(config []byte, monitor bool) error {
	graphData, err := ParseXdsRelationship(config)
	if err != nil {
		return err
	}
	dot, err := GenerateGraph(graphData)
	if err != nil {
		return err
	}

	if !monitor {
		url := "http://dreampuf.github.io/GraphvizOnline/#" + dot
		if err := OpenBrowser(url); err != nil {
			return err
		}
	}

	// save dot to file
	f, err := os.Create("config_graph.dot")
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write([]byte(dot))
	if err != nil {
		return err
	}
	fmt.Println("Config graph has been saved to config_graph.dot")
	return nil
}

// struct stores the nodes and edges maps of graph
type GraphData struct {
	nodes     []map[string]string
	relations []map[string]*treeset.Set
}

// ParseXdsRelationship parses relationship between xds and stores them in GraphData
func ParseXdsRelationship(js []byte) (GraphData, error) {
	var data map[string]interface{}
	err := json.Unmarshal(js, &data)
	if err != nil {
		return GraphData{}, err
	}
	lds := make(map[string]string)
	rds := make(map[string]string)
	cds := make(map[string]string)
	eds := make(map[string]string)
	ldsToRds := make(map[string]*treeset.Set)
	rdsToCds := make(map[string]*treeset.Set)
	cdsToEds := make(map[string]*treeset.Set)

	for _, config := range data["config"].([]interface{}) {
		configMap := config.(map[string]interface{})
		for _, xds := range configMap["xdsConfig"].([]interface{}) {
			for key, value := range xds.(map[string]interface{}) {
				if key == "status" {
					continue
				}
				switch key {
				case "listenerConfig":
					for _, listeners := range value.(map[string]interface{}) {
						for idx, listener := range listeners.([]interface{}) {
							detail := listener.(map[string]interface{})["activeState"].(map[string]interface{})["listener"].(map[string]interface{})
							name := detail["name"].(string)
							id := "LDS" + strconv.Itoa(idx)
							lds[name] = id
							rdsSet := treeset.NewWithStringComparator()

							for _, filterchain := range detail["filterChains"].([]interface{}) {
								for _, filter := range filterchain.(map[string]interface{})["filters"].([]interface{}) {
									rdsName := filter.(map[string]interface{})["typedConfig"].(map[string]interface{})["rds"].(map[string]interface{})["routeConfigName"].(string)
									rdsSet.Add(rdsName)
								}
							}
							ldsToRds[name] = rdsSet
						}
					}
				case "routeConfig":
					for _, routes := range value.(map[string]interface{}) {
						for idx, route := range routes.([]interface{}) {
							routeConfig := route.(map[string]interface{})["routeConfig"].(map[string]interface{})
							name := routeConfig["name"].(string)
							id := "RDS" + strconv.Itoa(idx)
							rds[name] = id
							cdsSet := treeset.NewWithStringComparator()

							for _, virtualHost := range routeConfig["virtualHosts"].([]interface{}) {
								for _, virtualRoutes := range virtualHost.(map[string]interface{})["routes"].([]interface{}) {
									virtualRoute := virtualRoutes.(map[string]interface{})["route"].(map[string]interface{})
									if weightedClusters, ok := virtualRoute["weightedClusters"]; ok {
										for _, cluster := range weightedClusters.(map[string]interface{})["clusters"].([]interface{}) {
											cdsName := cluster.(map[string]interface{})["name"].(string)
											cdsSet.Add(cdsName)
										}
									} else {
										cdsName := virtualRoute["cluster"].(string)
										cdsSet.Add(cdsName)
									}
								}
							}
							rdsToCds[name] = cdsSet
						}
					}
				case "clusterConfig":
					for _, clusters := range value.(map[string]interface{}) {
						for idx, cluster := range clusters.([]interface{}) {
							name := cluster.(map[string]interface{})["cluster"].(map[string]interface{})["name"].(string)
							id := "CDS" + strconv.Itoa(idx)
							cds[name] = id
						}
					}
				case "endpointConfig":
					for _, endpoints := range value.(map[string]interface{}) {
						for idx, endpoint := range endpoints.([]interface{}) {
							id := "EDS" + strconv.Itoa(idx)
							eds[id] = id

							clusterName := endpoint.(map[string]interface{})["endpointConfig"].(map[string]interface{})["clusterName"].(string)
							if cdsSet, ok := cdsToEds[clusterName]; ok {
								cdsSet.Add(id)
							} else {
								cdsSet = treeset.NewWithStringComparator()
								cdsSet.Add(id)
								cdsToEds[clusterName] = cdsSet
							}
						}
					}
				}
			}
		}
	}

	gData := GraphData{
		nodes:     []map[string]string{lds, rds, cds, eds},
		relations: []map[string]*treeset.Set{ldsToRds, rdsToCds, cdsToEds},
	}

	return gData, nil
}

// GenerateGraph generates dot string based on GraphData
func GenerateGraph(data GraphData) (string, error) {
	graphAst, err := gographviz.ParseString(`digraph G {}`)
	if err != nil {
		return "", err
	}
	graph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, graph); err != nil {
		return "", err
	}

	if err := graph.AddAttr("G", "rankdir", "LR"); err != nil {
		return "", err
	}

	// different colors for xDS nodes
	colors := map[string]string{"LDS": "#4285F4", "RDS": "#EA4335", "CDS": "#FBBC04", "EDS": "#34A853"}

	for _, xDS := range data.nodes {
		for name, node := range xDS {
			if err := graph.AddNode("G", `\"`+name+`\"`, map[string]string{"label": node, "fontcolor": "white", "fontname": "Roboto", "shape": "box", "style": `\""filled,rounded"\"`, "color": `\"` + colors[node[0:3]] + `\"`, "fillcolor": `\"` + colors[node[0:3]] + `\"`}); err != nil {
				return "", err
			}
		}
	}
	for _, relations := range data.relations {
		for src, set := range relations {
			for _, dst := range set.Values() {
				if err := graph.AddEdge(`\"`+src+`\"`, `\"`+dst.(string)+`\"`, true, map[string]string{"penwidth": "0.3", "arrowsize": "0.3"}); err != nil {
					return "", err
				}
			}
		}
	}

	return graph.String(), nil
}

// OpenBrowser opens url in browser based on platform
func OpenBrowser(url string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		return err
	}
	return nil
}

// PrintDetailedConfig prints out the detailed xDS config and calls visualize() if it is enabled
func PrintDetailedConfig(response proto.Message, opts client.ClientOptions) error {
	// parse response to json
	// format the json and resolve google.protobuf.Any types
	m := protojson.MarshalOptions{Multiline: true, Indent: "  ", Resolver: &TypeResolver{}}
	out, err := m.Marshal(response)
	if err != nil {
		return err
	}

	if opts.ConfigFile == "" {
		// output the configuration to stdout by default
		fmt.Println("Detailed Config:")
		fmt.Println(string(out))
	} else {
		// write the configuration to the file
		f, err := os.Create(opts.ConfigFile)
		if err != nil {
			return err
		}
		defer f.Close()
		_, err = f.Write(out)
		if err != nil {
			return err
		}
		fmt.Printf("Config has been saved to %v\n", opts.ConfigFile)
	}

	// call visualize to enable visualization
	if opts.Visualization {
		if err := Visualize(out, opts.MonitorInterval != 0); err != nil {
			return err
		}
	}
	return nil
}

// ConnWithJwt connects to uri with jwt authentication
func ConnWithJwt(opts client.ClientOptions) (*grpc.ClientConn, error) {
	if opts.Jwt == "" {
		return nil, errors.New("missing jwt file")
	}
	switch opts.Platform {
	case "gcp":
		scope := "https://www.googleapis.com/auth/cloud-platform"
		pool, err := x509.SystemCertPool()
		if err != nil {
			return nil, err
		}
		creds := credentials.NewClientTLSFromCert(pool, "")
		perRPC, err := oauth.NewServiceAccountFromFile(opts.Jwt, scope)
		if err != nil {
			return nil, err
		}

		clientConn, err := grpc.Dial(opts.Uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
		if err != nil {
			return nil, err
		}
		return clientConn, nil
	default:
		return nil, fmt.Errorf("%s platform is not supported, list of supported platforms: gcp", opts.Platform)
	}
}

// ConnWithAutoGcp connects to uri on gcp with auto authentication
func ConnWithAutoGcp(opts client.ClientOptions) (*grpc.ClientConn, error) {
	scope := "https://www.googleapis.com/auth/cloud-platform"
	pool, err := x509.SystemCertPool()
	if err != nil {
		return nil, err
	}
	creds := credentials.NewClientTLSFromCert(pool, "")
	perRPC, err := oauth.NewApplicationDefault(context.Background(), scope) // Application Default Credentials (ADC)
	if err != nil {
		return nil, err
	}

	clientConn, err := grpc.Dial(opts.Uri, grpc.WithTransportCredentials(creds), grpc.WithPerRPCCredentials(perRPC))
	if err != nil {
		return nil, err
	}
	return clientConn, nil
}

// ParseYamlFileToMap parses yaml file to map
func ParseYamlFileToMap(path string) (map[string]interface{}, error) {
	// parse yaml to json
	filename, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	yamlFile, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	js, err := yaml.YAMLToJSON(yamlFile)
	if err != nil {
		return nil, err
	}

	// parse the json array to a map to iterate it
	var data map[string]interface{}
	if err = json.Unmarshal(js, &data); err != nil {
		return nil, err
	}
	return data, nil
}

// ParseYamlStrToMap parses yaml string to map
func ParseYamlStrToMap(yamlStr string) (map[string]interface{}, error) {
	var js []byte
	var err error
	// json input
	if IsJson(yamlStr) {
		js = []byte(yamlStr)
	} else {
		// parse the yaml input into json
		js, err = yaml.YAMLToJSON([]byte(yamlStr))
		if err != nil {
			return nil, err
		}
	}

	// parse the json array to a map to iterate it
	var data map[string]interface{}
	if err = json.Unmarshal(js, &data); err != nil {
		return nil, err
	}
	return data, nil
}
