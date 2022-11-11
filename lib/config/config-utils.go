package config

import (
	"archive/zip"
	"bytes"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/utility"
)

// IsWhitelist checks if the parameters are in config whitelist.
// (Currently this function accepts as arguments the client request packet and the client address)
func (c *Configuration) IsWhitelist(reqPacket []byte, clientAddress string) *errco.MshLog {
	// check whitelist from minecraft server config
	if c.Msh.WhitelistImport {
		var wl []model.MSWhitelist

		// read from file whitelist.json file
		// load minecraft server whitelist
		// check elements of minecraft server whitelist against request packet
		if data, err := os.ReadFile(filepath.Join(c.Server.Folder, "whitelist.json")); err != nil {
			errco.Logln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_WHITELIST_CHECK, "whitelist.json file file can't be read")
		} else if err = json.Unmarshal(data, &wl); err != nil {
			errco.Logln(errco.TYPE_WAR, errco.LVL_3, errco.ERROR_WHITELIST_CHECK, "whitelist.json file format error")
		} else {
			for _, e := range wl {
				errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "searching byte array for: %s (whitelist import enabled)", e.Name)
				if bytes.Contains(reqPacket, []byte(e.Name)) {
					errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "whitelist ok!")
					return nil
				}
			}
		}
	} else {
		// minecraft server whitelist import not enabled
		errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "minecraft server whitelist import not enabled")
	}

	// check whitelist from msh config
	if len(c.Msh.Whitelist) > 0 {
		// check client address against msh config whitelist
		errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "searching whitelist for: %s", clientAddress)
		if utility.SliceContain(clientAddress, c.Msh.Whitelist) {
			errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "whitelist ok!")
			return nil
		}

		// check elements of msh config whitelist against request packet
		for _, w := range c.Msh.Whitelist {
			errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "searching byte array for: %s", w)
			if bytes.Contains(reqPacket, []byte(w)) {
				errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "whitelist ok!")
				return nil
			}
		}

		// no match found, error
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_WHITELIST_CHECK, "msh config whitelist check failed")

	} else {
		// msh config whitelist not enabled
		errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "msh config whitelist not enabled")
		return nil
	}
}

// loadIcon tries to load user specified server icon (base-64 encoded and compressed).
// The default icon is loaded by default
func (c *Configuration) loadIcon() *errco.MshLog {
	// set default server icon
	ServerIcon = defaultServerIcon

	// get the path of the user specified server icon
	userIconPaths := []string{}
	userIconPaths = append(userIconPaths, filepath.Join(c.Server.Folder, "server-icon-frozen.png"))
	userIconPaths = append(userIconPaths, filepath.Join(c.Server.Folder, "server-icon-frozen.jpg"))

	for _, uip := range userIconPaths {
		// check if user specified icon exists
		_, err := os.Stat(uip)
		if os.IsNotExist(err) {
			// user specified server icon not found
			continue
		}

		// open file
		f, err := os.Open(uip)
		if err != nil {
			errco.Logln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ICON_LOAD, err.Error())
			continue
		}
		defer f.Close()

		// read file data
		// it's important to read all file data and store it in a variable that can be read multiple times with a io.Reader.
		// using f *os.File directly in Decode(r io.Reader) results in f *os.File readable only once.
		fdata, err := io.ReadAll(f)
		if err != nil {
			errco.Logln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ICON_LOAD, err.Error())
			continue
		}

		// decode image (try different formats)
		var img image.Image
		if img, err = png.Decode(bytes.NewReader(fdata)); err == nil {
		} else if img, err = jpeg.Decode(bytes.NewReader(fdata)); err == nil {
		} else {
			errco.Logln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ICON_LOAD, "data format invalid: %s (%s)", uip, err.Error())
			continue
		}

		// scale image to 64x64
		scaImg, d := utility.ScaleImg(img, image.Rect(0, 0, 64, 64))
		errco.Logln(errco.TYPE_INF, errco.LVL_3, errco.ERROR_NIL, "scaled %s to 64x64. (%v ms)", uip, d.Milliseconds())

		// encode image to png
		enc, buff := &png.Encoder{CompressionLevel: -3}, &bytes.Buffer{} // -3: best compression
		err = enc.Encode(buff, scaImg)
		if err != nil {
			errco.Logln(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_ICON_LOAD, err.Error())
			continue
		}

		// load user specified server icon as base64 encoded string
		ServerIcon = base64.RawStdEncoding.EncodeToString(buff.Bytes())

		// as soon as a good image is loaded, break and return
		break
	}

	return nil
}

// loadIpPorts reads server.properties server file and loads correct ports to global variables
func (c *Configuration) loadIpPorts() *errco.MshLog {
	// ListenHost remains the same
	ListenPort = c.Msh.ListenPort
	// TargetHost remains the same
	// TargetPort is extracted from server.properties

	data, err := os.ReadFile(filepath.Join(c.Server.Folder, "server.properties"))
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_CONFIG_LOAD, err.Error())
	}

	TargetPortStr, logMsh := utility.StrBetween(strings.ReplaceAll(string(data), "\r", ""), "server-port=", "\n")
	if logMsh != nil {
		return logMsh.AddTrace()
	}

	TargetPort, err = strconv.Atoi(TargetPortStr)
	if err != nil {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_CONVERSION, err.Error())
	}

	if TargetPort == c.Msh.ListenPort {
		return errco.NewLog(errco.TYPE_ERR, errco.LVL_1, errco.ERROR_CONFIG_LOAD, "TargetPort and ListenPort appear to be the same, please change one of them")
	}

	return nil
}

// getVersionInfo reads version.json from the server JAR file
// and returns minecraft server version and protocol.
// In case of error "", 0, *errco.MshLog are returned.
func (c *Configuration) getVersionInfo() (string, int, *errco.MshLog) {
	reader, err := zip.OpenReader(filepath.Join(c.Server.Folder, c.Server.FileName))
	if err != nil {
		return "", 0, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_VERSION_LOAD, err.Error())
	}
	defer reader.Close()

	for _, file := range reader.File {
		// search for version.json file
		if file.Name != "version.json" {
			continue
		}

		f, err := file.Open()
		if err != nil {
			return "", 0, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_VERSION_LOAD, err.Error())
		}
		defer f.Close()

		versionsBytes, err := io.ReadAll(f)
		if err != nil {
			return "", 0, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_VERSION_LOAD, err.Error())
		}

		var info model.VersionInfo
		err = json.Unmarshal(versionsBytes, &info)
		if err != nil {
			return "", 0, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_VERSION_LOAD, err.Error())
		}

		return info.Version, info.Protocol, nil
	}

	return "", 0, errco.NewLog(errco.TYPE_ERR, errco.LVL_3, errco.ERROR_VERSION_LOAD, "minecraft server version and protocol could not be extracted from version.json")
}
