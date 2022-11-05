package config

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"os"
	"reflect"
	"strconv"
	"time"

	"github.com/denisbrodbeck/machineid"

	"msh/lib/errco"
	"msh/lib/model"
	"msh/lib/opsys"
	"msh/lib/utility"
)

const instanceFile string = "msh.instance"
const CFLAG string = "/*\\"

type MshInstanceV model.MshInstanceV
type MshInstanceV0 model.MshInstanceV0

// MshID returns msh id. A new istance is created if not healthy/not existent.
func MshID() string {
	// if msh instance does not exist, generate a new one
	_, err := os.Stat(instanceFile)
	if errors.Is(err, os.ErrNotExist) {
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "MshID", "msh instance file does not exist"))
		return newMshInstance("")
	}

	errco.Logln(errco.LVL_3, "msh instance file exists")

	// read from file
	instanceData, err := os.ReadFile(instanceFile)
	if err != nil {
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "MshID", "msh instance file can't be read"))
		return newMshInstance("")
	}

	// replace NULL char with CFLAG to prevent JSON format error and wrong health check
	instanceData = bytes.ReplaceAll(instanceData, []byte{0}, []byte(CFLAG))

	// extract msh instance version
	var iv *MshInstanceV = &MshInstanceV{}
	err = json.Unmarshal(instanceData, iv)
	if err != nil {
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "MshID", "msh instance file does not contain version or not json formatted"))
		return newMshInstance("")
	}

	switch iv.V {
	case 0:
		var i *MshInstanceV0 = &MshInstanceV0{}

		// unmarshal msh.instance file data into instance struct
		err = json.Unmarshal(instanceData, i)
		if err != nil {
			errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "MshID", "msh instance file not json formatted"))
			return newMshInstance("")
		}

		// msh instance health check
		if !i.okV0() {
			errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "MshID", "msh instance loaded is corrupted"))
			return newMshInstance("")
		}

		errco.Logln(errco.LVL_3, "msh instance loaded is healthy")

		return i.MshId
		// when msh instance version is upgraded, above line will be replaced by:
		// return newMshInstance(i.MshId)

	default:
		// msh instance version is unsupported, generate a new instance
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "MshID", "msh instance loaded is unsupported"))

		return newMshInstance("")
	}
}

// newMshInstance generates a new instance file and returns a new mshid
func newMshInstance(mshIDrecord string) string {
	var i *MshInstanceV0 = &MshInstanceV0{}

	errco.Logln(errco.LVL_3, "generating new msh instance")

	// touch instance file (to know in advance file id)
	f, err := os.Create(instanceFile)
	if err != nil {
		log.Fatalln(err.Error())
	}
	_ = f.Close()

	// generate instance parameters
	i.V = 0                                   // set instance file version
	i.CFlag = CFLAG                           // set copy flag to CFLAG
	i.MId, err = machineid.ProtectedID("msh") // get machine id
	if err != nil {
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "newMshInstance", err.Error()))
	}
	i.HostName, err = os.Hostname() // get instance hostname
	if err != nil {
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "newMshInstance", err.Error()))
	}
	i.FId, err = opsys.FileId(instanceFile) // get instance file id
	if err != nil {
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "newMshInstance", err.Error()))
	}
	// try to use mshID old record
	i.MshId = mshIDrecord
	if utility.Entropy(i.MshId) < 150 {
		// old mshID entropy is too low: generate new mshid
		i.MshId = genMshId()
	}

	// generate msh instance checksum
	i.CheckSum = i.calcCheckSumV0()

	// marshal instance to bytes
	instanceData, err := json.Marshal(i)
	if err != nil {
		log.Fatalln(err.Error())
	}

	// replace CFLAG with NULL char to prevent accidental copy of msh.instance
	instanceData = bytes.ReplaceAll(instanceData, []byte(CFLAG), []byte{0})

	// write to instance file
	err = os.WriteFile(instanceFile, instanceData, 0644)
	if err != nil {
		log.Fatalln(err.Error())
	}

	// instance health check at birth
	if !i.okV0() {
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "newMshInstance", "generated msh instance is corrupted"))
		return newMshInstance(mshIDrecord)
	}

	errco.Logln(errco.LVL_3, "generated msh instance is healthy")

	return i.MshId
}

// ok verify that msh instance V0 is healthy
func (i *MshInstanceV0) okV0() bool {
	// check that instance exists
	if i == nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "ok", "msh instance struct not loaded"))
		return false
	}

	// check checksum
	Checksum := i.calcCheckSumV0()
	if i.CheckSum != Checksum {
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "ok",
			"msh instance verification: wrong checksum"+"\n"+
				"\tinst checksum "+i.CheckSum+"\n"+
				"\tfile checksum "+Checksum))
		return false
	}

	// check machine id
	MId, err := machineid.ProtectedID("msh") // get machine id
	if err != nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "ok", err.Error()))
	}
	if i.MId != MId {
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "ok",
			"msh instance verification: wrong machine id"+"\n"+
				"\tinst checksum "+i.MId+"\n"+
				"\tfile checksum "+MId))
		return false
	}

	// check hostname
	HostName, err := os.Hostname()
	if err != nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "ok", err.Error()))
	}
	if i.HostName != HostName {
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "ok",
			"msh instance verification: wrong hostname"+"\n"+
				"\tinst checksum "+i.HostName+"\n"+
				"\tfile checksum "+HostName))
		return false
	}

	// check file id
	FId, err := opsys.FileId(instanceFile)
	if err != nil {
		errco.LogMshErr(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "ok", err.Error()))
	}
	if i.FId != FId {
		errco.LogWarn(errco.NewErr(errco.ERROR_CONFIG_MSHID, errco.LVL_3, "ok",
			"msh instance verification: wrong file id"+"\n"+
				"\tinst checksum "+strconv.FormatUint(i.FId, 10)+"\n"+
				"\tfile checksum "+strconv.FormatUint(FId, 10)))
		return false
	}

	return true
}

// calcCheckSum calculates msh instance V0 checksum.
// CheckSum instance parameter is excluded from computation.
func (i *MshInstanceV0) calcCheckSumV0() string {
	hasher := sha1.New()

	v := reflect.ValueOf(*i)
	t := v.Type()
	o := ""
	for i := 0; i < v.NumField(); i++ {
		// skip CheckSum field as we are calculating it
		if t.Field(i).Name == "CheckSum" {
			continue
		}
		o += fmt.Sprintf("%v", v.Field(i))
	}

	hasher.Write([]byte(o))
	return hex.EncodeToString(hasher.Sum(nil))
}

// genMshId generates a new mshID with Shannon entropy above 150 bits
func genMshId() string {
	rand.Seed(time.Now().UnixNano())
	mshID := ""

	// mshID must have a Shannon entropy of more than 150 bits
	for utility.Entropy(mshID) <= 150 {
		key := make([]byte, 64)
		_, _ = rand.Read(key) // returned error is always nil
		hasher := sha1.New()
		hasher.Write(key)
		mshID = hex.EncodeToString(hasher.Sum(nil))
	}

	return mshID
}
