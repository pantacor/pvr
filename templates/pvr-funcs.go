package templates

import "reflect"

var pvrFuncMap = map[string]interface{}{
	"pvrSliceIndex": func(content []interface{}, i int) interface{} {
		if len(content) > i {
			return content[i]
		}
		return nil
	},
	"pvrSliceFrom": func(content []interface{}, from int) interface{} {
		if len(content) > from {
			return content[from:]
		}
		return nil
	},
	"pvrSliceTo": func(content []interface{}, to int) interface{} {
		if len(content) >= to {
			return content[:to]
		}
		return nil
	},
	"pvrIsSlice": func(content interface{}) bool {
		// nil is not a slice for us
		if content == nil {
			return false
		}
		val := reflect.ValueOf(content)
		if val.Kind() == reflect.Array {
			return true
		}
		return false
	},
}

func PvrFuncMap() map[string]interface{} {
	sfm := make(map[string]interface{}, len(pvrFuncMap))
	for k, v := range pvrFuncMap {
		sfm[k] = v
	}
	return sfm
}
