package main

import (
	"encoding/json"
	"fmt"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"github.com/ghodss/yaml"
	"google.golang.org/protobuf/encoding/protojson"
	"io/ioutil"
	"path/filepath"
)

type Nodematchers struct {
}

func parseYaml(path string, nms *[]*envoy_type_matcher.NodeMatcher) error {
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
