package xchgr_server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/ipoluianov/gomisc/logger"
)

type HttpServer struct {
	srv      *http.Server
	r        *mux.Router
	server   *Router
	stopping bool
}

func CurrentExePath() string {
	dir, _ := filepath.Abs(filepath.Dir(os.Args[0]))
	return dir
}

func NewHttpServer() *HttpServer {
	var c HttpServer
	return &c
}

func (c *HttpServer) Start(server *Router) {
	c.server = server

	c.r = mux.NewRouter()
	c.r.HandleFunc("/api/request", c.processApiRequest)
	c.r.NotFoundHandler = http.HandlerFunc(c.processFile)
	c.srv = &http.Server{
		Addr: ":8084",
	}

	c.srv.Handler = c.r
	go c.thListen()
}

func (c *HttpServer) thListen() {
	err := c.srv.ListenAndServe()
	if err != nil {
		logger.Println("HttpServer thListen error: ", err)
	}
	logger.Println("HttpServer thListen end")
}

func (c *HttpServer) Stop() error {
	var err error

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	if err = c.srv.Shutdown(ctx); err != nil {
		logger.Println(err)
	}
	return err
}

func (c *HttpServer) processApiRequest(w http.ResponseWriter, r *http.Request) {
	var err error
	var responseText []byte
	function := r.FormValue("fn")

	if r.Method == "POST" {
		if err := r.ParseMultipartForm(1000000); err != nil {
			fmt.Fprintf(w, "ParseForm() err: %v", err)
			return
		}
		function = r.FormValue("fn")
	}

	//fmt.Println("processApiRequest", function)

	switch function {
	case "d":
		responseText, err = c.processD(w, r)
	case "debug":
		responseText, err = c.processDebug(w, r)
	}

	if err != nil {
		w.WriteHeader(500)
		b := []byte(err.Error())
		_, _ = w.Write(b)
		return
	}

	_, _ = w.Write([]byte(responseText))
}

func (c *HttpServer) processDebug(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	return
}

func (c *HttpServer) processD(w http.ResponseWriter, r *http.Request) (result []byte, err error) {
	data64 := r.FormValue("d")
	var dataBS []byte
	dataBS, err = base64.StdEncoding.DecodeString(data64)
	if err != nil {
		return
	}
	var resultBS []byte
	resultBS, err = c.server.processFrames(dataBS)
	if err != nil {
		return
	}
	resultStr := base64.StdEncoding.EncodeToString(resultBS)
	result = []byte(resultStr)
	return
}

func SplitRequest(path string) []string {
	return strings.FieldsFunc(path, func(r rune) bool {
		return r == '/'
	})
}

func (c *HttpServer) contentTypeByExt(ext string) string {
	var builtinTypesLower = map[string]string{
		".css":  "text/css; charset=utf-8",
		".gif":  "image/gif",
		".htm":  "text/html; charset=utf-8",
		".html": "text/html; charset=utf-8",
		".jpeg": "image/jpeg",
		".jpg":  "image/jpeg",
		".js":   "text/javascript; charset=utf-8",
		".mjs":  "text/javascript; charset=utf-8",
		".pdf":  "application/pdf",
		".png":  "image/png",
		".svg":  "image/svg+xml",
		".wasm": "application/wasm",
		".webp": "image/webp",
		".xml":  "text/xml; charset=utf-8",
	}

	logger.Println("Ext: ", ext)

	if ct, ok := builtinTypesLower[ext]; ok {
		return ct
	}
	return "text/plain"
}

func (c *HttpServer) processFile(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("wrong request"))
}

func getRealAddr(r *http.Request) string {

	remoteIP := ""
	// the default is the originating ip. but we try to find better options because this is almost
	// never the right IP
	if parts := strings.Split(r.RemoteAddr, ":"); len(parts) == 2 {
		remoteIP = parts[0]
	}
	// If we have a forwarded-for header, take the address from there
	if xff := strings.Trim(r.Header.Get("X-Forwarded-For"), ","); len(xff) > 0 {
		addrs := strings.Split(xff, ",")
		lastFwd := addrs[len(addrs)-1]
		if ip := net.ParseIP(lastFwd); ip != nil {
			remoteIP = ip.String()
		}
		// parse X-Real-Ip header
	} else if xri := r.Header.Get("X-Real-Ip"); len(xri) > 0 {
		if ip := net.ParseIP(xri); ip != nil {
			remoteIP = ip.String()
		}
	}

	return remoteIP

}

func (c *HttpServer) redirect(w http.ResponseWriter, r *http.Request, url string) {
	w.Header().Set("Cache-Control", "no-cache, private, max-age=0")
	w.Header().Set("Expires", time.Unix(0, 0).Format(http.TimeFormat))
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("X-Accel-Expires", "0")
	http.Redirect(w, r, url, 307)
}
