package storage

import (
	"bytes"
	"image"
	"io"
	"image/jpeg"		
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"

	"github.com/disintegration/imaging"
)

func processImage(r io.Reader, opts UploadOptions) (io.ReadSeeker, int64, string, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, 0, "", err
	}

	if opts.ResizeWidth > 0 || opts.ResizeHeight > 0 {
		img = imaging.Fit(img, opts.ResizeWidth, opts.ResizeHeight, imaging.Lanczos)
	}

	buf := &bytes.Buffer{}
	q := opts.Quality
	if q == 0 {
		q = 85
	}
	if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: q}); err != nil {
		return nil, 0, "", err
	}

	return bytes.NewReader(buf.Bytes()), int64(buf.Len()), "image/jpeg", nil
}
