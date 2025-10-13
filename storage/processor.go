package storage

import (
	"bytes"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"

	"github.com/HugoSmits86/nativewebp"
	"github.com/nfnt/resize"
)

func processImage(r io.Reader, opts UploadOptions) (io.ReadSeeker, int64, string, error) {
	img, _, err := image.Decode(r)
	if err != nil {
		return nil, 0, "", fmt.Errorf("error decode : %v", err)
	}

	// Resize 
	if opts.ResizeWidth > 0 || opts.ResizeHeight > 0 {
		w := uint(opts.ResizeWidth)
		h := uint(opts.ResizeHeight)
		img = resize.Resize(w, h, img, resize.Lanczos3)
	}

	// Encode into WebP
	buf := &bytes.Buffer{}
	err = nativewebp.Encode(buf, img, nil)
	if err != nil {
		return nil, 0, "", fmt.Errorf("error encode webp : %v", err)
	}

	data := buf.Bytes()
	return bytes.NewReader(data), int64(len(data)), "image/webp", nil
}
