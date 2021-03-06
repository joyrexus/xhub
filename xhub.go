package xhub

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/joyrexus/buckets"
	"github.com/julienschmidt/httprouter"
)

const verbose = true // if `true` you'll see log output

// NewServer creates a new studies server instance.
func NewServer(addr, dbpath string) *Server {
	// Open a buckets database.
	bux, err := buckets.Open(dbpath)
	if err != nil {
		log.Fatalf("couldn't open buckets db %q: %v\n", dbpath, err)
	}

	// Initialize our controller for handling specific routes.
	control := NewController(addr, bux)

	// Create and setup our router.
	mux := httprouter.New()

	// Setup study handlers.
	mux.POST("/studies", control.Study.Post)
	mux.GET("/studies", control.Study.List)
	mux.GET("/studies/:study", control.Study.Get)
	mux.DELETE("/studies/:study", control.Study.Delete)

	// Setup trial handlers.
	mux.POST("/studies/:study/trials", control.Trial.Post)
	mux.GET("/studies/:study/trials", control.Trial.List)
	mux.GET("/studies/:study/trials/:trial", control.Trial.Get)
	mux.DELETE("/studies/:study/trials/:trial", control.Trial.Delete)

	// Setup study-level file handlers.
	mux.POST("/studies/:study/files", control.File.Post)
	mux.GET("/studies/:study/files", control.File.List)
	mux.GET("/studies/:study/files/:file", control.File.Get)
	mux.DELETE("/studies/:study/files/:file", control.File.Delete)

	// Setup trial-level file handlers.
	mux.POST("/files/:study/:trial", control.File.Post)
	mux.GET("/files/:study/:trial", control.File.List)
	mux.GET("/files/:study/:trial/:file", control.File.Get)
	mux.DELETE("/files/:study/:trial/:file", control.File.Delete)

	// Setup index/make/view/edit handlers.
	// mux.GET("/view/studies", control.Study.Index)
	// mux.GET("/make/studies", control.Study.Make)
	mux.POST("/save/studies", control.Study.Save)
	mux.GET("/view/studies/:study", control.Study.View)
	mux.GET("/edit/studies/:study", control.Study.Edit)

	return &Server{addr, mux, bux}
}

// A Server is an http handler providing the studies service API.
type Server struct {
	Addr    string
	handler *httprouter.Router
	db      *buckets.DB
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

// ListenAndServe starts the http service.
func (s *Server) ListenAndServe() error {
	return http.ListenAndServe(s.Addr, s.handler)
}

// Close closes the server's database.
func (s *Server) Close() {
	s.db.Close()
}

/* -- CONTROLLER -- */

// NewController initializes a new instance of our controller.
// It provides handler methods for our router.
func NewController(host string, bux *buckets.DB) *Controller {
	study := NewStudyController(host, bux)
	trial := NewTrialController(host, bux)
	file := NewFileController(host, bux)
	return &Controller{study, trial, file}
}

// A Controller provides handler methods for our router.
type Controller struct {
	Study *StudyController
	Trial *TrialController
	File  *FileController
}

/* -- MODELS --*/

// A Resource models an experimental resource.
type Resource struct {
	Version  string          `json:"version"`
	Type     string          `json:"resource"` // "study", "trial", "file"
	ID       string          `json:"id"`       // resource identifier/name
	URL      string          `json:"url"`      // resource url
	Data     json.RawMessage `json:"data"`
	Created  string          `json:"created,omitempty"`
	Children []string        `json:"children,omitempty"`
}
