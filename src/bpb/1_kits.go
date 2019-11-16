package bpb

import (
	"net"
	"net/http"
	"time"
)

func ReqAPI() (code int, data interface{}) {
	msg := new(MSG)
	code, data = http.StatusOK, msg

	msg.Title = "API entries"
	msg.List = append(API_A_Get, API_B_Get...)
	msg.List = append(msg.List, API_B_POST...)

	return
}

func (pjt *Project) PV(sep string) string {
	return pjt.Project + sep + pjt.Version
}

func NewPort() (port int, err error) {
	var listener net.Listener

	if listener, err = net.Listen("tcp", ":0"); err != nil {
		return
	}

	port = listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	return
}

func NewRecord(operate, by, on string) (rec *Record) {
	rec = new(Record)
	rec.At = time.Now().Unix()
	rec.Operate, rec.By, rec.On = operate, by, on
	return
}
