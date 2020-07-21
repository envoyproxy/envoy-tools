package client

import (
	envoy_api_v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	envoy_config_filter_http_router_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/http/router/v2"
	envoy_config_filter_network_http_connection_manager_v2 "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	envoy_service_status_v2 "github.com/envoyproxy/go-control-plane/envoy/service/status/v2"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"

	"bytes"
	"encoding/json"
	"fmt"
	"github.com/awalterschulze/gographviz"
	"github.com/emirpasic/gods/sets/treeset"
	"github.com/ghodss/yaml"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
)

// isJson checks if str is a valid json format string
func isJson(str string) bool {
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

// parseYaml is a helper method for parsing csds request yaml to nodematchers
func parseYaml(path string, yamlStr string, nms *[]*envoy_type_matcher.NodeMatcher) error {
	if path != "" {
		// parse yaml to json
		filename, err := filepath.Abs(path)
		if err != nil {
			return err
		}
		yamlFile, err := ioutil.ReadFile(filename)
		if err != nil {
			return err
		}
		js, err := yaml.YAMLToJSON(yamlFile)
		if err != nil {
			return err
		}

		// parse the json array to a map to iterate it
		var data map[string]interface{}
		if err = json.Unmarshal(js, &data); err != nil {
			return err
		}

		// parse each json object to proto
		for _, n := range data["node_matchers"].([]interface{}) {
			x := &envoy_type_matcher.NodeMatcher{}

			jsonString, err := json.Marshal(n)
			if err != nil {
				return err
			}
			if err = protojson.Unmarshal(jsonString, x); err != nil {
				return err
			}
			*nms = append(*nms, x)
		}
	}
	if yamlStr != "" {
		var js []byte
		var err error
		// json input
		if isJson(yamlStr) {
			js = []byte(yamlStr)
		} else {
			// parse the yaml input into json
			js, err = yaml.YAMLToJSON([]byte(yamlStr))
			if err != nil {
				return err
			}
		}

		// parse the json array to a map to iterate it
		var data map[string]interface{}
		if err = json.Unmarshal(js, &data); err != nil {
			return err
		}

		// parse each json object to proto
		i := 0
		for _, n := range data["node_matchers"].([]interface{}) {
			x := &envoy_type_matcher.NodeMatcher{}

			jsonString, err := json.Marshal(n)
			if err != nil {
				return err
			}
			if err = protojson.Unmarshal(jsonString, x); err != nil {
				return err
			}

			// merge the proto with existing proto from request_file
			if i < len(*nms) {
				proto.Merge((*nms)[i], x)
			} else {
				*nms = append(*nms, x)
			}
			i++
		}
	}
	return nil
}

// getValueByKeyFromNodeMatcher get value by key from metadata of nodematchers
func getValueByKeyFromNodeMatcher(nms []*envoy_type_matcher.NodeMatcher, key string) string {
	for _, nm := range nms {
		for _, mt := range nm.NodeMetadatas {
			for _, path := range mt.Path {
				if path.GetKey() == key {
					return mt.Value.GetStringMatch().GetExact()
				}
			}
		}
	}
	return ""
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
	case "type.googleapis.com/envoy.api.v2.Cluster":
		cluster := envoy_api_v2.Cluster{}
		return cluster.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.api.v2.Listener":
		listener := envoy_api_v2.Listener{}
		return listener.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.config.filter.http.router.v2.Router":
		router := envoy_config_filter_http_router_v2.Router{}
		return router.ProtoReflect().Type(), nil
	case "type.googleapis.com/envoy.api.v2.RouteConfiguration":
		routeConfiguration := envoy_api_v2.RouteConfiguration{}
		return routeConfiguration.ProtoReflect().Type(), nil
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

// parseConfigStatus parses each xds config status to string
func parseConfigStatus(xdsConfig []*envoy_service_status_v2.PerXdsConfig) []string {
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
func printOutResponse(response *envoy_service_status_v2.ClientStatusResponse, fileName string, visualization bool) error {
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
			configStatus := parseConfigStatus(config.GetXdsConfig())
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

		if fileName == "" {
			// output the configuration to stdout by default
			fmt.Println("Detailed Config:")
			fmt.Println(string(out))
		} else {
			// write the configuration to the file
			f, err := os.Create(fileName)
			if err != nil {
				return err
			}
			defer f.Close()
			_, err = f.Write(out)
			if err != nil {
				return err
			}
			fmt.Printf("Config has been saved to %v\n", fileName)
		}

		// call visualize to enable visualization
		if visualization {
			if err := visualize(out); err != nil {
				return err
			}
		}
	}
	return nil
}

// visualize calls parseXdsRelationship and use the result to visualize
func visualize(config []byte) error {
	graphData, err := parseXdsRelationship(config)
	if err != nil {
		return err
	}
	dot, err := generateGraph(graphData)
	if err != nil {
		return err
	}

	url := "http://dreampuf.github.io/GraphvizOnline/#" + dot
	if err := openBrowser(url); err != nil {
		return err
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

// parseXdsRelationship parses relationship between xds and stores them in GraphData
func parseXdsRelationship(js []byte) (GraphData, error) {
	var data map[string]interface{}
	err := json.Unmarshal(js, &data)
	if err != nil {
		return GraphData{}, err
	}
	lds := make(map[string]string)
	rds := make(map[string]string)
	cds := make(map[string]string)
	ldsToRds := make(map[string]*treeset.Set)
	rdsToCds := make(map[string]*treeset.Set)

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
					break
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
					break
				case "clusterConfig":
					for _, clusters := range value.(map[string]interface{}) {
						for idx, cluster := range clusters.([]interface{}) {
							name := cluster.(map[string]interface{})["cluster"].(map[string]interface{})["name"].(string)
							id := "CDS" + strconv.Itoa(idx)
							cds[name] = id
						}
					}
					break
				}
			}
		}
	}

	gData := GraphData{
		nodes:     []map[string]string{lds, rds, cds},
		relations: []map[string]*treeset.Set{ldsToRds, rdsToCds},
	}

	return gData, nil
}

// generateGraph generates dot string based on GraphData
func generateGraph(data GraphData) (string, error) {
	graphAst, err := gographviz.ParseString(`digraph G {}`)
	if err != nil {
		return "", err
	}
	graph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, graph); err != nil {
		return "", err
	}

	for _, xDS := range data.nodes {
		for name, node := range xDS {
			if err := graph.AddNode("G", `\"`+name+`\"`, map[string]string{"label": node}); err != nil {
				return "", err
			}
		}
	}
	for _, relations := range data.relations {
		for src, set := range relations {
			for _, dst := range set.Values() {
				if err := graph.AddEdge(`\"`+src+`\"`, `\"`+dst.(string)+`\"`, true, nil); err != nil {
					return "", err
				}
			}
		}
	}

	return graph.String(), nil
}

// openBrowser opens url in browser based on platform
func openBrowser(url string) error {
	var err error
	switch runtime.GOOS {
	case "linux":
		err = exec.Command("xdg-open", url).Start()
		break
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
		break
	case "darwin":
		err = exec.Command("open", url).Start()
		break
	default:
		err = fmt.Errorf("unsupported platform")
	}
	if err != nil {
		return err
	}
	return nil
}
