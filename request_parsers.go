package web_request_readers

import (
	"encoding/json"
	"github.com/stretchr/goweb/context"
	"github.com/stretchr/objx"
	"io/ioutil"
	"strconv"
)

func ParseParams(ctx context.Context) (objx.Map, error) {
	if params, ok := ctx.Data()["params"]; ok {
		// We've already parsed this request, so return the cached
		// parameters.
		return params.(objx.Map), nil
	}
	request := ctx.HttpRequest()
	response := make(objx.Map)
	switch request.Header.Get("Content-Type") {
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
	case "multipart/form-data":
		request.ParseMultipartForm()
		fallthrough
	default:
		fallthrough
	case "application/x-www-form-urlencoded":
		request.ParseForm()
		for index, values := range request.Form {
			if len(values) == 1 {
				// Okay, so, here's how this works.  I hate just
				// assuming that there's only one value when I'm
				// reading a form, so I always end up testing the
				// length, which adds boilerplate code.  I want my
				// param parser to handle that case, so instead of
				// always adding a slice of values, I'm only adding
				// the single value if the length of the slice is 1.
				response.Set(index, values[0])
			} else {
				response.Set(index, values)
			}
		}
	}
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
