package main

import (
	"encoding/json"
	"fmt"
	envoy_type_matcher "github.com/envoyproxy/go-control-plane/envoy/type/matcher"
	"github.com/ghodss/yaml"
	"io/ioutil"
	"path/filepath"
)

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
	fmt.Println(string(js))

	var data map[string]interface{}
	err = json.Unmarshal(js, &data)
	if err != nil {
		return fmt.Errorf("%v", err)
	}

	for _, n := range data["node_matchers"].([]interface{}) {
		nMap := n.(map[string]interface{})
		x := &envoy_type_matcher.NodeMatcher{}

		if metadatas, ok := nMap["node_metadatas"]; ok {
			var node_metadatas []*envoy_type_matcher.StructMatcher
			for _, metadata := range metadatas.([]interface{}) {
				struct_matcher := &envoy_type_matcher.StructMatcher{}
				mMap := metadata.(map[string]interface{})
				if paths, ok := mMap["path"]; ok {
					for _, path := range paths.([]interface{}) {
						pathMap := path.(map[string]interface{})
						if key, ok := pathMap["key"]; ok {
							segment := &envoy_type_matcher.StructMatcher_PathSegment{
								Segment: &envoy_type_matcher.StructMatcher_PathSegment_Key{Key: key.(string)},
							}
							struct_matcher.Path = append(struct_matcher.Path, segment)
						} else {
							return fmt.Errorf("missing key in node_metadatas")
						}
					}
				} else {
					return fmt.Errorf("missing path in node_metadatas")
				}

				if value, ok := mMap["value"]; ok {
					valueMap := value.(map[string]interface{})
					if string_match, ok := valueMap["string_match"]; ok {
						string_matcher := &envoy_type_matcher.ValueMatcher_StringMatch{}
						matchMap := string_match.(map[string]interface{})
						for k, v := range matchMap {
							switch k {
							case "exact":
								string_matcher.StringMatch = &envoy_type_matcher.StringMatcher{MatchPattern: &envoy_type_matcher.StringMatcher_Exact{Exact: v.(string)}}
							case "prefix":
								string_matcher.StringMatch = &envoy_type_matcher.StringMatcher{MatchPattern: &envoy_type_matcher.StringMatcher_Prefix{Prefix: v.(string)}}
							case "suffix":
								string_matcher.StringMatch = &envoy_type_matcher.StringMatcher{MatchPattern: &envoy_type_matcher.StringMatcher_Suffix{Suffix: v.(string)}}
							case "regex":
								string_matcher.StringMatch = &envoy_type_matcher.StringMatcher{MatchPattern: &envoy_type_matcher.StringMatcher_Regex{Regex: v.(string)}}
							case "safe_regex":
								string_matcher.StringMatch = &envoy_type_matcher.StringMatcher{MatchPattern: &envoy_type_matcher.StringMatcher_SafeRegex{SafeRegex: &envoy_type_matcher.RegexMatcher{}}} //TODO: fill this field
							}
							struct_matcher.Value = &envoy_type_matcher.ValueMatcher{MatchPattern: string_matcher}
						}
					}
				} else {
					return fmt.Errorf("missing value in node_metadatas")
				}
				node_metadatas = append(node_metadatas, struct_matcher)
			}
			x.NodeMetadatas = node_metadatas
		} else {
			return fmt.Errorf("missing node_metadatas in node_matcher yaml")
		}

		if node_id, ok := nMap["node_id"]; ok {
			node_idMap := node_id.(map[string]interface{})

			if string_match, ok := node_idMap["string_match"]; ok {
				string_matcher := &envoy_type_matcher.StringMatcher{}
				matchMap := string_match.(map[string]interface{})
				for k, v := range matchMap {
					switch k {
					case "exact":
						string_matcher.MatchPattern = &envoy_type_matcher.StringMatcher_Exact{Exact: v.(string)}
					case "prefix":
						string_matcher.MatchPattern = &envoy_type_matcher.StringMatcher_Prefix{Prefix: v.(string)}
					case "suffix":
						string_matcher.MatchPattern = &envoy_type_matcher.StringMatcher_Suffix{Suffix: v.(string)}
					case "regex":
						string_matcher.MatchPattern = &envoy_type_matcher.StringMatcher_Regex{Regex: v.(string)}
					case "safe_regex":
						string_matcher.MatchPattern = &envoy_type_matcher.StringMatcher_SafeRegex{&envoy_type_matcher.RegexMatcher{}} //TODO: fill this field
					}
				}
				x.NodeId = string_matcher
			}
		}
		*nms = append(*nms, x)
	}
	return nil
}
