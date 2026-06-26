# Traefik Plugin Bulk Redirects

[![release](https://img.shields.io/github/release/DoodleScheduling/traefik-dynamic-redirects/all.svg)](https://github.com/DoodleScheduling/traefik-dynamic-redirects/releases)
[![report](https://goreportcard.com/badge/github.com/DoodleScheduling/traefik-dynamic-redirects)](https://goreportcard.com/report/github.com/DoodleScheduling/traefik-dynamic-redirects)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/traefik-dynamic-redirects/badge)](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/traefik-dynamic-redirects)
[![Coverage Status](https://coveralls.io/repos/github/DoodleScheduling/traefik-dynamic-redirects/badge.svg?branch=master)](https://coveralls.io/github/DoodleScheduling/traefik-dynamic-redirects?branch=master)
[![license](https://img.shields.io/github/license/DoodleScheduling/traefik-dynamic-redirects.svg)](https://github.com/DoodleScheduling/traefik-dynamic-redirects/blob/master/LICENSE)

A Traefik middleware plugin for Cloudflare-style bulk redirects. 
It allows defining multiple redirects in a single Traefik Middleware configuration.
This plugin supports exact redirects, subpath redirects, query string preservation, and configurable redirect status codes.

# Redirect fields

| Key | Description |
| --- | --- |
| `sourceURL` | absolute source URL to match |
| `targetURL` | absolute redirect destination URL |
| `statusCode` | redirect status code: `301`, `302`, `303`, `307`, `308` |
| `preserveQueryString` | `enabled` appends the original query string to the target URL |
| `subpathMatching` | `enabled` matches the source path and all child paths below it |

# Configuration

Example usage

```yaml
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: bulk-redirects
spec:
  plugin:
    bulkRedirects:
      redirects:
      - sourceURL: https://example.com/premium/coupon
        targetURL: https://example.com/en/premium/
        statusCode: 302
        preserveQueryString: enabled
        subpathMatching: disabled
      - sourceURL: https://example.com/docs
        targetURL: https://example.com/en/resources
        statusCode: 301
        preserveQueryString: enabled
        subpathMatching: enabled
```

# Static configuration

```yaml
experimental:
  plugins:
    bulkRedirects:
      moduleName: github.com/doodlescheduling/traefik-dynamic-redirects
      version: v0.1.0
```

