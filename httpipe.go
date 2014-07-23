// Package httpipe provides a general-purpose pipeline for manipulating
// HTTP requests and responses to an upstream server. Written for
// educational purposes. Based largely on https://github.com/elazarl/goproxy
package httpipe

import (
	"bytes"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"sync/atomic"
)

//Context keeps track of the request
type Context struct {
	Request  *http.Request
	Response *http.Response
	Error    error
	Body     []byte
	Session  int64
	Server   *Server
}

//RoundTrip initiates the call to the upstream resource
func (ctx *Context) RoundTrip(req *http.Request) (*http.Response, error) {
	return ctx.Server.Transport.RoundTrip(req)
}

//RequestHandler will be called by ServeHTTP before the
//request is made to the upstream server. If the returned
//response is not nil, it will short-circuit the request and
//the response will be sent to the client
type RequestHandler interface {
	Handle(req *http.Request, ctx *Context) (*http.Request, *http.Response)
}

//RequestWrapper will take a function with a signature
//matching RequestHandler's Handle function and return
//it as a RequestHandler type
type RequestWrapper func(req *http.Request, ctx *Context) (*http.Request, *http.Response)

func (f RequestWrapper) Handle(req *http.Request, ctx *Context) (*http.Request, *http.Response) {
	return f(req, ctx)
}

//ResponseHandler will be called by ServeHTTP whenever
//a response is returned to a client
type ResponseHandler interface {
	Handle(resp *http.Response, ctx *Context) *http.Response
}

//ResponseWrapper will take a function with a signature
//matching ResponseHandler's Handle function and return
//it as a ResponseHandler type
type ResponseWrapper func(resp *http.Response, ctx *Context) *http.Response

func (f ResponseWrapper) Handle(resp *http.Response, ctx *Context) *http.Response {
	return f(resp, ctx)
}

//Server implements the http.Handler interface.
//It will call all RequestHandlers before the request is
//made (or until a response is returned) and call all
//ResponseHandlers for the returned response.
type Server struct {
	Upstream         *url.URL
	RequestHandlers  []RequestHandler
	ResponseHandlers []ResponseHandler
	Transport        *http.Transport
	Session          int64
}

//New returns a pointer to an initialized Server instance
func New(upstream string) *Server {
	sURL, err := url.Parse(upstream)
	if err != nil {
		panic(err)
	}
	server := Server{
		RequestHandlers:  []RequestHandler{},
		ResponseHandlers: []ResponseHandler{},
		Upstream:         sURL,
		Transport:        &http.Transport{Proxy: http.ProxyFromEnvironment},
	}
	return &server
}

//HandleRequest ranges over each RequestHandler, breaking
//the loop if a response is returned
func (server *Server) HandleRequest(r *http.Request, ctx *Context) (req *http.Request, resp *http.Response) {
	req = r
	for _, h := range server.RequestHandlers {
		if req, resp = h.Handle(r, ctx); resp != nil {
			break
		}
	}
	return
}

//HandleResponse ranges over each response handler
func (server *Server) HandleResponse(orig *http.Response, ctx *Context) (resp *http.Response) {
	resp = orig
	for _, h := range server.ResponseHandlers {
		ctx.Response = h.Handle(resp, ctx)
	}
	return
}

//ServeHTTP manages the round trip of the request
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	nURL := *server.Upstream
	nURL.Path = r.URL.Path
	nURL.RawQuery = r.URL.RawQuery
	nURL.User = r.URL.User
	// oURL := r.URL
	r.URL = &nURL
	log.Printf("Request for %v", r.URL)
	ctx := &Context{Request: r, Session: atomic.AddInt64(&server.Session, 1), Server: server}
	r, resp := server.HandleRequest(r, ctx)
	if resp == nil {
		//nobody offered up a cached response, so make the round-trip.
		resp, err := ctx.RoundTrip(r)
		if err != nil {
			//we're here because you broke something
			ctx.Error = err
			resp = server.HandleResponse(nil, ctx)
			if resp == nil {
				//There's nothing in the desert. And no man needs nothing.
				http.Error(w, err.Error(), 500)
				return
			}
		}
		ctx.Body, err = ioutil.ReadAll(resp.Body)
		resp = server.HandleResponse(resp, ctx)

		resp.Header.Del("Content-Length")
		copyHeaders(w.Header(), resp.Header)
		w.WriteHeader(resp.StatusCode)
		// _, err = io.Copy(w, resp.Body)
		w.Write(ctx.Body)
		if err := resp.Body.Close(); err != nil {
			log.Printf("IO error sending response %v", err)
		}
	}
}

//NewResponse creates a new http.Response object with sane defaults
func NewResponse(r *http.Request, contentType string, status int, body string) *http.Response {
	resp := &http.Response{}
	resp.Request = r
	resp.TransferEncoding = r.TransferEncoding
	resp.Header = make(http.Header)
	resp.Header.Add("Content-Type", contentType)
	resp.StatusCode = status
	buf := bytes.NewBufferString(body)
	resp.ContentLength = int64(buf.Len())
	resp.Body = ioutil.NopCloser(buf)
	return resp
}
func copyHeaders(dst, src http.Header) {
	for k, _ := range dst {
		dst.Del(k)
	}
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}
