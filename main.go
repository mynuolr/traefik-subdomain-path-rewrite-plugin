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
	typeName           = "SubdomainPathRewrite"
	ReplacedPathHeader = "X-Replaced-Path"
	ReplacedHostHeader = "X-Replaced-Host"
)

// Config the plugin configuration.
type Config struct {
	ReplacementHost string `json:"replacementHost,omitempty"`
	BasePath        string `json:"basePath,omitempty"`
	KeepPath        bool   `json:"keepPath,omitempty"`
	LogLevel        string `json:"logLevel,omitempty"`
}

// CreateConfig creates the default plugin configuration.
func CreateConfig() *Config {
	return &Config{
		LogLevel: "INFO",
		KeepPath: true,
	}
}

// ReplacePath is a middleware used to replace the path of a URL request.
type DynamicRewrite struct {
	next            http.Handler
	name            string
	replacementHost string
	basePath        string
	keepPath        bool
	hostRegex       *regexp.Regexp
	log             *logger.Log
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
		next:            next,
		name:            name,
		replacementHost: config.ReplacementHost,
		basePath:        config.BasePath,
		keepPath:        config.KeepPath,
		hostRegex:       hostRegex,
		log:             log,
	}, nil
}

func (dr *DynamicRewrite) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	host := req.Host
	dr.log.Info("Original Host: %s", host)
	dr.log.Info("Original Path: %s", req.URL.Path)
	dr.log.Info("Original URL: %s", req.URL.String())

	matches := dr.hostRegex.FindStringSubmatch(host)
	if len(matches) > 1 {
		dynamicIdentifier := matches[1]
		baseHost := host[len(dynamicIdentifier)+1:]
		dr.log.Info("Dynamic Identifier: %s", dynamicIdentifier)

		currentPath := req.URL.RawPath
		if currentPath == "" {
			currentPath = req.URL.EscapedPath()
		}
		req.Header.Add(ReplacedPathHeader, currentPath)

		originalURL := req.URL.String()

		var newHost string
		if dr.replacementHost != "" {
			newHost = dr.replacementHost
		} else {
			newHost = baseHost
		}

		if len(req.URL.Host) > 0 {
			req.URL.Host = newHost
		}
		req.Host = newHost
		req.Header.Add(ReplacedHostHeader, host)

		dr.log.Info("Rewritten Host from %s to %s", host, newHost)

		newPath := fmt.Sprintf("%s/%s", dr.basePath, dynamicIdentifier)
		if dr.keepPath {
			newPath += currentPath
		}
		req.URL.RawPath = newPath

		var err error
		req.URL.Path, err = url.PathUnescape(req.URL.RawPath)
		if err != nil {
			dr.log.Error("Unable to unescape url raw path %q: %v", req.URL.RawPath, err)
			http.Error(rw, err.Error(), http.StatusInternalServerError)
			return
		}

		dr.log.Info("Rewritten Path from %s to %s", currentPath, req.URL.Path)

		req.RequestURI = req.URL.RequestURI()

		dr.log.Info("Rewritten URL from %s to %s", originalURL, req.URL.String())
	} else {
		dr.log.Info("No dynamic identifier found in host: %s", host)
	}
	dr.next.ServeHTTP(rw, req)
}
