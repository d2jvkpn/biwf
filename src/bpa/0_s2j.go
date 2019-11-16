package bpa

import (
	"fmt"
	"github.com/go-ini/ini"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func S2J(input *Input) {
	if len(input.Addi) == 0 {
		log.Fatal("expand input format:  <f1.ini,f2.ini...>  <sect1 sect2 sect3>")
	}

	var (
		err     error
		k       string
		ss      []string
		bts     []byte
		tmp     []byte
		data    map[string]map[string][]string
		cfg     *ini.File
		sect    *ini.Section
		key     *ini.Key
		options ini.LoadOptions
	)

	bts = make([]byte, 0)

	// stderr output 1 means section or key not found
	ss = strings.Fields(strings.Replace(input.Addi[0], ",", " ", -1))

	for _, k = range ss {
		if tmp, err = ioutil.ReadFile(k); err != nil {
			log.Fatalln(err)
		}
		bts = append(bts, tmp...)
	}

	// options.AllowNestedValues = true
	options.AllowPythonMultilineValues = true

	if cfg, err = ini.LoadSources(options, bts); err != nil {
		log.Fatalln(err)
	}

	if len(input.Addi) == 1 {
		for _, sect = range cfg.Sections() {
			fmt.Println(sect.Name())
		}
		return
	}

	ss = input.Addi[1:]
	data = make(map[string]map[string][]string)

	for _, k = range ss {
		ts := make(map[string][]string)

		if sect, err = cfg.GetSection(k); err != nil {
			log.Printf("section \"%s\" not found\n", k)
			data[k] = ts
			continue
		}

		for _, key = range sect.Keys() {
			ts[key.Name()] = Var2Slice(key.Value())
		}

		data[sect.Name()] = ts
	}

	JsonTo(data, os.Stdout, true)
}
