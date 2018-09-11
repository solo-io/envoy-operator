package downward

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"os"

	envoy_config_v2 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/gogo/protobuf/jsonpb"
	"gopkg.in/yaml.v2"
)

type Transformer struct {
	transformations []func(*envoy_config_v2.Bootstrap) error
}

func NewTransformer() *Transformer {
	return &Transformer{
		transformations: []func(*envoy_config_v2.Bootstrap) error{TransformConfigTemplates},
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

	var bootstrapConfig envoy_config_v2.Bootstrap
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

func TransformConfigTemplates(bootstrapConfig *envoy_config_v2.Bootstrap) error {

	api := RetrieveDownwardAPI()
	interpolator := NewInterpolator()

	var err error

	// interpolate the ID templates:
	bootstrapConfig.Node.Cluster, err = interpolator.InterpolateString(bootstrapConfig.Node.Cluster, api)
	if err != nil {
		return err
	}

	bootstrapConfig.Node.Id, err = interpolator.InterpolateString(bootstrapConfig.Node.Id, api)
	if err != nil {
		return err
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
