// Copyright 2018 Hajime Hoshi
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

const mainWasm = "main.wasm"
const miniaoExe = "miniao.exe"

const indexxHTML = `<!DOCTYPE html>
<script>
addEventListener(
    'beforeunload',
    function(e){
        e.stopPropagation();e.preventDefault();return false;
    },
    true
);
</script>
<iframe src="main.html" allow="autoplay" style="position:fixed; top:0; left:0; bottom:0; right:0; width:100%; height:100%; border:none; margin:0; padding:0; overflow:hidden; z-index:999999;"></iframe>`

const indexHTML = `<!DOCTYPE html>
<script src="wasm_exec.js"></script>
<script>
(async () => {

  const resp = await fetch({{.MainWasm}});
  if (!resp.ok) {
    const pre = document.createElement('pre');
    pre.innerText = await resp.text();
    document.body.appendChild(pre);
  } else {
    const go = new Go();
    const result = await WebAssembly.instantiateStreaming(resp, go.importObject);
    go.argv = {{.Argv}};
    go.env = {{.Env}};
    go.run(result.instance);
  }
})();
</script>
`

var (
	flagHTTP        = flag.String("http", ":8080", "HTTP bind address to serve")
	flagAllowOrigin = flag.String("allow-origin", "", "Allow specified origin (or * for all origins) to make requests to this server")
)

var (
	waitChannel = make(chan struct{})

	wasmFile   *os.File
	gameClient *os.File
	wasmExec   []byte

	goVersion = "go1.23.1"
)

func init() {
	var err error
	wasmFile, err = os.Open("./cmd/web-server/" + mainWasm)

	if err != nil {
		panic(err)
	}

	gameClient, err = os.Open("./cmd/web-server/" + miniaoExe)

	if err != nil {
		panic(err)
	}
	var resp *http.Response
	url := fmt.Sprintf("https://go.googlesource.com/go/+/refs/tags/%s/misc/wasm/wasm_exec.js?format=TEXT", goVersion)
	resp, err = http.Get(url)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	wasmExec, err = io.ReadAll(base64.NewDecoder(base64.StdEncoding, resp.Body))
	if err != nil {
		panic(err)
	}
}

func handle(w http.ResponseWriter, r *http.Request) {
	if *flagAllowOrigin != "" {
		w.Header().Set("Access-Control-Allow-Origin", *flagAllowOrigin)
	}

	log.Printf("Request \n%v\n", *r)
	upath := r.URL.Path[1:]
	fpath := path.Base(upath)
	file := filepath.Base(fpath)
	if !strings.HasSuffix(r.URL.Path, "/") && file != "wasm_exec.js" && file != mainWasm && file != "miniao.exe" && file != "main.html" {
		http.Error(w, "", http.StatusNotFound)
		return
	}

	switch filepath.Base(fpath) {
	case ".":
		fpath = filepath.Join(fpath, "index.html")
		fallthrough
	case "index.html":
		if _, err := os.Stat(fpath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if errors.Is(err, fs.ErrNotExist) {
			http.ServeContent(w, r, "index.html", time.Now(), bytes.NewReader([]byte(indexxHTML)))
			return
		}
	case "main.html":
		if _, err := os.Stat(fpath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if errors.Is(err, fs.ErrNotExist) {
			firstArg := filepath.Join("/cmd/run-web-server", mainWasm)
			fargs := make([]string, flag.NArg())
			copy(fargs, flag.Args())
			if len(fargs) == 0 {
				fargs = append(fargs, firstArg)
			} else {
				fargs[0] = firstArg
			}
			argv := make([]string, 0, len(fargs))
			for _, a := range fargs {
				argv = append(argv, `"`+template.JSEscapeString(a)+`"`)
			}
			h := strings.ReplaceAll(indexHTML, "{{.Argv}}", "["+strings.Join(argv, ", ")+"]")

			oenv := os.Environ()
			env := make([]string, 0, len(oenv))
			for _, e := range oenv {
				split := strings.SplitN(e, "=", 2)
				env = append(env, `"`+template.JSEscapeString(split[0])+`": "`+template.JSEscapeString(split[1])+`"`)
			}
			h = strings.ReplaceAll(h, "{{.Env}}", "{"+strings.Join(env, ", ")+"}")

			h = strings.ReplaceAll(h, "{{.MainWasm}}", `"`+template.JSEscapeString(mainWasm)+`"`)

			http.ServeContent(w, r, "main.html", time.Now(), bytes.NewReader([]byte(h)))
			return
		}
	case "wasm_exec.js":
		if _, err := os.Stat(fpath); err != nil && !errors.Is(err, fs.ErrNotExist) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		} else if errors.Is(err, fs.ErrNotExist) {
			http.ServeContent(w, r, "wasm_exec.js", time.Time{}, bytes.NewReader(wasmExec))
			return
		}
	case mainWasm:
		http.ServeContent(w, r, mainWasm, time.Now(), wasmFile)
		return
	case "miniao.exe":
		http.ServeContent(w, r, "miniao.exe", time.Now(), gameClient)
	case "_wait":
		waitForUpdate(w, r)
		return
	case "_notify":
		notifyWaiters(w, r)
		return
	}
}

func target() string {
	if flag.NArg() > 0 {
		return flag.Args()[0]
	}
	return "."
}

func waitForUpdate(w http.ResponseWriter, r *http.Request) {
	waitChannel <- struct{}{}
	http.ServeContent(w, r, "", time.Now(), bytes.NewReader(nil))
}

func notifyWaiters(w http.ResponseWriter, r *http.Request) {
	for {
		select {
		case <-waitChannel:
		default:
			http.ServeContent(w, r, "", time.Now(), bytes.NewReader(nil))
			return
		}
	}
}

var (
	//go:embed pk_path.txt
	PKPath []byte
	//go:embed cert_path.txt
	CertPath []byte
)

func main() {
	flag.Parse()
	PKPath = []byte(strings.Trim(string(PKPath), "\n"))
	CertPath = []byte(strings.Trim(string(CertPath), "\n"))
	var server http.Server

	shutdown := make(chan struct{})
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt)
		<-sigint

		log.Printf("Shutting down server...")

		// Received an interrupt signal, shut down.
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()
		err := server.Shutdown(ctx)
		if err != nil && !errors.Is(err, context.DeadlineExceeded) {
			log.Printf("Error at server.Shutdown: %v", err)
		}
		close(shutdown)

		<-sigint
		// Hard exit on the second ctrl-c.
		os.Exit(0)
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/", handle)
	server.Handler = mux
	server.Addr = *flagHTTP
	log.Println("Paths configured")
	log.Println(string(CertPath))
	log.Println(string(PKPath))
	log.Printf("Listening on %v", *flagHTTP)
	err := server.ListenAndServeTLS(string(CertPath), string(PKPath))
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Printf("Error at server.ListenAndServe: %v", err)
	}

	<-shutdown

	log.Printf("Exiting")
}
