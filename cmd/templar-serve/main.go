package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	gotl "github.com/panyam/templar"
)

func NewTemplateGroup(folders []string) *gotl.TemplateGroup {
	templates := gotl.NewTemplateGroup()
	if len(folders) == 0 {
		folders = multiStringFlag{"./templates"}
	}

	log.Println("Registering template folders: ", folders)
	templates.Loader = (&gotl.LoaderList{}).AddLoader(gotl.NewFileSystemLoader(folders...))
	// templates.AddFuncs(gotl.DefaultFuncMap())
	return templates
}

type multiStringFlag []string

func (m *multiStringFlag) String() string {
	return strings.Join(*m, ", ")
}

func (m *multiStringFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

var (
	addr = flag.String("addr", ":7777", "Address where the http server will be running")
)

func main() {
	var templateDirs multiStringFlag
	var staticFolders multiStringFlag
	flag.Var(&templateDirs, "t", "List of template directories to load templates from")
	flag.Var(&staticFolders, "s", "List of static directores and http static prefixes in the form <http prefix>:<local folder>")
	flag.Parse()
	ctx := context.Background()

	templates := NewTemplateGroup(templateDirs)

	mux := registerStaticFolders(nil, staticFolders)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Path: ", r.URL.Path)
		template := r.URL.Path[1:]
		tmpl, err := templates.Loader.Load(template, "")
		if err != nil {
			log.Println("Template Load Error: ", err)
			fmt.Fprint(w, "Error rendering: ", err.Error())
		} else {
			log.Println("Got Template: ", tmpl)
			templates.RenderHtmlTemplate(w, tmpl[0], template, map[string]any{}, nil)
		}
	})
	server := &http.Server{
		Addr:        *addr,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
		Handler:     mux,
	}
	log.Println("Starting server on: ", *addr)
	server.ListenAndServe()
}

func registerStaticFolders(mux *http.ServeMux, staticFolders []string) *http.ServeMux {
	if mux == nil {
		mux = http.NewServeMux()
	}
	// Setup static folders
	// setup some defaults
	if len(staticFolders) == 0 {
		staticFolders = multiStringFlag{"static:./static"}
	}

	log.Println("Registering static folders: ", staticFolders)
	for _, statics := range staticFolders {
		parts := strings.Split(statics, ":")
		prefix := parts[0]
		localfolder := parts[1]
		if strings.HasPrefix(prefix, "/") {
			prefix = prefix[1:]
		}
		prefix = "/" + prefix + "/"
		mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(http.Dir(localfolder))))
	}
	return mux
}
