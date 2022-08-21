package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/frontmatter"
	"gopkg.in/yaml.v2"
)

type Front struct {
	Title string `yaml:"title"`
	Url   string `yaml:"url"`
	Draft bool   `yaml:"draft"`
}

// recursively walk directory and return all files with given extension
func walk(root, ext string, index bool, ignorePaths map[string]struct{}) (res []Link, i ContentIndex, mapping map[string]string) {
	fmt.Printf("Scraping %s\n", root)
	i = make(ContentIndex)
	mapping = make(map[string]string)

	nPrivate := 0

	formats := []*frontmatter.Format{
		frontmatter.NewFormat("---", "---", yaml.Unmarshal),
	}

	start := time.Now()

	err := filepath.WalkDir(root, func(fp string, d fs.DirEntry, e error) error {
		if e != nil {
			return e
		}

		// path normalize fp
		s := filepath.ToSlash(fp)
		if _, ignored := ignorePaths[s]; ignored {
			fmt.Printf("[Ignored] %s\n", d.Name())
			nPrivate++
		} else if filepath.Ext(d.Name()) == ext {
			if index {
				text := getText(s)

				var matter Front
				raw_body, err := frontmatter.Parse(strings.NewReader(text), &matter, formats...)
				body := string(raw_body)
				if err != nil {
					matter = Front{
						Title: "Untitled Page",
						Draft: false,
					}
					body = text
				}
				// check if page is private
				if !matter.Draft {
					info, _ := os.Stat(s)
					source := processSource(trim(s, root, ".md"))

					// add to content and link index
					i[source] = Content{
						LastModified: info.ModTime(),
						Title:        matter.Title,
						Content:      body,
					}
					res = append(res, parse(s, root)...)

					mapping[source[strings.LastIndex(source, "/")+1:]] = "/" + matter.Url
				} else {
					fmt.Printf("[Ignored] %s\n", d.Name())
					nPrivate++
				}

			}
		}
		return nil
	})
	if err != nil {
		panic(err)
	}

	end := time.Now()

	fmt.Printf("[DONE] in %s\n", end.Sub(start).Round(time.Millisecond))
	fmt.Printf("Ignored %d private files \n", nPrivate)
	fmt.Printf("Parsed %d total links \n", len(res))
	return res, i, mapping
}

func getText(dir string) string {
	// read file
	fileBytes, err := ioutil.ReadFile(dir)
	if err != nil {
		panic(err)
	}

	return string(fileBytes)
}
