package gconv

import (
	"encoding/json"
	"fmt"
)

// iddConvertDiy idd convert diy
func iddConvertDiy(in doConvertInput) (any, error) {
	switch in.ToTypeName {
	// gorm.io/datatypes.JSON
	case "datatypes.JSON":
		return iddToDatatypesJSON(in.FromValue), nil
	// gorm.io/datatypes.JSONMap
	case "datatypes.JSONMap":
		return Map(in.FromValue), nil
	}
	return nil, fmt.Errorf("unsupported cast type: %T To %s", in.FromValue, in.ToTypeName)
}

// iddToDatatypesJSON idd convert v to datatypes.JSON
//
// See: Bytes
func iddToDatatypesJSON(v any) []byte {
	if v == nil {
		return nil
	}
	switch vv := v.(type) {
	case string:
		return []byte(vv)
	case *string:
		if vv == nil {
			return nil
		}
		return []byte(*vv)
	case []byte:
		return vv
	case *[]byte:
		if vv == nil {
			return nil
		}
		return *vv
	case json.RawMessage:
		return vv
	case *json.RawMessage:
		if vv == nil {
			return nil
		}
		return *vv
	default:
		if f, ok := vv.(iBytes); ok {
			return f.Bytes()
		}
		if b, err := json.Marshal(v); err == nil {
			return b
		}
		return nil
	}
}
