package main

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"regexp"

	vueglue "github.com/torenware/vite-go"
)

var vueGlue *vueglue.VueGlue

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s - %s %s %s", r.RemoteAddr, r.Proto, r.Method, r.URL.RequestURI())
		next.ServeHTTP(w, r)
	})
}

func serveOneFile(w http.ResponseWriter, r *http.Request, uri, contentType string) {
	strippedURI := uri[1:]
	buf, err := fs.ReadFile(vueGlue.DistFS, strippedURI)
	if err != nil {
		buf, err = fs.ReadFile(vueGlue.DistFS, "dist/"+strippedURI)
	}

	if err == nil {
		w.Header().Add("Content-Type", contentType)
		w.Write(buf)
		return
	}

	w.WriteHeader(http.StatusNotFound)
}

func pageWithAVue(w http.ResponseWriter, r *http.Request) {
	re := regexp.MustCompile(`^/([^.]+)\.(svg|ico|jpg)$`)
	matches := re.FindStringSubmatch(r.RequestURI)
	if matches != nil {
		if vueGlue.Environment == "development" {
			log.Printf("vite logo requested")
			url := vueGlue.BaseURL + r.RequestURI
			http.Redirect(w, r, url, http.StatusPermanentRedirect)
			return
		} else {
			var contentType string
			switch matches[2] {
			case "svg":
				contentType = "image/svg+xml"
			case "ico":
				contentType = "image/x-icon"
			case "jpg":
				contentType = "image/jpeg"
			}

			serveOneFile(w, r, r.RequestURI, contentType)
			return
		}

	}

	t, err := template.ParseFiles("./frontend/template.tmpl")
	if err != nil {
		log.Fatal(err)
	}

	t.Execute(w, vueGlue)
}

func main() {

	config := &vueglue.ViteConfig{
		Environment: "production",
		AssetsPath:  "dist",
		EntryPoint:  "src/main.js",
		Platform:    "vue",
		FS:          os.DirFS("frontend/app"),
	}

	glue, err := vueglue.NewVueGlue(config)
	if err != nil {
		log.Fatalln(err)
		return
	}
	vueGlue = glue

	mux := http.NewServeMux()

	fsHandler, err := glue.FileServer()
	if err != nil {
		log.Println("could not set up static file server", err)
		return
	}

	mux.Handle(config.URLPrefix, fsHandler)
	mux.Handle("/", logRequest(http.HandlerFunc(pageWithAVue)))

	log.Println("Starting server on :4000")
	generatedConfig, _ := json.MarshalIndent(config, "", "  ")
	log.Println("Generated Configuration:\n", string(generatedConfig))
	err = http.ListenAndServe(":4000", mux)
	log.Fatal(err)
}
