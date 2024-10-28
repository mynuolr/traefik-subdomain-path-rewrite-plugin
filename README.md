
# Traefik Subdomain to Path Rewrite Plugin

This Traefik plugin allows dynamic rewriting of incoming request URLs by modifying the host and path based on specified configurations. Specifically, it is designed to rewrite subdomains into paths, enabling more flexible routing. It uses regular expression-based dynamic URL rewriting, which can be customized for various routing needs in a Traefik reverse proxy setup.

## Features

- **Dynamic Host Replacement**: Replace the host of incoming requests based on a regex pattern.
- **Path Rewriting**: Rewrite the URL path by adding a configurable base path along with an identifier extracted from the host.s
- **Custom Headers**: Adds custom headers `X-Replaced-Path` and `X-Replaced-Host` for tracking original values.
- **Logging Support**: Adjustable log level for detailed debug information.

## Configuration

The plugin accepts the following configuration parameters:

- `replacementHost` (optional): The host to replace the original request host. If not provided, the host will remain unchanged.
- `basePath` (optional): A base path to prepend to the rewritten URL.
- `keepPath` (optional): Decides if the original request path will be appended to the rewritten URL, default is `true`.
- `logLevel` (optional): Sets the logging level, default is `INFO`.

### Example Configuration

#### Static Configuration

```yaml
experimental:
  plugins:
    traefik-subdomain-path-rewrite-plugin:
      moduleName: "github.com/lukas-r/traefik-subdomain-path-rewrite-plugin"
      version: "v0.2.0"
```

#### Dynamic Configuration

```yaml
http:
  middlewares:
    my-dynamic-rewrite:
      plugin:
        traefik-subdomain-path-rewrite-plugin:
          replacementHost: "example.com"
          basePath: "/service"
          keepPah: true
          logLevel: "DEBUG"
```

## Routing Example

The following table illustrates example routings based on the configuration above:

| Original URL                | Rewritten URL                        |
|-----------------------------|--------------------------------------|
| `http://foo.mypage.org/`    | `http://example.com/service/foo/`    |
| `http://bar.mypage.org/`    | `http://example.com/service/bar/`    |
| `http://baz.example.com/abc`| `http://example.com/service/baz/abc` |

## Headers Added

The middleware adds two headers to the request:

- **`X-Replaced-Path`**: The original path before rewrite.
- **`X-Replaced-Host`**: The original host before rewrite.
