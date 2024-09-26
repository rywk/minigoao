package webpage

import (
	"bytes"
	_ "embed"
	"encoding/base64"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"
)

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
const mainWasm = "main.wasm"
const miniaoExe = "miniao.exe"

var (
	wasmFile   *os.File
	gameClient *os.File
	wasmExec   []byte

	goVersion = "go1.23.1"
)

func init() {
	var err error
	wasmFile, err = os.Open("./bin/" + mainWasm)

	if err != nil {
		panic(err)
	}

	gameClient, err = os.Open("./bin/" + miniaoExe)

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

func Handle(upgrader func(w http.ResponseWriter, r *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/upgrader" {
			upgrader(w, r)
			return
		}

		if r.Method != "GET" {
			log.Printf("NON GET:%v %v %v\n", r.Method, r.Host, r.URL.String())
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		if !ValidPath(r.URL.Path) {
			log.Printf("INVALID GET:%v %v %v\n", r.Method, r.Host, r.URL.String())
			http.NotFound(w, r)
			return
		}

		HandleWeb(w, r, r.URL.Path)
	}
}

var paths = []string{
	"/",
	"/main.html",
	"/wasm_exec.js",
	"/main.wasm",
	"/miniao.exe",
}

func ValidPath(path string) bool {
	for _, p := range paths {
		if path == p {
			return true
		}
	}
	return false
}

func HandleWeb(w http.ResponseWriter, r *http.Request, path string) {
	defer func() {
		log.Printf("WEB:%v %v %v\n", r.Method, r.Host, r.URL.String())
	}()

	switch path {
	case "/":
		http.ServeContent(w, r, "index.html", time.Now(), bytes.NewReader([]byte(indexxHTML)))
		return
	case "/main.html":
		firstArg := filepath.Join("/bin/", mainWasm)
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
	case "/wasm_exec.js":
		http.ServeContent(w, r, "wasm_exec.js", time.Time{}, bytes.NewReader(wasmExec))
		return
	case "/" + mainWasm:
		//w.Header().Add("Content-Type", "application/wasm")
		http.ServeContent(w, r, mainWasm, time.Now(), wasmFile)
		return
	case "/miniao.exe":
		http.ServeContent(w, r, "miniao.exe", time.Now(), gameClient)
	}
}
