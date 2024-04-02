package gconv

import (
	"fmt"
)

// iddConvertDiy idd convert diy
func iddConvertDiy(in doConvertInput) (any, error) {
	switch in.ToTypeName {
	// gorm.io/datatypes.JSON
	case "datatypes.JSON":
		return Bytes(in.FromValue), nil
	// gorm.io/datatypes.JSONMap
	case "datatypes.JSONMap":
		return Map(in.FromValue), nil
	}
	return nil, fmt.Errorf("unsupported cast type: %T To %s", in.FromValue, in.ToTypeName)
}
