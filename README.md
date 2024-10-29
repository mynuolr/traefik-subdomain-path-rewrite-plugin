# Traefik Subdomain Path Rewrite Plugin

This plugin for Traefik enables dynamic path rewriting based on subdomains and provides fallback capabilities for handling 404 responses. It's particularly useful for scenarios where you want to map subdomain-based URLs to path-based routes while maintaining flexibility in URL structure.

## Features

- Rewrite subdomain-based URLs to path-based routes
- Optional path preservation after rewriting
- Configurable base path for all rewrites
- Custom host replacement
- Fallback path handling for 404 responses
- Detailed request tracking through custom headers

## Configuration

### Static Configuration

Static configuration is defined in your Traefik installation and enables the plugin. This is typically done in your `traefik.yml` file or through environment variables.

```yaml
experimental:
  plugins:
    subdomainPathRewrite:
      moduleName: "github.com/lukas-r/traefik-subdomain-path-rewrite-plugin"
      version: "v0.3.1"
```

### Dynamic Configuration

Dynamic configuration can be modified at runtime and controls the plugin's behavior for specific routers.

#### Parameters

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| rewriteSubdomain | bool | true | Enable/disable subdomain extraction and rewriting |
| replacementHost | string | "" | Override the target host. If empty, uses the base domain |
| basePath | string | "" | Base path prefix for all rewritten URLs |
| keepPath | bool | true | Preserve the original path after the rewritten portion |
| fallbackPath | string | "" | Path to try when original request returns 404 |
| logLevel | string | "INFO" | Logging level (INFO, DEBUG, ERROR) |

#### Detailed Parameter Behavior

##### rewriteSubdomain

- When `true`: Extracts the first subdomain segment and uses it in the path rewrite
- When `false`: No subdomain extraction occurs, other rewrite rules still apply
- Example: With `true`, `customer.example.com` becomes `example.com/customer`
- Subdomain extraction uses regex to capture everything before the first dot

##### replacementHost

- If set: Replaces the entire host with this value
- If empty: Uses the original host minus the extracted subdomain
- Example: With `replacementHost: "api.internal"`, `customer.example.com` becomes `api.internal`
- Useful for routing to internal services or different domains

##### basePath

- Prefixed to all rewritten paths
- If provided without leading slash, one is automatically added
- Applies before the subdomain segment in the final path
- Example: With `basePath: "/api"`, the path becomes `/api/customer/...`

##### keepPath

- When `true`: Preserves the original request path after the rewritten portion
- When `false`: Only uses the rewritten portion, ending with a slash
- Example with `true`: `/original/path` becomes `/api/customer/original/path`
- Example with `false`: `/original/path` becomes `/api/customer/`

##### fallbackPath

The fallbackPath parameter has two distinct behaviors based on whether it starts with a slash:

1. Absolute Path (starts with slash):
   - Appended to the base rewritten path
   - Original path is completely ignored
   - Example:
     - `fallbackPath: "/default"`
     - Original: `customer.example.com/not/found`
     - Fallback: `example.com/api/customer/default`

2. Relative Path (no starting slash):
   - Replaces only the last segment of the original path
   - Preserves the rest of the path structure
   - Example:
     - `fallbackPath: "default"`
     - Original: `customer.example.com/products/not-found`
     - Fallback: `example.com/api/customer/products/default`

##### logLevel

- "INFO": Standard operational logging
- "DEBUG": Detailed request/response information
- "ERROR": Only error conditions
- Affects the verbosity of plugin logs

### Example Configuration (Docker Labels)

```yaml
services:
  my-service:
    labels:
      - "traefik.enable=true"
      # Enable the plugin for this router
      - "traefik.http.routers.my-service.middlewares=subdomain-rewrite"
      - "traefik.http.middlewares.subdomain-rewrite.plugin.subdomainPathRewrite.rewriteSubdomain=true"
      - "traefik.http.middlewares.subdomain-rewrite.plugin.subdomainPathRewrite.basePath=/api"
      - "traefik.http.middlewares.subdomain-rewrite.plugin.subdomainPathRewrite.keepPath=true"
      - "traefik.http.middlewares.subdomain-rewrite.plugin.subdomainPathRewrite.fallbackPath=/default"
```

## URL Rewriting Examples

Based on the example configuration above, here's how different URLs would be rewritten:

| Original URL | Rewritten URL | Notes |
|-------------|---------------|--------|
| `customer1.example.com/users` | `example.com/api/customer1/users` | Subdomain becomes path segment |
| `customer2.example.com/` | `example.com/api/customer2/` | Minimal path case |
| `customer3.example.com/orders/123` | `example.com/api/customer3/orders/123` | Complex path preservation |
| `customer4.example.com/products/not-found` | `example.com/api/customer4/default` | Absolute fallback path (`/default`) |
| `customer5.example.com/catalog/missing` | `example.com/api/customer5/catalog/default` | Relative fallback path (`default`) |

## Headers

The plugin adds several headers to track the rewriting process:

- `X-Replaced-Path`: Original path before rewriting
- `X-Replaced-Host`: Original host before rewriting
- `X-Fallback-For`: Original URL when serving fallback content

These headers are useful for debugging and understanding how requests are being transformed by the plugin.
