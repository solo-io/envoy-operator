package downward

import (
	"io"
	"os"

	envoy_config_v2 "github.com/envoyproxy/go-control-plane/envoy/config/bootstrap/v2"
	"github.com/gogo/protobuf/jsonpb"
)

func TransformFiles(in, out string) error {
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
	return Transform(inreader, outwriter)

}

func Transform(in io.Reader, out io.Writer) error {
	api := RetrieveDownwardAPI()
	interpolator := NewInterpolator()

	var bootstrapConfig envoy_config_v2.Bootstrap
	var unmarshaler jsonpb.Unmarshaler
	err := unmarshaler.Unmarshal(in, &bootstrapConfig)

	if err != nil {
		return err
	}

	// interpolate the ID templates:
	bootstrapConfig.Node.Cluster, err = interpolator.InterpolateString(bootstrapConfig.Node.Cluster, api)
	if err != nil {
		return err
	}

	bootstrapConfig.Node.Id, err = interpolator.InterpolateString(bootstrapConfig.Node.Id, api)
	if err != nil {
		return err
	}
	var marshaller jsonpb.Marshaler

	return marshaller.Marshal(out, &bootstrapConfig)
}
