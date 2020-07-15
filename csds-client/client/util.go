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

type GraphData struct {
	nodes     []map[string]string
	relations []map[string]*treeset.Set
}

// printOutResponse posts process response and print
func printOutResponse(response *envoy_service_status_v2.ClientStatusResponse, fileName string) {
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
			fmt.Printf("%-50s %-30s %-30s \n", id, xdsType, configFile)
		} else {
			// parse response to json
			// format the json and resolve google.protobuf.Any types
			m := protojson.MarshalOptions{Multiline: true, Indent: "  ", Resolver: &TypeResolver{}}
			out, err := m.Marshal(response)
			if err != nil {
				fmt.Printf("%v", err)
			}
			if err := parseXdsRelationship(out); err != nil {
				fmt.Printf("%v\n",err)
				return
			}
			if fileName == "" {
				// output the configuration to stdout by default
				fmt.Printf("%-50s %-30s %-30s \n", id, xdsType, configFile)
				fmt.Println(string(out))
			} else {
				// write the configuration to the file
				configFile = fileName
				f, err := os.Create(configFile)
				if err != nil {
					fmt.Println(err)
				}
				defer f.Close()
				_, err = f.Write(out)
				if err != nil {
					fmt.Printf("%v", err)
				}
				fmt.Printf("%-50s %-30s %-30s \n", id, xdsType, configFile)
			}
		}
	}
}

func parseXdsRelationship(js []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(js, &data)
	if err != nil {
		return err
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
							ldsToRds[id] = rdsSet
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
							rdsToCds[id] = cdsSet
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

	for lds, rdsSet := range ldsToRds {
		rdsIdSet := treeset.NewWithStringComparator()
		for _, name := range rdsSet.Values() {
			rdsIdSet.Add(rds[name.(string)])
		}
		ldsToRds[lds] = rdsIdSet
	}
	for rds, cdsSet := range rdsToCds {
		cdsIdSet := treeset.NewWithStringComparator()
		for _, name := range cdsSet.Values() {
			cdsIdSet.Add(cds[name.(string)])
		}
		rdsToCds[rds] = cdsIdSet
	}

	gData := GraphData{
		nodes:     []map[string]string{lds, rds, cds},
		relations: []map[string]*treeset.Set{ldsToRds, rdsToCds},
	}
	if err := generateGraph(gData); err != nil {
		return err
	}
	fmt.Printf("%v\n%v\n%v\n", lds, rds, cds)
	fmt.Printf("%v\n%v\n", ldsToRds, rdsToCds)
	return nil
}

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

func generateGraph(data GraphData) error {
	graphAst, err := gographviz.ParseString(`digraph G {}`)
	if err != nil {
		return err
	}
	graph := gographviz.NewGraph()
	if err := gographviz.Analyse(graphAst, graph); err != nil {
		return err
	}

	for _, xDS := range data.nodes {
		for _, node := range xDS {
			if err := graph.AddNode("G", node, nil); err != nil {
				return err
			}
		}
	}
	for _, relations := range data.relations {
		for src, set := range relations {
			for _, dst := range set.Values() {
				if err := graph.AddEdge(src, dst.(string), true, nil); err != nil {
					return err
				}
			}
		}
	}

	fmt.Printf("%v\n", graph.String())

	url := "http://dreampuf.github.io/GraphvizOnline/#" + graph.String()
	if err := openBrowser(url); err != nil {
		return err
	}
	return nil
}
