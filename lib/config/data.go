package config

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"

	"msh/lib/errco"
)

// loadIcon return server logo (base-64 encoded and compressed).
// If image is missing or error, the default image is returned
func loadIcon(serverDirPath string) (string, *errco.Error) {
	// get the path of the user specified server icon
	userIconPath := filepath.Join(serverDirPath, "server-icon-frozen.png")

	// if user specified icon exists: load the user specified server icon
	if _, err := os.Stat(userIconPath); !os.IsNotExist(err) {
		// use a decoder to read and then an encoder to compress the image data

		buff := &bytes.Buffer{}
		enc := &png.Encoder{CompressionLevel: -3} // -3: best compression

		// open file
		f, err := os.Open(userIconPath)
		if err != nil {
			return defaultServerIcon, errco.NewErr(errco.LOAD_ICON_ERROR, errco.LVL_D, "loadIcon", err.Error())
		}
		defer f.Close()

		// decode png
		pngIm, err := png.Decode(f)
		if err != nil {
			return defaultServerIcon, errco.NewErr(errco.LOAD_ICON_ERROR, errco.LVL_D, "loadIcon", err.Error())
		}

		// return if image is not 64x64
		if pngIm.Bounds().Max != image.Pt(64, 64) {
			return defaultServerIcon, errco.NewErr(errco.LOAD_ICON_ERROR, errco.LVL_D, "loadIcon", "incorrect server-icon-frozen.png size. Current size: "+fmt.Sprint(pngIm.Bounds().Max.X)+"x"+fmt.Sprint(pngIm.Bounds().Max.Y))
		}

		// encode png
		err = enc.Encode(buff, pngIm)
		if err != nil {
			return defaultServerIcon, errco.NewErr(errco.LOAD_ICON_ERROR, errco.LVL_D, "loadIcon", err.Error())
		}

		// return user specified server icon as base64 encoded string
		return base64.RawStdEncoding.EncodeToString(buff.Bytes()), nil
	}

	// return default server icon (user specified server icon not found)
	return defaultServerIcon, nil
}

// defaultServerIcon is the msh logo base-64 encoded
const defaultServerIcon string = "" +
	"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"
