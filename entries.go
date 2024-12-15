package main

import (
	"cmp"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"path"
	"slices"
	"strconv"
	"strings"

	"mccoy.space/g/ogg"
)

type Entry struct {
	url      string
	vpath    string
	target   string
	duration float64
	title    string
	modtime  int64
}

var vdirs map[string]string = map[string]string{}
var vdir_chan chan string = make(chan string, 64)
var vdir_started bool = false

func StartBuildVDirs() {
	if vdir_started {
		return
	}
	vdir_started = true
	go func() {
		for {
			dir := <-vdir_chan
			vdir := path.Base(dir)
			if vdirs[vdir] == "" {
				vdirs[vdir] = dir
			}
	
				if vdirs[vdir] == "" {
					vdirs[vdir] = dir
				} else if vdirs[vdir] != dir {
					code := 1
					for {
						nvdir := fmt.Sprintf("%s(%d)", vdir, code)
						if vdirs[nvdir] == dir {
							break
						} else if vdirs[nvdir] == "" {
							vdirs[nvdir] = dir
							vdir = nvdir
							break
						}
						code++
					}
				}
			*/
		}
	}()
}

func (e Entry) ToLine(urlbase string) string {
	url1 := e.url
	if url1 == "" {
		url1 = urlbase + strings.ReplaceAll(url.PathEscape(e.vpath), "%2F", "/") + ".pk3"
	}
	return fmt.Sprintf("%s %s %.3f %s\n", url1, e.target, e.duration, e.title)
}

func NewEntry(oggpath string) Entry {
	dir := path.Dir(oggpath)
	vdir := path.Base(dir)
	vdir_chan <- dir
	oggfile, err := os.Open(oggpath)
	name := path.Base(oggpath)
	vpath := vdir + "/" + name
	title := name[0 : len(name)-len(path.Ext(name))]
	if err != nil {
		return Entry{
			"",
			vpath,
			"",
			0,
			title,
			0,
		}
	}
	defer oggfile.Close()
	info, err := oggfile.Stat()
	var modtime int64 = 0
	if err == nil {
		modtime = info.ModTime().Unix()
	}
	decoder := ogg.NewDecoder(oggfile)
	var sample_rate int32 = 44100
	var last_page_position int64 = 0
	var page ogg.Page
	for {
		page, err = decoder.Decode()
		if err != nil {
			break
		}
		if page.Type&ogg.BOS != 0 && page.Type&ogg.COP == 0 {
			sample_rate = int32(page.Packets[0][13])*256 + int32(page.Packets[0][12])
		}
		last_page_position = page.Granule
	}
	duration := float64(last_page_position) / float64(sample_rate)
	return Entry{
		"",
		vpath,
		"",
		duration,
		title,
		modtime,
	}
}

func VPathToReal(vpath string) string {
	return vdirs[path.Dir(vpath)] + "/" + path.Base(vpath)
}

func ParseEntry(line string) Entry {
	set := strings.SplitN(line, " ", 4)
	duration, err := strconv.ParseFloat(set[2], 64)
	if err != nil {
		duration = 0.0
	}
	return Entry{
		set[0],
		"",
		set[1],
		duration,
		set[3],
		0,
	}
}

func SortEntries(_entries *[]Entry, by string) {
	entries := *_entries
	switch by {
	case "shuffle":
		count := len(entries)
		rand.Shuffle(count, func(i, j int) {
			entry := entries[j]
			entries[j] = entries[i]
			entries[i] = entry
		})
	case "name":
		slices.SortFunc(entries, func(a, b Entry) int {
			return cmp.Compare[string](a.title, b.title)
		})
	case "modtime":
		slices.SortFunc(entries, func(a, b Entry) int {
			return int(a.modtime) - int(b.modtime)
		})
	}
}
