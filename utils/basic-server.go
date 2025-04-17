package utils

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"

	"github.com/panyam/templar"
)

type BasicServer struct {
	StaticDirs   []string
	TemplateDirs []string
	FuncMaps     []map[string]any
	Templates    *templar.TemplateGroup
	mux          *http.ServeMux
}

func (b *BasicServer) Init() {
	b.Templates = templar.NewTemplateGroup()
	if len(b.TemplateDirs) == 0 {
		b.TemplateDirs = []string{"./templates"}
	}

	log.Println("Registering template folders: ", b.TemplateDirs)
	b.Templates.Loader = (&templar.LoaderList{}).AddLoader(templar.NewFileSystemLoader(b.TemplateDirs...))
	for _, fm := range b.FuncMaps {
		b.Templates.AddFuncs(fm)
	}

	b.createMux()
}

func (b *BasicServer) createMux() {
	b.mux = http.NewServeMux()
	// Setup static folders
	// setup some defaults
	if len(b.StaticDirs) == 0 {
		b.StaticDirs = []string{"static:./static"}
	}

	staticDirs := b.StaticDirs

	log.Println("Registering static folders: ", staticDirs)
	for _, statics := range staticDirs {
		parts := strings.Split(statics, ":")
		prefix := parts[0]
		localfolder := parts[1]
		if strings.HasPrefix(prefix, "/") {
			prefix = prefix[1:]
		}
		prefix = "/" + prefix + "/"
		b.mux.Handle(prefix, http.StripPrefix(prefix, http.FileServer(http.Dir(localfolder))))
	}

	b.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		log.Println("Path: ", r.URL.Path)
		template := r.URL.Path[1:]
		tmpl, err := b.Templates.Loader.Load(template, "")
		if err != nil {
			log.Println("Template Load Error: ", err)
			fmt.Fprint(w, "Error rendering: ", err.Error())
		} else {
			log.Println("Got Template: ", tmpl)
			b.Templates.RenderHtmlTemplate(w, tmpl[0], template, map[string]any{}, nil)
		}
	})
}

func (b *BasicServer) Serve(ctx context.Context, addr string) error {
	b.Init()

	if ctx == nil {
		ctx = context.Background()
	}

	server := &http.Server{
		Addr:        addr,
		BaseContext: func(_ net.Listener) context.Context { return ctx },
		Handler:     b.mux,
	}
	log.Println("Starting server on: ", addr)
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal("error starting server: ", err)
	}
	return err
}
