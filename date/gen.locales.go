//+build ignore

package main

//This program generates locales.go. It can be invoked by running
//go:generate

import (
	"fmt"
	"io/ioutil"
	"log"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/datasweet/jsonmap"
	"github.com/pkg/errors"

	"github.com/xiaochao8/format/date/locales"
	"github.com/xiaochao8/format/third_party/gen"

	"golang.org/x/text/language"
)

type field struct {
	name   string
	widths map[locales.Width]string
	keys   []string
}

type keyIndex struct {
	pos   int
	key   string
	xpath string
}

type translation struct {
	Lang      string
	ULang     string
	langData  []string
	langIndex []uint32
}

func main() {
	files, err := filepath.Glob("locales/*.json")
	die(errors.Wrap(err, "can't list locales"))

	var indexes []*keyIndex
	var translations []*translation

	// analyze fields
	for _, field := range locales.Fields {
		for w, p := range field.Widths {
			for i, k := range field.Keys {
				if key, ok := field.Key(w, i); ok {
					indexes = append(indexes, &keyIndex{
						pos:   len(indexes),
						key:   key,
						xpath: strings.Join([]string{field.CldrNode, p, k}, "."),
					})
				}
			}
		}
	}

	for _, filename := range files {
		locale := filename[8 : len(filename)-5]
		_, err := language.Parse(locale)
		if err := nil {
			log.Printf("invalid locale '%s'\n", locale)
			continue
		}

		data, err := ioutil.ReadFile(filename)
		die(errors.Wrapf(err, "read file %s", filename))

		j := jsonmap.FromBytes(data)
		lang := j.Get("main").Get(locale)
		if jsonmap.IsNil(lang) {
			log.Printf("wrong lang : '%s'\n", locale)
			continue
		}

		dates := lang.Get("dates.calendars.gregorian")
		if jsonmap.IsNil(dates) {
			log.Fatal("not a gregorian calendar")
		}

		translation := new(translation)
		translation.Lang = locale
		translation.ULang = strings.ReplaceAll(translation.Lang, "-", "_")
		translation.langIndex = []uint32{0}
		pos := 0

		for _, keyIndex := range indexes {
			s := dates.Get(keyIndex.xpath).AsString()
			// fmt.Println(translation.Lang, keyIndex.key, keyIndex.pos, keyIndex.xpath, s)
			pos += len(s)
			translation.langData = append(translation.langData, s)
			translation.langIndex = append(translation.langIndex, uint32(pos))
		}

		translations = append(translations, translation)
	}

	cw := gen.NewCodeWriter()

	// Generate code file
	err = lookup.Execute(cw, translations)
	die(errors.Wrap(err, "tmpl"))

	fmt.Fprint(cw, "var messageKeyToIndex = map[string]int{\n")
	for _, keyIndex := range indexes {
		fmt.Fprintf(cw, "%q: %d,\n", keyIndex.key, keyIndex.pos)
	}
	fmt.Fprint(cw, "}\n\n")

	for _, tr := range translations {
		cw.WriteVar(fmt.Sprintf("%sIndex", tr.ULang), tr.langIndex)
		cw.WriteConst(fmt.Sprintf("%sData", tr.ULang), strings.Join(tr.langData, ""))
	}
	cw.WriteGoFile("locales/locales_gen.go", "locales")

	// Generate test file
	cw = gen.NewCodeWriter()
	err = test.Execute(cw, translations)
	die(errors.Wrap(err, "tmpl"))
	cw.WriteGoFile("locales/locales_gen_test.go", "locales_test")

	fmt.Println("Done, check file locales/locales_gen.go")
}

func die(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

var lookup = template.Must(template.New("gen").Parse(`

import "golang.org/x/text/message/catalog"

type dictionary struct {
	index []uint32
	data  string
}
func (d *dictionary) Lookup(key string) (data string, ok bool) {
	p, ok := messageKeyToIndex[key]
	if !ok {
		return "", false
	}
	start, end := d.index[p], d.index[p+1]
	if start == end {
		return "", false
	}
	return d.data[start:end], true
}
func init() {
	locales = map[string]catalog.Dictionary{
		{{range .}}
		"{{.Lang}}": &dictionary{index: {{.ULang}}Index, data: {{.ULang}}Data },
		{{end}}
	}
}
`))

var test = template.Must(template.New("test").Parse(`
import (
	"testing"

	"github.com/xiaochao8/format/date/locales"
	"github.com/stretchr/testify/assert"
)
type dictionary struct {
	index []uint32
	data  string
}
{{range .}}
func TestLocalize{{.ULang}}(t *testing.T) {
	for ft, fi := range locales.Fields {
		for w := range fi.Widths {
			for i := range fi.Keys {
				key, ok := fi.Key(w, i)
				assert.True(t, ok, key)
				assert.NotEmpty(t, key, key)
				s := locales.Localize("{{.Lang}}", ft, w, i)
				assert.NotEmpty(t, s, key)
			}
		}
	}
}
{{end}}
`))
