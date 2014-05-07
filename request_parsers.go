package web_request_readers

import (
	"encoding/json"
	codec_services "github.com/stretchr/codecs/services"
	"github.com/stretchr/goweb/context"
	"github.com/stretchr/objx"
	"io/ioutil"
	"strconv"
	"errors"
)

var multipartMem int64 = 2 << 20 * 10

func MultipartMem() int64 {
	return multipartMem
}

func SetMultipartMem(mem int64) {
	multipartMem = mem
}

// ConvertMSIToObjxMap recursively converts map[string]interface{}
// values to objx.Map.  This is designed around the return types of
// json.Unmarshal, so it may not work for non-json data.
func ConvertMSIToObjxMap(value interface{}) interface{} {
	switch src := value.(type) {
	case map[string]interface{}:
		for key, val := range src {
			src[key] = ConvertMSIToObjxMap(val)
		}
		return objx.Map(src)
	case []interface{}:
		for index, val := range src {
			src[index] = ConvertMSIToObjxMap(val)
		}
		return src
	}
	return value
}

// ParseParams will parse parameters out of a request body.  The
// result will be an objx.Map of values, or an error if something
// unexpected happened.  All map[string]interface{} values are
// converted to objx.Map before returning.
func ParseParams(ctx context.Context) (objx.Map, error) {
	val, err := ParseBody(ctx)
	if err != nil {
		return nil, err
	}
	if val == nil {
		return nil, nil
	}
	params, ok := val.(objx.Map)
	if !ok {
		return nil, errors.New("Cannot use non-map body as params")
	}
	return params, nil
}

// ParseBody will parse a request body, regardless of type.  The body
// could be a json array, and this will return it properly.  All
// map[string]interface{} values are converted to objx.Map before
// returning.
func ParseBody(ctx context.Context) (interface{}, error) {
	if params, ok := ctx.Data()["params"]; ok {
		// We've already parsed this request, so return the cached
		// parameters.
		return params, nil
	}
	request := ctx.HttpRequest()
	var response interface{}
	contentType, _ := codec_services.ParseContentType(request.Header.Get("Content-Type"))
	var mimeType string
	if contentType != nil {
		mimeType = contentType.MimeType
	}
	switch mimeType {
	case "text/json":
		fallthrough
	case "application/json":
		body, err := ioutil.ReadAll(request.Body)
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(body, &response); err != nil {
			return nil, err
		}
	default:
		fallthrough
	case "application/x-www-form-urlencoded":
		fallthrough
	case "multipart/form-data":
		params := make(objx.Map)
		request.ParseMultipartForm(MultipartMem())
		if request.MultipartForm != nil {
			params.Set("files", request.MultipartForm.File)
			for key, values := range request.MultipartForm.Value {
				if len(values) == 1 {
					params.Set(key, values[0])
				} else {
					params.Set(key, values)
				}
			}
		}
		for index, values := range request.Form {
			if len(values) == 1 {
				// Okay, so, here's how this works.  I hate just
				// assuming that there's only one value when I'm
				// reading a form, so I always end up testing the
				// length, which adds boilerplate code.  I want my
				// param parser to handle that case, so instead of
				// always adding a slice of values, I'm only adding
				// the single value if the length of the slice is 1.
				params.Set(index, values[0])
			} else {
				params.Set(index, values)
			}
		}
		response = params
	}
	response = ConvertMSIToObjxMap(response)
	ctx.Data().Set("params", response)
	return response, nil
}

// ParsePage reads "page" and "page_size" from a set of parameters and
// parses them into offset and limit values.
//
// It assumes that the params will be coming from a set of query
// parameters, and that the page and page_size values will be of type
// []string, since that's the default for query parameters.  It will
// simply read the first value from each slice, ignoring extra values.
func ParsePage(params objx.Map, defaultPageSize int) (offset, limit int, err error) {
	limit = defaultPageSize

	sizeVal, sizeOk := params["page_size"]
	pageVal, pageOk := params["page"]

	if sizeOk {
		sizeSlice := sizeVal.([]string)
		sizeStr := sizeSlice[0]
		var pageSize int
		pageSize, err = strconv.Atoi(sizeStr)
		if err != nil {
			return
		}
		limit = pageSize
	}

	if pageOk {
		pageSlice := pageVal.([]string)
		pageStr := pageSlice[0]
		var page int
		page, err = strconv.Atoi(pageStr)
		if err != nil {
			return
		}
		offset = (page - 1) * limit
	}

	return
}
