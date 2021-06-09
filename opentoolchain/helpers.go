package opentoolchain

import "encoding/json"

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
