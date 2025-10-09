package storage

import (
	"bytes"
	"fmt"
	"image"
  _ "image/gif"
	"image/jpeg"
	"image/png"
	"io"

	"github.com/disintegration/imaging"
)

func processImage(r io.Reader, opts UploadOptions) (io.ReadSeeker, int64, string, error) {
    limited := io.LimitReader(r, 25<<20)

    data, err := io.ReadAll(limited)
    if err != nil {
        return nil, 0, "", fmt.Errorf("read image: %w", err)
    }

    img, format, err := image.Decode(bytes.NewReader(data))
    if err != nil {
        return nil, 0, "", fmt.Errorf("decode image: %w", err)
    }

    if opts.ResizeWidth > 0 || opts.ResizeHeight > 0 {
        img = imaging.Fit(img, opts.ResizeWidth, opts.ResizeHeight, imaging.Lanczos)
    }

    buf := &bytes.Buffer{}
    q := opts.Quality
    if q == 0 { q = 85 }

    if format == "png" {
        if err := png.Encode(buf, img); err != nil {
            return nil, 0, "", fmt.Errorf("encode png: %w", err)
        }
        return bytes.NewReader(buf.Bytes()), int64(buf.Len()), "image/png", nil
    }

    if err := jpeg.Encode(buf, img, &jpeg.Options{Quality: q}); err != nil {
        return nil, 0, "", fmt.Errorf("encode jpeg: %w", err)
    }
    return bytes.NewReader(buf.Bytes()), int64(buf.Len()), "image/jpeg", nil
}
