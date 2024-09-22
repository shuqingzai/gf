// Copyright GoFrame Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package goai

type ConfigPkgPathPattern int

const (
	// ConfigPkgPathPatternLast uses the last part of package path for schema name.
	//
	// Eg: github.com/gogf/gf/api/v1/user.GetReq -> user.GetReq
	ConfigPkgPathPatternLast ConfigPkgPathPattern = iota

	// ConfigPkgPathPatternFull uses full package path for schema name.
	//
	// Eg: github.com/gogf/gf/api/v1/user.GetReq -> github.com/gogf/gf/api/v1/user.GetReq
	ConfigPkgPathPatternFull

	// ConfigPkgPathPatternIgnoreModule ignores the module name of package path for schema name.
	//
	// Eg: github.com/gogf/gf/api/v1/user.GetReq -> api/v1/user.GetReq
	// ConfigPkgPathPatternIgnoreModule

	// ConfigPkgPathPatternIgnoreModuleAPI ignores the module name and "api" of package path for schema name.
	//
	// Because the APIs in the GoFrame framework are placed under the api package by default, the api package name is ignored here.
	//
	// Eg: github.com/gogf/gf/api/v1/user.GetReq -> v1/user.GetReq
	// ConfigPkgPathPatternIgnoreModuleAPI

	// ConfigPkgPathPatternCustomLastPartLen uses the last part of package path for schema name with custom length.
	//
	// needs to be set in [Config.PkgPathParts]
	//
	// Eg: github.com/gogf/gf/api/v1/user.GetReq && Config.PkgPathParts = 2 -> v1/user.GetReq
	ConfigPkgPathPatternCustomLastPartLen
)

// Config provides extra configuration feature for OpenApiV3 implements.
type Config struct {
	ReadContentTypes        []string    // ReadContentTypes specifies the default MIME types for consuming if MIME types are not configured.
	WriteContentTypes       []string    // WriteContentTypes specifies the default MIME types for producing if MIME types are not configured.
	CommonRequest           interface{} // Common request structure for all paths.
	CommonRequestDataField  string      // Common request field name to be replaced with certain business request structure. Eg: `Data`, `Request.`.
	CommonResponse          interface{} // Common response structure for all paths.
	CommonResponseDataField string      // Common response field name to be replaced with certain business response structure. Eg: `Data`, `Response.`.
	// IgnorePkgPath is used for ignoring package path in schema name.
	//
	// Deprecated, use PkgPathPattern instead.
	IgnorePkgPath bool
	// PkgPathPattern is used for customizing package path in schema name.
	PkgPathPattern ConfigPkgPathPattern
	// PkgPathPartLength is used for customizing package path in schema name with custom length.
	//
	// It is used when PkgPathPattern is ConfigPkgPathPatternCustomLastPartLen.
	PkgPathPartLength int
}

// fillWithDefaultValue fills configuration object of `oai` with default values if these are not configured.
func (oai *OpenApiV3) fillWithDefaultValue() {
	if oai.OpenAPI == "" {
		oai.OpenAPI = `3.0.0`
	}
	if len(oai.Config.ReadContentTypes) == 0 {
		oai.Config.ReadContentTypes = defaultReadContentTypes
	}
	if len(oai.Config.WriteContentTypes) == 0 {
		oai.Config.WriteContentTypes = defaultWriteContentTypes
	}
}
