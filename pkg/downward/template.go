package downward

import (
	"github.com/pkg/errors"
	"bytes"
	"io"
	"io/ioutil"
	"text/template"
)

type Interpolator interface {
	InterpolateIO(in io.Reader, out io.Writer, data DownwardAPI) error
	Interpolate(tmpl string, out io.Writer, data DownwardAPI) error
	InterpolateString(string, DownwardAPI) (string, error)
}

func NewInterpolator() Interpolator {
	return &interpolator{}
}

type interpolator struct{}

func (i *interpolator) InterpolateIO(in io.Reader, out io.Writer, data DownwardAPI) error {
	inbyte, err := ioutil.ReadAll(in)
	if err != nil {
		return errors.Wrap(err, "reading input")
	}

	return i.Interpolate(string(inbyte), out, data)
}

func (*interpolator) Interpolate(tmpl string, out io.Writer, data DownwardAPI) error {
	t, err := template.New("template").Option("missingkey=zero").Parse(tmpl)
	if err != nil {
		return err
	}
	err = t.Execute(out, data)
	if err != nil {
		return errors.Wrap(err, "executing template")
	}
	return nil
}

func (i *interpolator) InterpolateString(tmpl string, data DownwardAPI) (string, error) {
	var b bytes.Buffer
	err := i.Interpolate(tmpl, &b, data)
	if err != nil {
		return "", errors.Wrap(err, "interpolating string")
	}
	return b.String(), nil
}
