# httpipe
--
    import "github.com/fasterness/httpipe"

Package httpipe provides a general-purpose pipeline for manipulating HTTP
requests and responses to an upstream server. Written for educational purposes.
Based largely on https://github.com/elazarl/goproxy

## Usage

#### func  NewResponse

```go
func NewResponse(r *http.Request, contentType string, status int, body string) *http.Response
```
NewResponse creates a new http.Response object with sane defaults

#### type Context

```go
type Context struct {
	Request  *http.Request
	Response *http.Response
	Error    error
	Body     []byte
	Session  int64
	Server   *Server
}
```

Context keeps track of the request

#### func (*Context) RoundTrip

```go
func (ctx *Context) RoundTrip(req *http.Request) (*http.Response, error)
```
RoundTrip initiates the call to the upstream resource

#### type RequestHandler

```go
type RequestHandler interface {
	Handle(req *http.Request, ctx *Context) (*http.Request, *http.Response)
}
```

RequestHandler will be called by ServeHTTP before the request is made to the
upstream server. If the returned response is not nil, it will short-circuit the
request and the response will be sent to the client

#### type RequestWrapper

```go
type RequestWrapper func(req *http.Request, ctx *Context) (*http.Request, *http.Response)
```

RequestWrapper will take a function with a signature matching RequestHandler's
Handle function and return it as a RequestHandler type

#### func (RequestWrapper) Handle

```go
func (f RequestWrapper) Handle(req *http.Request, ctx *Context) (*http.Request, *http.Response)
```

#### type ResponseHandler

```go
type ResponseHandler interface {
	Handle(resp *http.Response, ctx *Context) *http.Response
}
```

ResponseHandler will be called by ServeHTTP whenever a response is returned to a
client

#### type ResponseWrapper

```go
type ResponseWrapper func(resp *http.Response, ctx *Context) *http.Response
```

ResponseWrapper will take a function with a signature matching ResponseHandler's
Handle function and return it as a ResponseHandler type

#### func (ResponseWrapper) Handle

```go
func (f ResponseWrapper) Handle(resp *http.Response, ctx *Context) *http.Response
```

#### type Server

```go
type Server struct {
	Upstream         *url.URL
	RequestHandlers  []RequestHandler
	ResponseHandlers []ResponseHandler
	Transport        *http.Transport
	Session          int64
}
```

Server implements the http.Handler interface. It will call all RequestHandlers
before the request is made (or until a response is returned) and call all
ResponseHandlers for the returned response.

#### func  New

```go
func New(upstream string) *Server
```
New returns a pointer to an initialized Server instance

#### func (*Server) HandleRequest

```go
func (server *Server) HandleRequest(r *http.Request, ctx *Context) (req *http.Request, resp *http.Response)
```
HandleRequest ranges over each RequestHandler, breaking the loop if a response
is returned

#### func (*Server) HandleResponse

```go
func (server *Server) HandleResponse(orig *http.Response, ctx *Context) (resp *http.Response)
```
HandleResponse ranges over each response handler

#### func (*Server) ServeHTTP

```go
func (server *Server) ServeHTTP(w http.ResponseWriter, r *http.Request)
```
ServeHTTP manages the round trip of the request
