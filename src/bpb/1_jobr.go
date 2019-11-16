package bpb

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strings"
	"time"
)

func (job *Job) ReqQuery(item string) (code int, data interface{}) {
	var (
		msg  *MSG
		subm *Submit
		ok   bool
	)

	msg = new(MSG)
	code, data = http.StatusOK, msg

	switch item {
	case "":
		msg.Title = "query var required"
		msg.List = []string{
			"?item=leaf",
			"?item=record",
			"?item=submit",
			"?item=[submit_name]",
		}

	case "leaf":
		data = struct {
			UUID, Link string
			Created    time.Time
			*Project
		}{job.UUID, job.Link, job.Created, job.Project}

	case "record":
		data = job.Records

	case "submit":
		var k, at, rc string
		var ct time.Time
		msg.Title = "submitted runners"
		msg.List = make([]string, 0, len(job.SubmitMap))

		for k, subm = range job.SubmitMap {
			ct = time.Unix(subm.At, 0)
			at = ct.Format(time.RFC3339)
			rc = fmt.Sprintf("%s, %s, %s", at, subm.Status, k)
			if subm.Code >= 0 {
				rc += ", endat=" + subm.EndAt.Format(time.RFC3339)
			} else {
				rc += ", actived=" + time.Now().Sub(ct).String()
			}

			msg.List = append(msg.List, rc)
		}

	default:
		if subm, ok = job.SubmitMap[item]; !ok {
			msg.Title = "submit not found: " + item
			code, data = http.StatusNotFound, msg
		} else {
			data = subm
		}
	}

	return
}

func (job *Job) ReqInterrupt(item string) (code int, data interface{}) {
	var msg *MSG
	var rec *Record

	msg, rec = new(MSG), NewRecord("interrupt", "node", item)
	code, data = http.StatusOK, msg

	msg.Title = fmt.Sprintf("%s %s: ", rec.On, rec.Operate)
	_, err := job.Operate(rec)

	if err == nil {
		msg.Title += "OK"
	} else {
		msg.Title += err.Error()
		code = http.StatusConflict
	}

	return
}

func (job *Job) ReqProgress(item string) (code int, data interface{}) {
	var (
		result, d string
		err       error
		tmp       []string
		msg       *MSG
		rec       *Record
		subm      *Submit
	)

	msg, rec = new(MSG), NewRecord("progress", "node", item)
	code, data = http.StatusOK, msg

	msg.Title = fmt.Sprintf("%s %s: ", rec.On, rec.Operate)
	result, err = job.Operate(rec)
	tmp = strings.Split(result, "\n")

	if err != nil {
		msg.Title += err.Error()
		code = http.StatusConflict
		return
	}

	subm, msg.List = job.SubmitMap[rec.On], tmp[1:]

	if subm.Code < 0 {
		d = time.Now().Sub(time.Unix(subm.At, 0)).String()
		msg.Title += tmp[0] + fmt.Sprintf(", %s=%s", "actived", d)
	} else {
		msg.Title += tmp[0] + fmt.Sprintf(", %s=%s", "elapsed", subm.Elapsed)
	}

	return
}

func (job *Job) ReqSubmit(c *gin.Context, timeout string) (
	code int, data interface{}) {

	var (
		msg    *MSG
		subm   *Submit
		bts    []byte
		cfgstr string
		err    error
		c1     int
	)

	msg, subm = new(MSG), new(Submit)
	code, data = http.StatusOK, msg

	if bts, err = ReadUpload(c, "json"); err != nil {
		msg.Title = "failed parse json: " + err.Error()
		code = http.StatusNotAcceptable
		return
	}

	if err = json.Unmarshal(bts, &subm); err != nil {
		msg.Title = "failed parse json: " + err.Error()
		code = http.StatusNotAcceptable
		return
	}

	// discard timeout value from input json
	if subm.Timeout, err = time.ParseDuration(timeout); err != nil {
		msg.Title = "failed parse timeout: " + err.Error()
		code = http.StatusNotAcceptable
		return
	}

	if c.PostForm("cfgfile") != "" {
		if bts, err = ReadUpload(c, "cfgfile"); err != nil {
			msg.Title = "failed parse cfg: " + err.Error()
			code = http.StatusNotAcceptable
			return
		}

		cfgstr = string(bts)
	}

	subm.At = time.Now().Unix()

	switch subm.Mode {
	case "recover":
		subm.Cfg = cfgstr
		c1, err = job.SubmitRecover(subm)
	case "run":
		c1, err = job.SubmitNew(subm)
	default:
		c1 = 1
		err = fmt.Errorf("invalid mode: %s", subm.Mode)
	}

	switch c1 {
	case 0:
		msg.Title = subm.Name
	case 1:
		msg.Title = "failed to parse submit in node: " + err.Error()
		code = http.StatusBadRequest
	case 2:
		msg.Title = "failed to excute runner in leaf: " + err.Error()
		code = http.StatusUnprocessableEntity
	}

	return
}

func ReadUpload(c *gin.Context, name string) (bts []byte, err error) {
	var fh *multipart.FileHeader
	var file multipart.File

	if fh, err = c.FormFile(name); err != nil {
		err = fmt.Errorf("failed to get " + name)
		return
	}

	if file, err = fh.Open(); err != nil {
		err = fmt.Errorf("failed to open " + name)
		return
	}

	defer file.Close()

	if bts, err = ioutil.ReadAll(file); err != nil {
		err = fmt.Errorf("failed to read " + name)
		return
	}

	return
}

func (job *Job) ReqParameter(tk string) (code int, data interface{}) {
	var (
		msg   *MSG
		rec   *Record
		reply string
		err   error
	)

	msg, rec = new(MSG), NewRecord("var", "node", tk)
	code, data = http.StatusOK, msg
	msg.Title = fmt.Sprintf("%s: %s", rec.Operate, rec.On)

	if err = job.Client.Call("LFX.Parameter", rec, &reply); err != nil {
		msg.List = []string{err.Error()}
		code = http.StatusBadRequest
		return
	}

	msg.List = strings.Split(reply, "\n")

	return
}
