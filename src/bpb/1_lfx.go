package bpb

import (
	"fmt"
	"net/rpc"
	"path/filepath"
	"strings"
	. "x/src/bpa"
)

func (lfx *LFX) Accept(subm *Submit, msg *string) (err error) {
	var ok bool
	var tmp string

	lfs := lfx.LFS

	if subm.Mode != "run" && subm.Mode != "recover" {
		*msg = "failed"
		err = fmt.Errorf("invalid submit mode: " + subm.Mode)
		return
	}

	if len(subm.Addi) == 0 {
		*msg = "no task name to run or runner to recover set"
		err = fmt.Errorf("invalid submit Addi item")
		return
	}

	tmp = fmt.Sprintf("run_%x_%s", subm.At, subm.Name)

	defer func() {
		if err == nil {
			*msg = "Ok"
			lfs.Record("<--", subm.Mode, "node", tmp, "nil")
		} else {
			*msg = "failed"
			lfs.Record("<--", subm.Mode, "node", tmp, err.Error())
		}
	}()

	if _, ok = lfs.RBDMap[subm.Name]; ok {
		err = fmt.Errorf("runner \"%s\" name confict", subm.Addi[0])
		return
	}

	if subm.Mode == "recover" {
		var rbd *RBD
		if rbd, ok = lfs.RBDMap[subm.Addi[0]]; !ok {
			err = fmt.Errorf("runner \"%s\" not exists", subm.Addi[0])
			return
		}

		if rbd.R.Code < 1 || rbd.R.Status == "falling" {
			err = fmt.Errorf("runner \"\" status: %s", rbd.R.Name, rbd.R.Status)
			return
		}
	}

	if subm.Mode == "run" {
		err = lfs.NewRunner(subm)
	} else {
		err = lfs.NewRecover(subm)
	}

	return
}

func (lfx *LFX) Leave(by, reply *string) (err error) {
	var (
		tmp, address string
		client       *rpc.Client
		rec          *Record
	)

	lfs := lfx.LFS
	rec = NewRecord("leave", *by, lfs.PV("@"))

	defer func() {
		lfs.Ch <- [2]string{*by, "stop"}
	}()

	lfs.InterruptAll()
	address = strings.Join(strings.Fields(lfs.Node)[:2], ":")
	address = strings.Replace(address, "ip=", "", 1)
	address = strings.Replace(address, "rpc=", "", 1)

	if client, err = rpc.DialHTTP("tcp", address); err != nil {
		lfs.Record("!!!", rec.Operate, *by, "leaf", err.Error())
		return
	}

	if err = client.Call("NDX.Leave", rec, &tmp); err == nil {
		lfs.Record("-->", rec.Operate, *by, "leaf", "nil")
	} else {
		lfs.Record("!!!", rec.Operate, *by, "leaf", err.Error())
	}

	client.Close()

	return
}

func (lfx *LFX) Operate(rec *Record, reply *string) (err error) {
	var (
		ok      bool
		rbd     *RBD
		st, tmp string
		code    int
	)

	*reply = "OK"
	lfs := lfx.LFS

	rbd, ok = lfs.RBDMap[rec.On]

	if !ok {
		*reply = "failed"
		err = fmt.Errorf("runner not found")
		lfs.Record("<--", rec.Operate, "node", rec.On, err.Error())
		return
	}

	switch rec.Operate {
	case "progress":
		perc, records := rbd.D.Progress(0)
		*reply = perc
		if len(records) > 0 {
			*reply += "\n" + strings.Join(records, "\n")
		}

	case "interrupt":
		st, code = rbd.R.ReadStatus()
		if st == "falling" || code < 0 {
			rbd.R.WriteStatus("interrupted", 2)
			rbd.R.Cancel()
		} else {
			err = fmt.Errorf("conflict: %s, %d", st, code)
		}
	default:
		err = fmt.Errorf("undefined operate: " + rec.Operate)
	}

	if err != nil {
		*reply = "failed"
		tmp = err.Error()
	} else {
		tmp = "nil"
	}

	lfs.Record("<--", rec.Operate, rec.By, rec.On, tmp)
	return
}

func (lfx *LFX) Parameter(rec *Record, reply *string) (err error) {
	var df [][]string
	var list []string

	lfs := lfx.LFS
	df, err = ParamDF(filepath.Dir(lfs.Main), strings.Split(rec.On, ","))

	if err != nil {
		lfs.Record("!!!", "var", rec.By, rec.On, err.Error())
		return
	}

	list = make([]string, len(df))

	for i := range df {
		if df[i][2] != "" && df[i][3] == "" {
			df[i][3] = "???"
		}
		list[i] = strings.Join(df[i][:], "\t")
	}

	*reply = strings.Join(list, "\n")

	lfx.LFS.Record("<--", "var", rec.By, rec.On, "nil")

	return
}
