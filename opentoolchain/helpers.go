package opentoolchain

import (
	"encoding/json"
)

func getStringPtr(s string) *string {
	val := s
	return &val
}

func getBoolPtr(b bool) *bool {
	val := b
	return &val
}

func dbgPrint(data interface{}) string {
	dataJSON, _ := json.MarshalIndent(data, "", "  ")
	return string(dataJSON)
}

func expandStringList(list []interface{}) []string {
	vs := make([]string, 0, len(list))
	for _, v := range list {
		val, ok := v.(string)
		if ok && val != "" {
			vs = append(vs, val)
		}
	}
	return vs
}

// compares source map keys or array of strings against target map keys
// returns a list of matched keys and new keys
func getKeyDiff(targetMap map[string]interface{}, source interface{}) (matchedKeys, newKeys []interface{}) {
	if m, ok := source.(map[string]interface{}); ok {
		for k := range m {
			if _, ok := targetMap[k]; ok {
				matchedKeys = append(matchedKeys, k)
				continue
			}

			newKeys = append(newKeys, k)
		}
	}

	if arr, ok := source.([]interface{}); ok {
		for _, k := range arr {
			if _, ok := targetMap[k.(string)]; ok {
				matchedKeys = append(matchedKeys, k)
				continue
			}

			newKeys = append(newKeys, k)
		}
	}

	return matchedKeys, newKeys
}
