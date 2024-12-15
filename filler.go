package main

import (
	"archive/zip"
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"time"

	"mccoy.space/g/ogg"
)

type Filler struct {
	config        *Config
	entries       []Entry
	entries_chan  chan Entry
	loaded        int
	total         int
	index         int
	last_fill_at  int64
	next_fill_at  int64
	oggdata_index int
	oggdata       *[]byte
}

func NewFiller(config *Config) *Filler {
	f := &Filler{
		config,
		[]Entry{},
		make(chan Entry, 64),
		0, 0, 0, 0, 0, 0, nil,
	}
	f.entries = []Entry{}
	go func() {
		for {
			c := *f.config
			entry := <-f.entries_chan
			if entry.duration > 0 {
				f.entries = append(f.entries, entry)
			}
			f.loaded += 1
			if f.loaded%100 == 0 {
				fmt.Printf("loaded %3d / %3d\n", f.loaded, f.total)
			}
			if f.loaded == f.total {
				fmt.Printf("load complete, %d valid entries\n", len(f.entries))
				f.SortEntriesBy(c["sort_by"].raw)
				f.InitEntry(0)
			}
		}
	}()
	return f
}

func (f *Filler) StartLoad() {
	StartBuildVDirs()
	f.loaded = 0
	f.entries = []Entry{}
	f.total = 0
	c := *f.config
	target := c["target_in_pk3"].raw
	for _, dir := range c["oggdirs"].List() {
		files, _ := os.ReadDir(dir)
		f.total += len(files)
		for _, file := range files {
			name := file.Name()
			oggpath := path.Join(dir, name)
			go func() {
				entry := NewEntry(oggpath)
				entry.target = target
				f.entries_chan <- entry
			}()
		}
	}
	var count int64 = 0
	for _, list := range c["lists"].List() {
		file, err := os.Open(list)
		if err != nil {
			continue
		}
		defer file.Close()
		reader := bufio.NewReader(file)
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				break
			}
			id := count
			go func() {
				entry := ParseEntry(line[0 : len(line)-1])
				entry.modtime = id
				f.entries_chan <- entry
			}()
			count += 1
			f.total += 1
		}
	}
}

func (f *Filler) SortEntriesBy(by string) {
	SortEntries(&f.entries, by)
}

func (f *Filler) Fill(aggressive bool) Entry {
	now := time.Now().Unix()
	if (aggressive && now >= f.last_fill_at) ||
		now >= f.next_fill_at {
		f.InitEntry(f.index + 1)
	}
	return f.entries[f.index]
}

func (f *Filler) InitEntry(index int) {
	now := time.Now().Unix()
	if index >= len(f.entries) {
		index = 0
	}
	f.last_fill_at = now
	f.next_fill_at = f.last_fill_at + int64(f.entries[f.index].duration)/2
	f.index = index
}

type ReaderAtBytes struct {
	data *[]byte
}

func (r ReaderAtBytes) ReadAt(p []byte, off int64) (n int, err error) {
	offset := int(off)
	maxsize := len(*r.data)
	var i int = 0
	for ; i < len(p); i += 1 {
		if offset+i >= maxsize {
			return i, io.EOF
		}
		p[i] = (*r.data)[offset+i]
	}
	return i, nil
}

var encoder_serial uint32 = 0

func (f *Filler) GetOggBuffer(index int) (*bytes.Reader, error) {
	var err error
	var oggf fs.File
	if index == f.oggdata_index && f.oggdata != nil {
		return bytes.NewReader(*f.oggdata), nil
	}
	entry := f.entries[f.index]
	if entry.url != "" {
		resp, err := http.Get(entry.url)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		pk3, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		pk3r, err := zip.NewReader(ReaderAtBytes{&pk3}, int64(len(pk3)))
		if err != nil {
			return nil, err
		}
		oggf, err = pk3r.Open(entry.target)
		if err != nil {
			return nil, err
		}
	} else {
		oggf, err = os.Open(VPathToReal(entry.vpath))
		if err != nil {
			return nil, err
		}
	}
	oggdata, err := io.ReadAll(oggf)
	oggf.Close()
	if err != nil {
		return nil, err
	}
	f.oggdata = &oggdata
	f.oggdata_index = index
	return bytes.NewReader(oggdata), nil
}

func (f *Filler) StreamTo(w *io.PipeWriter) {
	buffer, err := f.GetOggBuffer(f.index)
	if err != nil {
		return
	}
	encoder := ogg.NewEncoder(encoder_serial, w)
	encoder_serial += 1
	decoder := ogg.NewDecoder(buffer)
	var sample_rate int32 = 44100
	var last_buff_at int64 = 0
	var buff_samples int64 = 5 * int64(sample_rate)
	current_index := f.index
start:
	for {
		page, err := decoder.Decode()
		switch err {
		case nil:
		default:
			if f.index == current_index {
				f.InitEntry(f.index + 1)
			}
			buffer, err = f.GetOggBuffer(f.index)
			last_buff_at = 0
			if err != nil {
				continue start
			}
			current_index = f.index
			decoder = ogg.NewDecoder(buffer)
			continue
		}
		// theora can't be properly encoded
		if bytes.Contains(page.Packets[0], []byte("theora")) {
			continue
		}
		if page.Type&ogg.BOS != 0 {
			err = encoder.EncodeBOS(page.Granule, page.Packets)
			if page.Type&ogg.COP == 0 {
				sample_rate = int32(page.Packets[0][13])*256 + int32(page.Packets[0][12])
			}
		} else {
			checkpoint := (time.Now().Unix() - f.last_fill_at) * int64(sample_rate)
			if page.Granule == 0 || page.Granule > checkpoint {
				if page.Type&ogg.EOS != 0 {
					err = encoder.EncodeEOS(page.Granule, page.Packets)
				} else {
					err = encoder.Encode(page.Granule, page.Packets)
				}
				if last_buff_at == 0 {
					buff_samples = (*f.config)["buffer"].Int() * int64(sample_rate) / 1000
					last_buff_at = page.Granule
				} else if page.Granule-last_buff_at >= buff_samples {
					buff_duration := (*f.config)["buffer"].Int()
					time.Sleep(time.Duration(buff_duration) * time.Millisecond)
					buff_samples = buff_duration * int64(sample_rate) / 1000
					last_buff_at = page.Granule
				}
			}
		}
		if err != nil {
			w.Close()
			return
		}
	}
}
