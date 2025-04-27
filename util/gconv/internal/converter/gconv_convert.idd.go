package converter

import (
	"encoding/json"
	"fmt"

	"github.com/gogf/gf/v2/util/gconv/internal/localinterface"
)

// iddConvertDiy idd convert diy
func (c *Converter) iddConvertDiy(in doConvertInput, option ConvertOption) (any, error) {
	switch in.ToTypeName {
	// gorm.io/datatypes.JSON
	case "datatypes.JSON", "*datatypes.JSON":
		return iddToDatatypesJSON(in.FromValue), nil
	// gorm.io/datatypes.JSONMap
	case "datatypes.JSONMap", "*datatypes.JSONMap":
		return c.Map(in.FromValue, option.MapOption)
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
		if f, ok := vv.(localinterface.IBytes); ok {
			return f.Bytes()
		}
		if b, err := json.Marshal(v); err == nil {
			return b
		}
		return nil
	}
}
