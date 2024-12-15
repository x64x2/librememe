package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
)

const ROUTE_LIST = "/~list"
const ROUTE_PK3 = "/~pk3"
const ROUTE_FILL = "/~fill"

func get_urlbase(req *http.Request) string {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}
	host := req.Host
	if host == "" {
		host = "127.0.0.1"
	}
	return fmt.Sprintf("%s://%s%s/", scheme, host, ROUTE_PK3)
}

func write_oggpk3(oggpath string, target string, w http.ResponseWriter) {
	zipw := zip.NewWriter(w)
	defer zipw.Close()
	oggz, err := zipw.Create(target)
	if err != nil {
		return
	}
	oggf, err := os.Open(oggpath)
	if err != nil {
		return
	}
	defer oggf.Close()
	oggb := make([]byte, 256<<10)
	for {
		oggs, err := oggf.Read(oggb)
		if err != nil {
			break
		}
		oggz.Write(oggb[0:oggs])
	}
}

func main() {
	cfgpath := "config.txt"
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if arg == "--help" || arg == "-h" || arg == "/?" {
			fmt.Printf("usage: %s [path to config file, optional]\n", os.Args[0])
			os.Exit(0)
		}
		cfgpath = arg
	}
	config := NewConfig(cfgpath)
	filler := NewFiller(&config)
	fmt.Println("xonobo-go - type 'help' for a list of commands")

	log := Log{io.Discard}
	if config["log"].Bool() {
		log = Log{os.Stdout}
	}

	msgchan := make(chan Thing, 64)

	http.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		port_at := strings.LastIndex(req.Host, ":")
		host := req.Host
		if port_at != -1 {
			host = host[:port_at]
		}
		w.Write([]byte(fmt.Sprintf(
			"// Radio service for Xonotic SMB modded servers\n"+
				"set sv_radio 1\n"+
				"set sv_radio_queue_autofill 1\n"+
				"set sv_radio_queue_autofill_server http://%s/~fill\n\n"+
				"// or, listen to live stream at: http://%s:%d/stream.ogg\n\n"+
				"// source code: https://codeberg.org/NaitLee/xonobo-go\n",
			req.Host, host, config["stream"].Int(),
		)))
	})
	http.HandleFunc(ROUTE_LIST, func(w http.ResponseWriter, req *http.Request) {
		urlbase := get_urlbase(req)
		log.Println("list >>", req.RemoteAddr)
		for _, entry := range filler.entries {
			w.Write([]byte(entry.ToLine(urlbase)))
		}
	})
	http.HandleFunc(ROUTE_PK3+"/", func(w http.ResponseWriter, req *http.Request) {
		vpath := req.URL.Path[len(ROUTE_PK3)+1 : len(req.URL.Path)-len(".pk3")]
		log.Println("pk3  >>", req.RemoteAddr, "\t<", vpath)
		oggpath := VPathToReal(vpath)
		write_oggpk3(oggpath, config["target_in_pk3"].raw, w)
	})
	http.HandleFunc(ROUTE_FILL, func(w http.ResponseWriter, req *http.Request) {
		urlbase := get_urlbase(req)
		is_xonotic := strings.Contains(req.Header.Get("Host"), "Xonotic")
		entry := filler.Fill(is_xonotic)
		log.Println("fill >>", req.RemoteAddr, "\t<", entry.title)
		w.Write([]byte(entry.ToLine(urlbase)))
	})

	go config.FetchMsg(msgchan)
	go func() {
		for {
			msg := (<-msgchan).List()
			switch msg[0] {
			case "help":
				fmt.Print(
					"exit\tStop server\n" +
						"reload\tReload configuration and ogg files\n" +
						"sort\tSort entries by <name|modtime|shuffle>\n",
				)
			case "exit":
				os.Remove(config["pidpath"].raw)
				os.Remove(config["fdpath"].raw)
				os.Exit(0)
			case "reload":
				config = NewConfig(cfgpath)
				filler.config = &config
				filler.StartLoad()
			case "sort":
				filler.SortEntriesBy(msg[1])
			}
		}
	}()
	msgchan <- NewThing("reload")

	address := fmt.Sprintf("%s:%d", config["host"].raw, config["port"].Int())
	address_stream := fmt.Sprintf("%s:%d", config["host"].raw, config["stream"].Int())
	go func() {
		ln, err := net.Listen("tcp", address_stream)
		if err != nil {
			fmt.Printf("cannot listen on %s, so no stream\n", address_stream)
			return
		}
		fmt.Printf("stream at: http://%s/\n", address_stream)
		for {
			conn, err := ln.Accept()
			if err != nil {
				continue
			}
			log.Println("stream >>", conn.RemoteAddr().String())
			go func() {
				var err error
				r, w := io.Pipe()
				go filler.StreamTo(w)
				buffer := make([]byte, 256<<10)
				conn.Write([]byte(
					"HTTP/1.1 200 OK\r\n" +
						"Content-Type: application/ogg\r\n" +
						"Transfer-Encoding: chunked\r\n" +
						"\r\n",
				))
				// https://developer.mozilla.org/docs/Web/HTTP/Headers/Transfer-Encoding
				for {
					n, _err := r.Read(buffer)
					err = _err
					if err != nil {
						break
					}
					_, err = conn.Write([]byte(fmt.Sprintf("%x\r\n", n)))
					if err != nil {
						break
					}
					_, err = conn.Write(buffer[:n])
					if err != nil {
						break
					}
					_, err = conn.Write([]byte("\r\n"))
					if err != nil {
						break
					}
				}
				r.Close()
				w.Close()
				conn.Close()
			}()
		}
	}()

	fmt.Printf("xonobo at: http://%s/\n", address)
	http.ListenAndServe(address, nil)
}
