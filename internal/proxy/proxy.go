package proxy

import (
	"io"
	"mime"
	"net"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/kunalvirwal/minato/internal/utils"
)

type RevProxy struct {
	Transport       *http.Transport
	RequestModifier func(*http.Request)
	BufferPool      *sync.Pool
}

var hopByHopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"TE",
	"Trailer",
	"Transfer-Encoding",
	"Upgrade",
}

// NewRevProxy creates a new RevProxy object for a particular upstream backend
func NewRevProxy(backendURL *url.URL) *RevProxy {
	// This function modifies the URL of any request to a particular backend URL
	modifier := func(req *http.Request) {
		ModifyRequestURL(req, backendURL)
	}
	return &RevProxy{
		Transport:       CreateTransport(),
		RequestModifier: modifier,
		BufferPool:      CreateBufferPool(),
	}
}

func ModifyRequestURL(r *http.Request, backendURL *url.URL) {
	backendQueryParams := backendURL.RawQuery
	r.URL.Scheme = backendURL.Scheme
	r.URL.Host = backendURL.Host
	r.URL.Path, r.URL.RawPath = joinURLPath(backendURL, r.URL)
	if backendQueryParams == "" || r.URL.RawQuery == "" {
		r.URL.RawQuery = backendQueryParams + r.URL.RawQuery
	} else {
		r.URL.RawQuery = backendQueryParams + "&" + r.URL.RawQuery
	}
}

// Create a Buffer Pool to reuse buffers for response copying
func CreateBufferPool() *sync.Pool {
	return &sync.Pool{
		New: func() any {
			buf := make([]byte, 4096) // 4KB buffer
			return buf
		},
	}
}

// Returns the Path and the RawPath formed form the backend and req paths
func joinURLPath(a, b *url.URL) (string, string) {
	// Both upstream and req don't use RawPath
	if a.RawPath == "" && b.RawPath == "" {
		first := a.Path
		second := b.Path
		aslash := strings.HasPrefix(first, "/")
		bslash := strings.HasSuffix(second, "/")

		if aslash && bslash {
			second = second[1:]
		} else if !aslash && !bslash {
			second = "/" + second
		}
		return first + second, ""
	}

	// Any one or both use RawPaths
	first := a.EscapedPath()
	second := b.EscapedPath()
	aslash := strings.HasPrefix(first, "/")
	bslash := strings.HasSuffix(second, "/")

	if aslash && bslash {
		return a.Path + b.Path[1:], first + second[1:]
	} else if !aslash && !bslash {
		return a.Path + "/" + b.Path, first + "/" + second
	}
	return a.Path + b.Path, first + second

}

func CreateTransport() *http.Transport {

	dialer := &net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	return &http.Transport{
		Proxy:                  nil, // can change from nil to http.ProxyFromEnvironment if needed
		DialContext:            dialer.DialContext,
		ForceAttemptHTTP2:      true,
		MaxIdleConnsPerHost:    10,
		MaxConnsPerHost:        100,
		IdleConnTimeout:        90 * time.Second,
		TLSHandshakeTimeout:    10 * time.Second,
		ExpectContinueTimeout:  1 * time.Second,
		ResponseHeaderTimeout:  10 * time.Second,
		DisableCompression:     false,   // Disable automatic golang gzip if you want to preserve raw response
		MaxResponseHeaderBytes: 2 << 20, // 2MB
	}
}

// This function does not support 1xx response codes or switching of protocols
func (p *RevProxy) ServeRequest(w http.ResponseWriter, r *http.Request) {

	// For incoming requests, ctx cancels when connection to client closes.
	// In that case the outbound request should also be cancelled.
	// So they share the same context.
	ctx := r.Context()
	outReq := r.Clone(ctx)

	// If the content length is 0 we can set body = nil.
	// This way http.Transport is safe to retry any POST requests without body.
	// We should not retry requests with bodies as if a connection breaks mid transmission of body,
	// upon retry body will be resent.
	if r.ContentLength == 0 {
		outReq.Body = nil
	}

	// If this handler returns before transport has finished reading the body,
	// transport might reference this handler's stack.
	// Calling Body.Close() indicates that this handler has completed.
	// And signals transport to finish reading gracefully
	if outReq.Body != nil {
		defer outReq.Body.Close()
	}

	// Many Go Http functions manipulate the Header so its safer to create if there isn't one.
	if outReq.Header == nil {
		outReq.Header = make(http.Header)
	}

	// So that the backend does not close connection after a request
	outReq.Close = false

	// Modify the outbound request's URL to point to backend
	p.RequestModifier(outReq)

	// Remove inconsistencies like ; inplace of & or invalid encodings
	if outReq.Form != nil {
		outReq.URL.RawQuery = validateQuery(outReq.URL.RawQuery)
	}

	// Remove Hop-by-hop headers
	removeHopByHopHeaders(outReq.Header)

	// Append the client's IP to X-Forwarded-For if X-Forwarded-For is not nill
	clientIP, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		xffChain, ok := outReq.Header["X-Forwarded-For"]
		omit := ok && xffChain == nil
		if len(xffChain) > 0 {
			clientIP = strings.Join(xffChain, ", ") + ", " + clientIP
		}
		if !omit {
			outReq.Header.Set("X-Forwarded-For", clientIP)
		}
	}

	// If user's req does not have a User-Agent set then set it to "" and not Go's default
	if _, ok := outReq.Header["User-Agent"]; !ok {
		outReq.Header.Set("User-Agent", "")
	}

	// sending the request to backend, returns when it gets headers
	res, err := p.Transport.RoundTrip(outReq)
	if err != nil {
		utils.LogNewError("http: proxy error:" + err.Error())
		w.WriteHeader(http.StatusBadGateway)
		return
	}

	// res.Body is never nil
	defer res.Body.Close()

	// Remove Hop-by-hop headers from backend response
	removeHopByHopHeaders(res.Header)

	// Copy headers from res.Header to w
	for key, headers := range res.Header {
		for _, header := range headers {
			w.Header().Add(key, header)
		}
	}

	// Send the status code
	w.WriteHeader(res.StatusCode)

	// Copy the response body to the client
	p.copyResponse(w, res)

}

func (p *RevProxy) copyResponse(w http.ResponseWriter, res *http.Response) {
	var continuousFlush bool = false
	resType := res.Header.Get("Content-Type")
	baseCT, _, _ := mime.ParseMediaType(resType)
	if baseCT == "text/event-stream" || res.ContentLength == -1 {
		continuousFlush = true
	}

	var buf []byte

	// Only used for Streaming response
	var flusher http.Flusher
	if continuousFlush {
		var ok bool
		flusher, ok = w.(http.Flusher)
		if !ok {
			utils.LogNewError("ResponseWriter does not support streaming (Flush) ")
			return
		}
		// SSE buffers are long lived so we create a new one instead of fetching from pool
		buf = make([]byte, 4096)
	} else {
		// Fetching buffer from pool for normal responses
		buf = p.BufferPool.Get().([]byte)
		defer p.BufferPool.Put(buf)

	}

	for {
		n, err := res.Body.Read(buf)
		if n > 0 {
			// Write event chunk the n bytes read
			nw, err := w.Write(buf[:n])
			if err != nil {
				utils.LogNewError("Error writing to response: " + err.Error())
				return
			}
			if nw != n {
				utils.LogNewError("Less bytes written to response than read from body: " + io.ErrShortWrite.Error())
				return
			}
			// Flush after every write
			if continuousFlush && flusher != nil {
				flusher.Flush()
			}
		}
		if err != nil {
			if err != io.EOF {
				utils.LogNewError("Error reading response body: " + err.Error())
				return
			}
			break
		}
	}

}

// Remove Hop-by-hop headers
func removeHopByHopHeaders(Headers http.Header) {
	for _, header := range Headers["Connection"] {
		for _, h := range strings.Split(header, ",") {
			// Using textproto.TrimString because strings.TrimSpace does not follow Http standards
			if h = textproto.TrimString(h); h != "" {
				Headers.Del(h)
			}
		}
	}

	for _, h := range hopByHopHeaders {
		Headers.Del(h)
	}
}

func validateQuery(s string) string {
	for i := 0; i < len(s); {
		switch s[i] {
		case ';':
			v, _ := url.ParseQuery(s)
			return v.Encode()
		case '%':
			if i+2 >= len(s) || !ishex(s[i+1]) || !ishex(s[i+2]) {
				v, _ := url.ParseQuery(s)
				return v.Encode()
			}
			i += 3
		default:
			i++
		}
	}
	return s
}

func ishex(c byte) bool {
	switch {
	case '0' <= c && c <= '9':
		return true
	case 'a' <= c && c <= 'f':
		return true
	case 'A' <= c && c <= 'F':
		return true
	}
	return false
}
