package downward

import (
	"io"
	"os"
)

func TransformFiles(in, out string) error {
	inreader, err := os.Open(in)
	if err != nil {
		return err
	}
	defer inreader.Close()

	outwriter, err := os.Create(in)
	if err != nil {
		return err
	}
	defer outwriter.Close()
	return Transform(inreader, outwriter)

}

func Transform(in io.Reader, out io.Writer) error {
	api := RetrieveDownwardAPI()
	interpolator := NewInterpolator()
	return interpolator.InterpolateIO(in, out, api)
}
