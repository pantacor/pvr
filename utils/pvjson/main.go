package pvjson

import (
	"bytes"
	"encoding/json"

	cjson "github.com/gibson042/canonicaljson-go"
)

type MarshalOptions struct {
	Canonical bool
}

func Marshal(v interface{}, opts ...MarshalOptions) ([]byte, error) {
	var options MarshalOptions
	if len(opts) == 0 {
		options = MarshalOptions{Canonical: false}
	}
	if len(opts) > 0 {
		options = opts[0]
	}

	if options.Canonical {
		return cjson.Marshal(v)
	}

	return json.MarshalIndent(v, "", "    ")
}

func Unmarshal(data []byte, v interface{}) error {
	decoder := json.NewDecoder(bytes.NewBuffer(data))
	decoder.UseNumber()

	if err := decoder.Decode(v); err != nil {
		return err
	}

	return nil
}
