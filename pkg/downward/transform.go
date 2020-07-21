package downward

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	structpb "github.com/golang/protobuf/ptypes/struct"

	envoy_config_bootstrap "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v3"
	"github.com/golang/protobuf/jsonpb"
	yaml "gopkg.in/yaml.v2"

	// register all top level types used in the bootstrap config
	_ "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
)

type Transformer struct {
	transformations []func(*envoy_config_bootstrap.Bootstrap) error
}

func NewTransformer() *Transformer {
	return &Transformer{
		transformations: []func(*envoy_config_bootstrap.Bootstrap) error{TransformConfigTemplates},
	}
}

func (t *Transformer) TransformFiles(in, out string) error {
	inreader, err := os.Open(in)
	if err != nil {
		return err
	}
	defer inreader.Close()

	outwriter, err := os.Create(out)
	if err != nil {
		return err
	}
	defer outwriter.Close()
	return t.Transform(inreader, outwriter)
}

func (t *Transformer) Transform(in io.Reader, out io.Writer) error {
	// first step - serialize yaml to json
	jsondata, err := getJson(in)

	if err != nil {
		return err
	}

	jsonreader := bytes.NewReader(jsondata)

	var bootstrapConfig envoy_config_bootstrap.Bootstrap
	var unmarshaler jsonpb.Unmarshaler
	err = unmarshaler.Unmarshal(jsonreader, &bootstrapConfig)

	if err != nil {
		return err
	}

	for _, transformation := range t.transformations {
		err := transformation(&bootstrapConfig)
		if err != nil {
			return err
		}
	}
	var marshaller jsonpb.Marshaler

	return marshaller.Marshal(out, &bootstrapConfig)
}

func TransformConfigTemplates(bootstrapConfig *envoy_config_bootstrap.Bootstrap) error {
	api := RetrieveDownwardAPI()
	return TransformConfigTemplatesWithApi(bootstrapConfig, api)
}

func TransformConfigTemplatesWithApi(bootstrapConfig *envoy_config_bootstrap.Bootstrap, api DownwardAPI) error {

	interpolator := NewInterpolator()

	var err error

	interpolate := func(s *string) error { return interpolator.InterpolateString(s, api) }
	// interpolate the ID templates:
	err = interpolate(&bootstrapConfig.Node.Cluster)
	if err != nil {
		return err
	}

	err = interpolate(&bootstrapConfig.Node.Id)
	if err != nil {
		return err
	}

	transformStruct(interpolate, bootstrapConfig.Node.Metadata)

	return nil
}
func transformValue(interpolate func(*string) error, v *structpb.Value) error {
	switch v := v.Kind.(type) {
	case (*structpb.Value_StringValue):
		return interpolate(&v.StringValue)
	case (*structpb.Value_StructValue):
		return transformStruct(interpolate, v.StructValue)
	case (*structpb.Value_ListValue):
		for _, val := range v.ListValue.Values {
			if err := transformValue(interpolate, val); err != nil {
				return err
			}
		}
	}
	return nil
}

func transformStruct(interpolate func(*string) error, s *structpb.Struct) error {
	if s == nil {
		return nil
	}

	for _, v := range s.Fields {
		if err := transformValue(interpolate, v); err != nil {
			return err
		}
	}
	return nil
}

func getJson(in io.Reader) ([]byte, error) {
	readbytes, err := ioutil.ReadAll(in)
	if err != nil {
		return nil, err
	}
	var body interface{}
	if err := yaml.Unmarshal(readbytes, &body); err != nil {
		return nil, err
	}
	body = convert(body)
	if b, err := json.Marshal(body); err != nil {
		return nil, err
	} else {
		return b, nil
	}
}
func convert(i interface{}) interface{} {
	switch x := i.(type) {
	case map[interface{}]interface{}:
		m2 := map[string]interface{}{}
		for k, v := range x {
			m2[k.(string)] = convert(v)
		}
		return m2
	case []interface{}:
		for i, v := range x {
			x[i] = convert(v)
		}
	}
	return i
}
