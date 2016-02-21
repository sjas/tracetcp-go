package main

import (
	"fmt"
	"io"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unicode"
)

type flushWriter struct {
	f http.Flusher
	w io.Writer
}

func (fw *flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	//log.Printf("%s", p)
	if fw.f != nil {
		fw.f.Flush()
	}
	return
}

func editCommandHandler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, "editcmd.html")
}

func validate(s string) bool {
	for _, r := range s {
		if !unicode.IsDigit(r) && !unicode.IsLetter(r) && r != '.' {
			return false
		}
	}
	return true
}

type traceConfig struct {
	host     string
	port     string
	starthop int
	endhop   int
	timeout  time.Duration
	queries  int
}

var defaultConfig = traceConfig{
	host:     "",
	port:     "http",
	starthop: 1,
	endhop:   30,
	timeout:  1 * time.Second,
	queries:  3,
}

func doTrace(w http.ResponseWriter, config traceConfig) {
	fw := flushWriter{w: w}
	if f, ok := w.(http.Flusher); ok {
		fw.f = f
	}

	if !validate(config.host) {
		fmt.Fprint(w, "Invalid Host Name")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	if !validate(config.port) {
		fmt.Fprint(w, "Invalid Port Number")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	cmd := exec.Command("tracetcp")
	cmd.Stdout = &fw
	cmd.Stderr = &fw

	cmd.Args = append(cmd.Args, config.host+":"+config.port)

	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(w, "%s\n", err)
		w.WriteHeader(http.StatusInternalServerError)
	}
}

func doTraceHandler(w http.ResponseWriter, r *http.Request) {
	config := defaultConfig

	if v, ok := r.URL.Query()["host"]; ok {
		config.host = v[0]
	}

	if v, ok := r.URL.Query()["port"]; ok {
		config.port = v[0]
	}

	var err error

	if v, ok := r.URL.Query()["starthop"]; ok {
		config.starthop, err = strconv.Atoi(v[0])
		if err != nil {
			fmt.Fprint(w, "Invalid Start Hop: ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	if v, ok := r.URL.Query()["endhop"]; ok {
		config.endhop, err = strconv.Atoi(v[0])
		if err != nil {
			fmt.Fprint(w, "Invalid End Hop: ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	if v, ok := r.URL.Query()["timeout"]; ok {
		config.timeout, err = time.ParseDuration(v[0])
		if err != nil {
			fmt.Fprint(w, "Invalid Timeout Duration: ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	if v, ok := r.URL.Query()["queries"]; ok {
		config.queries, err = strconv.Atoi(v[0])
		if err != nil {
			fmt.Fprint(w, "Invalid Query Count: ", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	doTrace(w, config)
}

func execHandler(w http.ResponseWriter, r *http.Request) {

	config := defaultConfig

	config.host = r.FormValue("host")
	config.port = r.FormValue("port")

	if r.FormValue("source") == "ok" {
		config.host = r.RemoteAddr[:strings.Index(r.RemoteAddr, ":")]
	}

	doTrace(w, config)
}

func main() {
	http.HandleFunc("/editcmd/", editCommandHandler)
	http.HandleFunc("/exec/", execHandler)
	http.HandleFunc("/dotrace", doTraceHandler)
	http.ListenAndServe(":8080", nil)
}