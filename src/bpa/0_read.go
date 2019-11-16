package bpa

import (
	"fmt"
	"github.com/go-ini/ini"
	"io/ioutil"
	"log"
	"os"
	"strings"
)

func ReadCfg(input *Input) {
	if len(input.Addi) == 0 {
		log.Fatal("read mode input format:  <f1.ini,f2.ini...>  [section]  [key]")
	}

	var (
		options ini.LoadOptions
		cfg     *ini.File
		err     error
		bts     []byte = make([]byte, 0)
		tmp     []byte
		section *ini.Section
		key     *ini.Key
		out     []string
	)

	fs := strings.Fields(strings.Replace(input.Addi[0], ",", " ", -1))

	// stderr output 1 means section or key not found
	for _, fn := range fs {
		if tmp, err = ioutil.ReadFile(fn); err != nil {
			log.Fatalln(err)
		}
		bts = append(bts, tmp...)
	}

	// options.AllowNestedValues = true
	options.AllowPythonMultilineValues = true

	if cfg, err = ini.LoadSources(options, bts); err != nil {
		log.Fatalln(err)
	}

	out = make([]string, 0, 10)

	switch len(input.Addi) {
	case 3:
		if section, err = cfg.GetSection(input.Addi[1]); err != nil {
			fmt.Fprintf(os.Stderr, "section \"%s\" not found\n", input.Addi[1])
			return
		}

		if key, err = section.GetKey(input.Addi[2]); err != nil {
			fmt.Fprintf(os.Stderr, "key \"%s\" not found\n", input.Addi[2])
			return
		}

		v := key.Value()
		vs := key.NestedValues()

		if v == "" && len(vs) > 0 {
			for i := range vs {
				out = append(out, vs[i])
			}
		} else {
			out = append(out, v)
		}

	case 2:
		if section, err = cfg.GetSection(input.Addi[1]); err != nil {
			fmt.Fprintf(os.Stderr, "section \"%s\" not found\n", input.Addi[1])
			return
		}

		for _, v := range section.KeyStrings() {
			out = append(out, v)
		}

	case 1:
		for _, v := range cfg.SectionStrings() {
			out = append(out, v)
		}

	default:
		log.Fatal("input format:  <f1.ini,f2.ini...>  [section]  [key]")
	}

	fmt.Printf(strings.Join(out, "\n") + "\n")
}
