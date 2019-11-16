package bpa

import (
	"encoding/json"
	"fmt"
	"github.com/go-ini/ini"
	"io"
	"log"
	"math/rand"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"
)

func StrSliceIndex(slice []string, value string) (p int) {
	for p = range slice {
		if slice[p] == value {
			return
		}
	}

	p = -1
	return
}

func FileCopy(input, out string) (err error) {
	var fi, fo *os.File

	if fi, err = os.Open(input); err != nil {
		return
	}
	defer fi.Close()

	if err = os.MkdirAll(filepath.Dir(out), 0755); err != nil {
		return
	}

	if fo, err = os.Create(out); err != nil {
		return
	}
	defer fo.Close()

	if _, err = io.Copy(fo, fi); err != nil {
		os.Remove(out)
	}

	return
}

func HasElem(s interface{}, elem interface{}) bool {
	arrV := reflect.ValueOf(s)

	if arrV.Kind() == reflect.Slice {
		for i := 0; i < arrV.Len(); i++ {
			// XXX - panics if slice element points to an unexported struct field
			// see https://golang.org/pkg/reflect/#Value.Interface
			if arrV.Index(i).Interface() == elem {
				return true
			}
		}
	}

	return false
}

func SortStrSlice(s []string) {
	sort.Slice(s, func(i, j int) bool {
		return strings.ToLower(s[i]) < strings.ToLower(s[j])
	})
}

func CheckStr(s string) (v int) {
	var ok bool

	// identifier
	if ok, _ = regexp.MatchString("^[_a-zA-Z][_a-zA-Z0-9]*$", s); ok {
		v = 1
		return
	}

	// simple string
	if ok, _ = regexp.MatchString("^[_\\.\\-a-zA-Z0-9]*$", s); ok {
		v = 0
		return
	}

	v = -1
	return
}

func ErrExit(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func JsonTo(data interface{}, out io.Writer, readable bool) (err error) {
	var bts []byte

	if readable {
		bts, err = json.MarshalIndent(data, "", "  ")
	} else {
		bts, err = json.Marshal(data)
	}

	if err != nil {
		return
	}

	bts = append(bts, '\n')
	_, err = out.Write(bts)

	return
}

func JsonToFile(data interface{}, out string, readable bool) (err error) {

	if out == "-" {
		JsonTo(data, os.Stdout, true)
		return
	}

	var file *os.File
	if file, err = os.Create(out); err != nil {
		return
	}

	defer file.Close()

	if err = JsonTo(data, file, readable); err != nil {
		return
	}

	return
}

func ReadParam(input string, isPath bool) (kv map[string]string, err error) {
	var (
		options ini.LoadOptions
		k, v    string
		cfg     *ini.File
	)

	// options.AllowNestedValues = true
	options.AllowPythonMultilineValues = true

	kv = make(map[string]string)

	if isPath {
		cfg, err = ini.LoadSources(options, input)
	} else {
		cfg, err = ini.LoadSources(options, []byte(input))
	}

	if err != nil {
		err = fmt.Errorf("failed to read parameter config: " + err.Error())
		return
	}

	for _, key := range cfg.Section("").Keys() {
		k, v = key.Name(), key.Value()

		if strings.Contains(v, "'''") {
			err = fmt.Errorf("invalid value for \"%s\" in parameter config", k)
			return
		}

		if CheckStr(k) < 1 {
			err = fmt.Errorf("invalid key name \"%s\" in parameter config", k)
			return
		}

		kv[k] = v
	}

	return
}

func RandomStr(l int) string {
	str := "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ"

	bytes := []byte(str)
	result := make([]byte, l)

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < l; i++ {
		result[i] = bytes[r.Intn(len(bytes))]
	}

	return string(result)
}

func Min1(n int) int {
	if n < 1 {
		n = 1
	}

	return n
}

// 2019-06-15
func Map2Ini(mp map[string]string, sect string, hds, skip []string) (out string) {
	var k, v string
	var ks []string

	ks = make([]string, 0, len(mp))

	for k = range mp {
		ks = append(ks, k)
	}

	SortStrSlice(ks)

	if sect != "" {
		out = fmt.Sprintf("[%s]\n", sect)
	}

	for _, k = range hds {
		v = strings.Replace(mp[k], "\n", "\n\t", -1)
		out += fmt.Sprintf("%s = %s\n", k, v)
	}

	if len(hds) > 0 {
		out += "##\n"
	}

	for _, k = range ks {
		if StrSliceIndex(hds, k) > -1 || StrSliceIndex(skip, k) > -1 {
			continue
		}

		v = strings.Replace(mp[k], "\n", "\n\t", -1)
		out += fmt.Sprintf("%s = %s\n", k, v)
	}

	return
}

func ParseTarget(s string) (target []string, err error) {
	target = strings.SplitN(s, ":", 2)
	target[0] = strings.Trim(target[0], " ")

	if target[0] == "" {
		err = fmt.Errorf("invalid target \"%s\"", s)
		return
	}

	if len(target) == 1 {
		return
	}

	target[1] = strings.Replace(target[1], ",", " ", -1)

	if target[1] == "" {
		err = fmt.Errorf("invalid target \"%s\"", s)
		return
	}

	target = append(target[:1], strings.Fields(target[1])...)

	return
}

func Var2Slice(s string) (ss []string) {
	if !strings.Contains(s, ",") {
		ss = strings.Fields(s)
		return
	}

	ss = strings.Split(s, ",")
	for i := range ss {
		ss[i] = strings.TrimSpace(ss[i])
	}

	return
}
