//+build ignore

package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"

	"github.com/pingcap/log"
	"github.com/pkg/fileutils"
)

var targetFolder = flag.String("target", "locales", "The folder to put locale JSON files")
var sourceFolder = flag.String("source", "", "The folder containing CLDR data")

func main() {
	flag.Parse()
	if len(*sourceFolder) == 0 {
		panic("source folder must provide")
	}

	filepath.Walk(*sourceFolder, func(path string, info os.FileInfo, err error) error {
		if info.Name() == "ca-gregorian.json" {
			parts := strings.Split(path, string(os.PathSeparator))
			targetFilePath := strings.Join([]string{*targetFolder, parts[len(parts)-2] + ".json"}, string(os.PathSeparator))
			err = fileutils.CopyFile(targetFilePath, path)
			if err != nil {
				log.Fatal(err.Error())
			}
		}

		return nil
	})
}
