package traefik_subdomain_path_rewrite_plugin

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"

	logger "github.com/lukas-r/traefik-subdomain-path-rewrite-plugin/pkg/logger"
)

const (
	typeName           = "DynamicRewrite"
	ReplacedPathHeader = "X-Replaced-Path"
	ReplacedHostHeader = "X-Replaced-Host"
)

// Config the plugin configuration.
type Config struct {
	ReplacementHost string `json:"replacementHost,omitempty"`
	BasePath        string `json:"basePath,omitempty"`
	LogLevel        string `json:"logLevel,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		LogLevel: "INFO",
	}
}

// ReplacePath is a middleware used to replace the path of a URL request.
type DynamicRewrite struct {
	next            http.Handler
	name            string
	replacementHost string
	basePath        string
	hostRegex       *regexp.Regexp
	log             *logger.Log
}

// New creates a new replace path middleware.
func New(ctx context.Context, next http.Handler, config *Config, name string) (http.Handler, error) {
	log := logger.New(config.LogLevel)

	hostRegex, err := regexp.Compile(`^(?P<identifier>[^\.]+)\..+$`)
	if err != nil {
		return nil, err
	}

	if config.BasePath != "" && config.BasePath[0] != '/' {
		config.BasePath = "/" + config.BasePath
	}

	return &DynamicRewrite{
		next:            next,
		name:            name,
		replacementHost: config.ReplacementHost,
		basePath:        config.BasePath,
		hostRegex:       hostRegex,
		log:             log,
	}, nil
}

func (dr *DynamicRewrite) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	host := req.Host
	dr.log.Info("[DynamicRewrite] Original Host: %s", host)

	matches := dr.hostRegex.FindStringSubmatch(host)
	if len(matches) > 1 {
		dynamicIdentifier := matches[1]
		dr.log.Info("[DynamicRewrite] Dynamic Identifier: %s", dynamicIdentifier)

		currentPath := req.URL.RawPath
		if currentPath == "" {
			currentPath = req.URL.EscapedPath()
		}
		req.Header.Add(ReplacedPathHeader, currentPath)

		originalURL := req.URL.String()

		if dr.replacementHost != "" {
			if len(req.URL.Host) > 0 {
				req.URL.Host = dr.replacementHost
			}
			req.Host = dr.replacementHost
			req.Header.Add(ReplacedHostHeader, host)
		}

		req.URL.RawPath = fmt.Sprintf("%s/%s%s", dr.basePath, dynamicIdentifier, currentPath)

		var err error
		req.URL.Path, err = url.PathUnescape(req.URL.RawPath)
		if err != nil {
			dr.log.Error("Unable to unescape url raw path %q: %v", req.URL.RawPath, err)
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		req.RequestURI = req.URL.RequestURI()

		dr.log.Info("[DynamicRewrite] Rewritten URL from %s to %s", originalURL, req.URL.String())
	} else {
		dr.log.Info("[DynamicRewrite] No dynamic identifier found in host: %s", host)
	}
	dr.next.ServeHTTP(rw, req)
}
