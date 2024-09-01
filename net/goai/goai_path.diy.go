package goai

import (
	"net/http"
	"strconv"
	"strings"
)

// fillMoreResponse fills the response object from the given meta map.
func (oai *OpenApiV3) fillMoreResponse(metaMap map[string]string, resp Responses) {
	oai.fillSuccessStatusCodeResponse(metaMap, resp)
	oai.fillErrorStatusCodeResponse(metaMap, resp)
}

// fillSuccessStatusCodeResponse fills the success status code response object from the given meta map.
//
// The success status code is specified by the tag "successStatusCode".
// The value of the tag is a string containing multiple status codes separated by commas.
// If the status code is not in the 2xx range, it will be ignored.
// If the status code is not defined in the response object, it will be replaced by the default success response object.
func (oai *OpenApiV3) fillSuccessStatusCodeResponse(metaMap map[string]string, resp Responses) {
	if _, ok := metaMap["successStatusCode"]; !ok {
		return
	}
	vv := strings.TrimSpace(metaMap["successStatusCode"])
	if vv == "" {
		return
	}

	// 取出用户定义的成功响应对象
	successResp, ok := resp[responseOkKey]
	if !ok {
		return
	}

	// 支持配置多个成功状态码，以逗号分隔
	successStatusCodes := strings.Split(vv, ",")
	successSeen := make(map[string]struct{})
	for _, code := range successStatusCodes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}

		// 成功状态码必须是 2xx 范围内的状态码
		if codeValue, _ := strconv.Atoi(code); codeValue < 200 || codeValue > 299 {
			continue
		}

		// 自定义成功状态码时，替换默认的响应状态码
		resp[code] = successResp
		successSeen[code] = struct{}{}
	}

	// 如果只有一个成功状态码，那么将默认的成功状态码替换为用户定义的成功状态码
	// 如: 创建成功的状态码为 201，那么将默认的 200 替换为 201
	if len(successSeen) == 1 {
		if _, ok := successSeen[responseOkKey]; !ok {
			delete(resp, responseOkKey)
		}
	}
}

// fillErrorStatusCodeResponse fills the error status code response object from the given meta map.
//
// The error status code is specified by the tag "errorStatusCode".
// The value of the tag is a string containing multiple status codes separated by commas.
// If the status code is not in the 4xx or 5xx range, it will be ignored.
// If the status code is not defined in the response object, it will be replaced by the default error response object.
func (oai *OpenApiV3) fillErrorStatusCodeResponse(metaMap map[string]string, resp Responses) {
	if _, ok := metaMap["errorStatusCode"]; !ok {
		return
	}
	vv := strings.TrimSpace(metaMap["errorStatusCode"])
	if vv == "" {
		return
	}

	errorStatusCodes := strings.Split(vv, ",")
	for _, code := range errorStatusCodes {
		code = strings.TrimSpace(code)
		if code == "" {
			continue
		}

		// 有效的状态码必须是 4xx 或 5xx 范围内的状态码
		codeValue, _ := strconv.Atoi(code)
		if codeValue < 400 || codeValue > 599 {
			continue
		}

		// 先尝试使用 状态码 作为 key 查找对应的响应对象
		if rv, ok := oai.Components.Responses[code]; ok {
			resp[code] = rv
			continue
		}

		// 使用状态码描述文本作为 key 查找对应的响应对象
		statusText := http.StatusText(codeValue)
		if statusText == "" {
			continue
		}
		// 标准的状态码描述文本
		// 400 Bad Request
		// 500 Internal Server Error
		if rv, ok := oai.Components.Responses[statusText]; ok {
			resp[code] = rv
			continue
		}
		// 去掉空格的状态码描述文本
		//
		// 400 BadRequest
		// 500 InternalServerError
		if rv, ok := oai.Components.Responses[strings.ReplaceAll(statusText, " ", "")]; ok {
			resp[code] = rv
		}
	}
}
