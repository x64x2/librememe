package server

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"io"
	stdfs "io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"syscall"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/playground"
	"github.com/gorilla/mux"
	"github.com/quantumsheep/range-parser"
	"github.com/rs/cors"
	"github.com/sirupsen/logrus"

)

func StartServer() {
	bind := config.Global.GetString(config.KeyServerBind)
	port := config.Global.GetString(config.KeyServerPort)

	resolver, err := graph.NewResolver()
	if err != nil {
		log.Fatalf("Failed to create graphQL resolver: %v", err)
	}
	srv := handler.NewDefaultServer(generated.NewExecutableSchema(generated.Config{
		Resolvers: resolver,
	}))

	r := mux.NewRouter()

	r.Handle("/graphql", playground.Handler("GraphQL playground", "/query"))
	r.Handle("/query", srv).Methods(http.MethodPost)
	r.HandleFunc("/file/{name:.*}", serveFile(false)).Methods(http.MethodGet)
	r.HandleFunc("/file/{name:.*}", serveFile(true)).Methods(http.MethodHead)
	r.PathPrefix("/").Handler(spaHandler{
		staticFS:   client.Build,
		staticPath: "build",
		indexPath:  "index.html",
	}).Methods(http.MethodGet)

	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:8090"},
		AllowedMethods: []string{http.MethodHead, http.MethodGet, http.MethodPost},
	})
	handle := c.Handler(r)

	log.Printf("App is running at http://%s:%s/", bind, port)
	log.Fatal(http.ListenAndServe(bind+":"+port, handle))
}

func fileServe(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	if name == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(http.StatusText(http.StatusNotFound)))
		return
	}

	exists, err := fs.SharedFS.Get(r.Context(), name, 0, 0, w)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		log.Errorf("Error from fs while requesting '%s': %v", name, err)
	} else if !exists {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(http.StatusText(http.StatusNotFound)))
	}
}

func fileHead(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	name := vars["name"]
	if name == "" {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(http.StatusText(http.StatusNotFound)))
		return
	}

	stat, err := fs.SharedFS.Stat(r.Context(), name)
	if err != nil {
		if r.Context().Err() != nil {
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
		log.Errorf("Error from fs while getting stat '%s': %v", name, err)
		return
	}

	if stat == -1 {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(http.StatusText(http.StatusNotFound)))
		return
	}

	ext := filepath.Ext(name)
	if ext != "" {
		w.Header().Add("Content-Type", mime.TypeByExtension(ext))
	}

	w.Header().Add("Content-Length", strconv.FormatInt(stat, 10))
	w.Header().Add("Accept-Ranges", "bytes")
}

func serveFile(isHead bool) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		name := vars["name"]
		if name == "" {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(http.StatusText(http.StatusNotFound)))
			return
		}

		stat, err := fs.SharedFS.Stat(r.Context(), name)
		if err != nil {
			if r.Context().Err() != nil {
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			log.Errorf("Error from fs while getting stat '%s': %v", name, err)
			return
		}

		if stat == -1 {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(http.StatusText(http.StatusNotFound)))
			return
		}

		ext := filepath.Ext(name)
		if ext != "" {
			w.Header().Add("Content-Type", mime.TypeByExtension(ext))
		}

		if isHead {
			w.Header().Add("Content-Length", strconv.FormatInt(stat, 10))
			w.Header().Add("Accept-Ranges", "bytes")
		} else {
			rngHeader := r.Header.Get("Range")
			var (
				rng              *range_parser.Range
				getStart, getLen int64
			)
			if rngHeader != "" {
				ranges, err := range_parser.Parse(int64(stat), rngHeader)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, "Invalid range '%s': %v", rngHeader, err)
					log.Errorf("Invalid range '%s': %v", rngHeader, err)
					return
				}
				if len(ranges) == 1 {
					rng = ranges[0]
					if rng.Start > stat {
						w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
						fmt.Fprintf(w, "Range not satisfiable '%s'", rngHeader)
						log.Errorf("Range not satisfiable '%s'", rngHeader)
						return
					}
					if rng.Start > rng.End {
						w.WriteHeader(http.StatusBadRequest)
						fmt.Fprintf(w, "Invalid range '%s': %v", rngHeader, err)
						log.Errorf("Invalid range '%s': %v", rngHeader, err)
						return
					}
					if rng.End > stat {
						rng.End = stat
					}
					getStart = rng.Start
					getLen = rng.End - rng.Start + 1

					w.Header().Add("Content-Length", strconv.FormatInt(getLen, 10))
					w.Header().Add("Content-Range", fmt.Sprintf("bytes %d-%d/%d", rng.Start, rng.End, stat))
					w.Header().Add("Accept-Ranges", "bytes")
					w.WriteHeader(http.StatusPartialContent)
				} else if len(ranges) > 1 {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintf(w, "Multiple ranges unsupported: '%s'", rngHeader)
					log.Errorf("Multiple ranges unsupported: '%s'", rngHeader)
					return
				}
			}

			if rng == nil {
				w.Header().Add("Content-Length", strconv.FormatInt(stat, 10))
				w.Header().Add("Accept-Ranges", "bytes")
			}

			exists, err := fs.SharedFS.Get(r.Context(), name, getStart, getLen, w)
			if err != nil {
				if r.Context().Err() != nil || errors.Is(err, context.Canceled) || errors.Is(err, io.ErrClosedPipe) || errors.Is(err, syscall.EPIPE) {
					return
				}
				w.WriteHeader(http.StatusInternalServerError)
				log.Errorf("Error from fs while requesting '%s': %v", name, err)
				return
			}
			if !exists {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(http.StatusText(http.StatusNotFound)))
				return
			}
		}
	}
}

type spaHandler struct {
	staticFS   embed.FS
	staticPath string
	indexPath  string
}

func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	reqpath, err := filepath.Abs(r.URL.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	reqpath = filepath.Join(h.staticPath, reqpath)

	_, err = h.staticFS.Open(reqpath)
	if os.IsNotExist(err) {
		index, err := h.staticFS.ReadFile(filepath.Join(h.staticPath, h.indexPath))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusAccepted)
		w.Write(index)
		return
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	spa, err := stdfs.Sub(h.staticFS, h.staticPath)
	http.FileServer(http.FS(spa)).ServeHTTP(w, r)
}
