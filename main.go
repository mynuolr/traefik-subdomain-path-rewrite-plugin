package traefik_subdomain_path_rewrite_plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"regexp"
	"strings"

	logger "github.com/lukas-r/traefik-subdomain-path-rewrite-plugin/pkg/logger"
	response_recorder "github.com/lukas-r/traefik-subdomain-path-rewrite-plugin/pkg/response_recorder"
)

const (
	typeName           = "SubdomainPathRewrite"
	ReplacedPathHeader = "X-Replaced-Path"
	ReplacedHostHeader = "X-Replaced-Host"
	FallbackURLHeader  = "X-Fallback-For"
)

// Config the plugin configuration.
type Config struct {
	RewriteSubdomain bool   `json:"rewriteSubdomain,omitempty"`
	ReplacementHost  string `json:"replacementHost,omitempty"`
	BasePath         string `json:"basePath,omitempty"`
	KeepPath         bool   `json:"keepPath,omitempty"`
	FallbackPath     string `json:"fallbackPath,omitempty"`
	LogLevel         string `json:"logLevel,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		RewriteSubdomain: true,
		KeepPath:         true,
		LogLevel:         "INFO",
	}
}

// ReplacePath is a middleware used to replace the path of a URL request.
type DynamicRewrite struct {
	next                  http.Handler
	name                  string
	rewriteSubdomain      bool
	replacementHost       string
	basePath              string
	keepPath              bool
	fallbackPathComponent string
	hostRegex             *regexp.Regexp
	log                   *logger.Log
}

// New creates a new replace path middleware.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	log := logger.New(config.LogLevel, fmt.Sprintf("[%s] ", typeName))

	hostRegex, err := regexp.Compile(`^(?P<identifier>[^\.]+)\..+$`)
	if err != nil {
		return nil, err
	}

	if config.BasePath != "" && config.BasePath[0] != '/' {
		config.BasePath = "/" + config.BasePath
	}

	return &DynamicRewrite{
		next:                  next,
		name:                  name,
		rewriteSubdomain:      config.RewriteSubdomain,
		replacementHost:       config.ReplacementHost,
		basePath:              config.BasePath,
		keepPath:              config.KeepPath,
		fallbackPathComponent: config.FallbackPath,
		hostRegex:             hostRegex,
		log:                   log,
	}, nil
}

func (dr *DynamicRewrite) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	host := req.Host
	dr.logRequestDetails(host, req)
	internalBasePath := dr.rewriteRequest(host, req)
	dr.handleResponse(rw, req, internalBasePath)
}

func (dr *DynamicRewrite) logRequestDetails(host string, req *http.Request) {
	dr.log.Info("Original Host: %s", host)
	dr.log.Info("Original Path: %s", req.URL.Path)
	dr.log.Info("Original URL: %s", req.URL.String())
}

func (dr *DynamicRewrite) rewriteRequest(host string, req *http.Request) string {
	dynamicIdentifier, baseHost := dr.extractDynamicIdentifierAndHost(host)
	basePath := "/"
	dr.log.Info("ReplacedHostHeader: %s", req.Header.Get(ReplacedHostHeader))
	if req.Header.Get(ReplacedHostHeader) == "" {
		dr.rewriteHost(req, baseHost)
	}
	dr.log.Info("ReplacedPathHeader: %s", req.Header.Get(ReplacedPathHeader))
	if req.Header.Get(ReplacedPathHeader) == "" {
		basePath = dr.rewritePath(req, dynamicIdentifier)
	}
	return basePath
}

func (dr *DynamicRewrite) extractDynamicIdentifierAndHost(host string) (string, string) {
	dynamicIdentifier := ""
	baseHost := host
	if dr.rewriteSubdomain {
		dynamicIdentifier, baseHost = dr.extractSubdomainAndHost(host)
		if dynamicIdentifier == "" {
			dr.log.Info("No dynamic identifier found in host: %s", host)
		}
	}
	return dynamicIdentifier, baseHost
}

func (dr *DynamicRewrite) extractSubdomainAndHost(host string) (string, string) {
	matches := dr.hostRegex.FindStringSubmatch(host)
	if len(matches) > 1 {
		dynamicIdentifier := matches[1]
		baseHost := host[len(dynamicIdentifier)+1:]
		dr.log.Info("Dynamic Identifier: %s", dynamicIdentifier)
		return dynamicIdentifier, baseHost
	}
	return "", host
}

func (dr *DynamicRewrite) rewriteHost(req *http.Request, baseHost string) {
	originalHost := req.Host
	newHost := dr.getReplacementHost(baseHost)
	req.Host = newHost
	req.Header.Add(ReplacedHostHeader, originalHost)
	dr.log.Info("Rewritten Host from %s to %s", originalHost, newHost)
}

func (dr *DynamicRewrite) getReplacementHost(baseHost string) string {
	if dr.replacementHost != "" {
		return dr.replacementHost
	}
	return baseHost
}

func (dr *DynamicRewrite) rewritePath(req *http.Request, dynamicIdentifier string) string {
	originalPath := req.URL.Path
	req.Header.Add(ReplacedPathHeader, originalPath)

	newPath, basePath := dr.buildNewPath(req, dynamicIdentifier)
	req.URL.RawPath = newPath

	var err error
	req.URL.Path, err = url.PathUnescape(req.URL.RawPath)
	if err != nil {
		dr.log.Error("Unable to unescape url raw path %q: %v", req.URL.RawPath, err)
	}

	dr.log.Info("Rewritten Path from %s to %s", originalPath, req.URL.Path)
	req.RequestURI = req.URL.RequestURI()
	dr.log.Info("Rewritten URL: %s", req.URL.String())

	return basePath
}

func (dr *DynamicRewrite) buildNewPath(req *http.Request, dynamicIdentifier string) (string, string) {
	if dynamicIdentifier != "" {
		dynamicIdentifier = "/" + dynamicIdentifier
	}
	basePath := fmt.Sprintf("%s%s", dr.basePath, dynamicIdentifier)
	newPath := basePath
	if dr.keepPath {
		newPath += req.URL.EscapedPath()
	} else {
		newPath += "/"
	}
	return newPath, basePath
}

func (dr *DynamicRewrite) recordResponse(_ http.ResponseWriter, req *http.Request) *response_recorder.ResponseRecorder {
	recorder := response_recorder.New()
	dr.next.ServeHTTP(recorder, req)
	dr.log.Info("Response status code: %d", recorder.StatusCode)
	return recorder
}

func (dr *DynamicRewrite) handleResponse(rw http.ResponseWriter, req *http.Request, internalBasePath string) {
	recorder := dr.recordResponse(rw, req)
	fallbackURL := dr.constructFallbackURL(req, internalBasePath)
	if dr.shouldUseFallback(req, recorder, fallbackURL) {
		dr.log.Info("Fetching fallback content from %s", fallbackURL.String())
		dr.respondWithFallback(rw, req, fallbackURL)
	} else {
		dr.respondWithOriginal(rw, recorder)
	}
}

func (dr *DynamicRewrite) constructFallbackURL(req *http.Request, internalBasePath string) *url.URL {
	scheme := "http"
	if req.TLS != nil {
		scheme = "https"
	}

	fallbackPath := dr.getFallbackPath(req.URL.Path, internalBasePath)
	return &url.URL{
		Scheme: scheme,
		Host:   req.Host,
		Path:   fallbackPath,
	}
}

func (dr *DynamicRewrite) getFallbackPath(originalPath string, internalBasePath string) string {
	if dr.fallbackPathComponent[0] != '/' {
		pathParts := strings.Split(originalPath, "/")
		pathParts[len(pathParts)-1] = dr.fallbackPathComponent
		return strings.Join(pathParts, "/")
	} else {
		return internalBasePath + dr.fallbackPathComponent
	}
}

func (dr *DynamicRewrite) shouldUseFallback(req *http.Request, recorder *response_recorder.ResponseRecorder, fallbackURL *url.URL) bool {
	return req.Header.Get(FallbackURLHeader) != fallbackURL.String() &&
		dr.fallbackPathComponent != "" &&
		recorder.StatusCode == http.StatusNotFound
}

func (dr *DynamicRewrite) respondWithFallback(rw http.ResponseWriter, req *http.Request, fallbackURL *url.URL) {
	proxy := &httputil.ReverseProxy{Director: func(r *http.Request) {}}
	newReq, _ := http.NewRequest(req.Method, fallbackURL.String(), req.Body)
	newReq.Header = req.Header.Clone()
	newReq.Header.Set(FallbackURLHeader, req.URL.String())
	proxy.ServeHTTP(rw, newReq)
}

func (dr *DynamicRewrite) respondWithOriginal(rw http.ResponseWriter, recorder *response_recorder.ResponseRecorder) {
	for key, values := range recorder.Header() {
		rw.Header()[key] = values
	}
	rw.WriteHeader(recorder.StatusCode)
	rw.Write(recorder.Body)
}
