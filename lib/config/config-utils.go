package config

import (
	"archive/zip"
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/utility"
)

// InWhitelist checks if the playerName or clientAddress is in config whitelist
func (c *Configuration) InWhitelist(params ...string) *errco.Error {
	// check if whitelist is enabled
	// if empty then it is not enabled and no checks are needed
	if len(c.Msh.Whitelist) == 0 {
		errco.Logln(errco.LVL_D, "whitelist not enabled")
		return nil
	}

	errco.Logln(errco.LVL_D, "checking whitelist for: %s", strings.Join(params, ", "))

	// check if playerName or clientAddress are in whitelist
	for _, p := range params {
		if utility.SliceContain(p, c.Msh.Whitelist) {
			errco.Logln(errco.LVL_D, "playerName or clientAddress is whitelisted!")
			return nil
		}
	}

	// playerName or clientAddress not found in whitelist
	errco.Logln(errco.LVL_D, "playerName or clientAddress is not whitelisted!")
	return errco.NewErr(errco.ERROR_PLAYER_NOT_IN_WHITELIST, errco.LVL_B, "InWhitelist", "playerName or clientAddress is not whitelisted")
}

// loadIcon return server logo (base-64 encoded and compressed).
// If image is missing or error, the default image is returned
func (c *Configuration) loadIcon() (string, *errco.Error) {
	// get the path of the user specified server icon
	userIconPath := filepath.Join(c.Server.Folder, "server-icon-frozen.png")

	// check if user specified icon exists
	_, err := os.Stat(userIconPath)
	if os.IsNotExist(err) {
		// user specified server icon not found
		// return default server icon, but no error should be reported
		return defaultServerIcon, nil
	}

	// open file
	f, err := os.Open(userIconPath)
	if err != nil {
		return defaultServerIcon, errco.NewErr(errco.ERROR_ICON_LOAD, errco.LVL_D, "loadIcon", err.Error())
	}
	defer f.Close()

	// decode png
	pngIm, err := png.Decode(f)
	if err != nil {
		return defaultServerIcon, errco.NewErr(errco.ERROR_ICON_LOAD, errco.LVL_D, "loadIcon", err.Error())
	}

	// check that image is 64x64
	if pngIm.Bounds().Max != image.Pt(64, 64) {
		return defaultServerIcon, errco.NewErr(errco.ERROR_ICON_LOAD, errco.LVL_D, "loadIcon", fmt.Sprintf("incorrect server-icon-frozen.png size. Current size: %dx%d", pngIm.Bounds().Max.X, pngIm.Bounds().Max.Y))
	}

	// encode png
	enc, buff := &png.Encoder{CompressionLevel: -3}, &bytes.Buffer{} // -3: best compression
	err = enc.Encode(buff, pngIm)
	if err != nil {
		return defaultServerIcon, errco.NewErr(errco.ERROR_ICON_LOAD, errco.LVL_D, "loadIcon", err.Error())
	}

	// return user specified server icon as base64 encoded string
	return base64.RawStdEncoding.EncodeToString(buff.Bytes()), nil
}

// getIpPorts reads server.properties server file and returns the correct ports
func (c *Configuration) getIpPorts() (string, int, string, int, *errco.Error) {
	data, err := ioutil.ReadFile(filepath.Join(c.Server.Folder, "server.properties"))
	if err != nil {
		return "", -1, "", -1, errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_B, "getIpPorts", err.Error())
	}

	dataStr := strings.ReplaceAll(string(data), "\r", "")

	TargetPortStr, errMsh := utility.StrBetween(dataStr, "server-port=", "\n")
	if errMsh != nil {
		return "", -1, "", -1, errMsh.AddTrace("getIpPorts")
	}

	TargetPort, err = strconv.Atoi(TargetPortStr)
	if err != nil {
		return "", -1, "", -1, errco.NewErr(errco.ERROR_CONVERSION, errco.LVL_D, "getIpPorts", err.Error())
	}

	if TargetPort == c.Msh.ListenPort {
		return "", -1, "", -1, errco.NewErr(errco.ERROR_CONFIG_LOAD, errco.LVL_B, "getIpPorts", "TargetPort and ListenPort appear to be the same, please change one of them")
	}

	// return ListenHost, TargetHost, TargetPort, nil
	return ListenHost, c.Msh.ListenPort, TargetHost, TargetPort, nil
}

// getVersionInfo reads version.json from the server JAR file
// and returns minecraft server version and protocol.
// In case of error "", 0, *errco.Error are returned.
func (c *Configuration) getVersionInfo() (string, int, *errco.Error) {
	reader, err := zip.OpenReader(filepath.Join(c.Server.Folder, c.Server.FileName))
	if err != nil {
		return "", 0, errco.NewErr(errco.ERROR_VERSION_LOAD, errco.LVL_D, "getVersionInfo", err.Error())
	}
	defer reader.Close()

	for _, file := range reader.File {
		// search for version.json file
		if file.Name != "version.json" {
			continue
		}

		f, err := file.Open()
		if err != nil {
			return "", 0, errco.NewErr(errco.ERROR_VERSION_LOAD, errco.LVL_D, "getVersionInfo", err.Error())
		}
		defer f.Close()

		versionsBytes, err := ioutil.ReadAll(f)
		if err != nil {
			return "", 0, errco.NewErr(errco.ERROR_VERSION_LOAD, errco.LVL_D, "getVersionInfo", err.Error())
		}

		var info model.VersionInfo
		err = json.Unmarshal(versionsBytes, &info)
		if err != nil {
			return "", 0, errco.NewErr(errco.ERROR_VERSION_LOAD, errco.LVL_D, "getVersionInfo", err.Error())
		}

		return info.Version, info.Protocol, nil
	}

	return "", 0, errco.NewErr(errco.ERROR_VERSION_LOAD, errco.LVL_D, "getVersionInfo", "minecraft server version and protocol could not be extracted from version.json")
}

// assignMshID assigns a random mshid to config in case the present one is not correct
func (c *Configuration) assignMshID() {
	if len(c.Msh.ID) == 40 {
		errco.LogMshErr(errco.NewErr(errco.ERROR_CONFIG_CHECK, errco.LVL_D, "assignMshID", "mshid in config is valid, keeping it"))
	} else {
		// generate random mshid
		key := make([]byte, 64)
		_, _ = rand.Read(key)
		hasher := sha1.New()
		hasher.Write(key)
		c.Msh.ID = hex.EncodeToString(hasher.Sum(nil))
		ConfigDefaultSave = true
		errco.LogMshErr(errco.NewErr(errco.ERROR_CONFIG_CHECK, errco.LVL_D, "assignMshID", "mshid in config is not valid, new one is: "+c.Msh.ID))
	}
}
