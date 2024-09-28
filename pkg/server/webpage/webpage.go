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

	"github.com/andybalholm/brotli"
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
const miniaoMsi = "miniao-installer.msi"
const iconIco = "icon.ico"

var (
	wasmFileBs      []byte
	wasmFileBrBs    []byte
	gameClientBs    []byte
	gameInstallerBs []byte
	iconImgBs       []byte
	wasmExecBs      []byte
	mainHtml        string
	goVersion       = "go1.23.1"
)

func init() {
	var err error
	wasmFile, err := os.Open("./bin/" + mainWasm)
	if err != nil {
		panic(err)
	}
	wasmFileBs, err = io.ReadAll(wasmFile)
	if err != nil {
		panic(err)
	}
	buf := bytes.NewBuffer(make([]byte, 0))
	compr := brotli.NewWriter(buf)
	_, err = compr.Write(wasmFileBs)
	if err != nil {
		panic(err)
	}
	compr.Close()

	wasmFileBrBs = buf.Bytes()

	gameClient, err := os.Open("./bin/" + miniaoExe)
	if err != nil {
		panic(err)
	}
	gameClientBs, err = io.ReadAll(gameClient)
	if err != nil {
		panic(err)
	}
	gameInstaller, err := os.Open("./bin/" + miniaoMsi)
	if err != nil {
		panic(err)
	}
	gameInstallerBs, err = io.ReadAll(gameInstaller)
	if err != nil {
		panic(err)
	}
	iconImg, err := os.Open("./pkg/server/webpage/" + iconIco)
	if err != nil {
		panic(err)
	}
	iconImgBs, err = io.ReadAll(iconImg)
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

	wasmExecBs, err = io.ReadAll(base64.NewDecoder(base64.StdEncoding, resp.Body))
	if err != nil {
		panic(err)
	}

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

	oenv := os.Environ()
	env := make([]string, 0, len(oenv))
	for _, e := range oenv {
		split := strings.SplitN(e, "=", 2)
		env = append(env, `"`+template.JSEscapeString(split[0])+`": "`+template.JSEscapeString(split[1])+`"`)
	}
	rpl := strings.NewReplacer(
		"{{.Argv}}", "["+strings.Join(argv, ", ")+"]",
		"{{.Env}}", "{"+strings.Join(env, ", ")+"}",
		"{{.MainWasm}}", `"`+template.JSEscapeString(mainWasm)+`"`,
	)
	mainHtml = rpl.Replace(indexHTML)
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
	"/miniao-installer.msi",
	"/favicon.ico",
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
		log.Printf("WEB:%v %v %v %v\n", r.RemoteAddr, r.Method, r.Host, r.URL.String())
	}()

	switch path {
	case "/":
		http.ServeContent(w, r, "index.html", time.Now(), bytes.NewReader([]byte(indexxHTML)))
	case "/main.html":
		http.ServeContent(w, r, "main.html", time.Now(), bytes.NewReader([]byte(mainHtml)))
	case "/wasm_exec.js":
		http.ServeContent(w, r, "wasm_exec.js", time.Time{}, bytes.NewReader(wasmExecBs))
	case "/main.wasm":
		w.Header().Add("Content-Encoding", "br")
		http.ServeContent(w, r, "main.wasm", time.Now(), bytes.NewReader(wasmFileBrBs))
	case "/miniao.exe":
		http.ServeContent(w, r, "miniao.exe", time.Now(), bytes.NewReader(gameClientBs))
	case "/miniao-installer.msi":
		http.ServeContent(w, r, "miniao-installer.msi", time.Now(), bytes.NewReader(gameInstallerBs))
	case "/favicon.ico":
		http.ServeContent(w, r, "favicon.ico", time.Now(), bytes.NewReader(iconImgBs))
	}
}
