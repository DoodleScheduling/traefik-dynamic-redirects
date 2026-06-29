# Traefik Dynamic Redirects

[![release](https://img.shields.io/github/release/DoodleScheduling/traefik-dynamic-redirects/all.svg)](https://github.com/DoodleScheduling/traefik-dynamic-redirects/releases)
[![report](https://goreportcard.com/badge/github.com/DoodleScheduling/traefik-dynamic-redirects)](https://goreportcard.com/report/github.com/DoodleScheduling/traefik-dynamic-redirects)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/traefik-dynamic-redirects/badge)](https://api.securityscorecards.dev/projects/github.com/DoodleScheduling/traefik-dynamic-redirects)
[![Coverage Status](https://coveralls.io/repos/github/DoodleScheduling/traefik-dynamic-redirects/badge.svg?branch=master)](https://coveralls.io/github/DoodleScheduling/traefik-dynamic-redirects?branch=master)
[![license](https://img.shields.io/github/license/DoodleScheduling/traefik-dynamic-redirects.svg)](https://github.com/DoodleScheduling/traefik-dynamic-redirects/blob/master/LICENSE)

A Traefik middleware plugin for dynamic redirects based on cookies and the `Accept-Language` header.

This plugin allows defining multiple dynamic redirect rules in a single Traefik Middleware configuration. It is useful for root-page routing such as:

- redirecting authenticated users to an application/home page;
- redirecting anonymous users based on a locale cookie;
- falling back to the browser `Accept-Language` header;
- falling back to a default target URL.

## Redirect behavior

For each matching `sourceURL`, the plugin resolves the redirect target in this order:

1. `authenticatedCookie` + `authenticatedTarget`
2. `localeCookie` + `localeTargets`
3. `Accept-Language` + `localeTargets`
4. `defaultTarget`

If none of the configured redirects match the request URL, the request is passed to the next handler.

## Redirect fields

| Key | Description |
| --- | --- |
| `sourceURL` | Absolute source URL to match. Query strings and fragments are not supported in `sourceURL`. |
| `statusCode` | Redirect status code: `301`, `302`, `303`, `307`, `308`. Defaults to `302` when omitted. |
| `preserveQueryString` | Boolean. When `true`, appends the original request query string to the selected target URL. |
| `authenticatedCookie` | Cookie name used to detect authenticated users. |
| `authenticatedTarget` | Absolute target URL used when `authenticatedCookie` exists and has a non-empty value. |
| `localeCookie` | Cookie name used to select a localized redirect target. |
| `defaultTarget` | Absolute fallback target URL. |
| `localeTargets` | Map of locale code to absolute target URL, for example `en`, `de`, `fr`, `it`, `es`. |

## Configuration

Example Middleware:

```yaml
apiVersion: traefik.io/v1alpha1
kind: Middleware
metadata:
  name: dynamic-redirects
spec:
  plugin:
    dynamicRedirects:
      redirects:
        - sourceURL: https://example.com/
          statusCode: 302
          preserveQueryString: true
          authenticatedCookie: Authentication
          authenticatedTarget: https://example.com/home/
          localeCookie: locale
          defaultTarget: https://example.com/en/
          localeTargets:
            de: https://example.com/de/
            es: https://example.com/es/
            fr: https://example.com/fr/
            it: https://example.com/it/

        - sourceURL: https://www.example.com/
          statusCode: 302
          preserveQueryString: true
          authenticatedCookie: Authentication
          authenticatedTarget: https://example.com/home/
          localeCookie: locale
          defaultTarget: https://example.com/en/
          localeTargets:
            de: https://example.com/de/
            es: https://example.com/es/
            fr: https://example.com/fr/
            it: https://example.com/it/
```

## Static configuration

Enable the plugin in Traefik static configuration:

```yaml
experimental:
  plugins:
    dynamicRedirects:
      moduleName: github.com/DoodleScheduling/traefik-dynamic-redirects
      version: v0.0.1
```

## Example results

Authenticated user:

```http
GET https://example.com/
Cookie: Authentication=abc
```

Redirects to:

```text
https://example.com/home/
```

Locale cookie:

```http
GET https://example.com/
Cookie: locale=es
```

Redirects to:

```text
https://example.com/es/
```

Accept-Language fallback:

```http
GET https://example.com/
Accept-Language: fr-FR,fr;q=0.9,en;q=0.8
```

Redirects to:

```text
https://example.com/fr/
```

Default fallback:

```http
GET https://example.com/
```

Redirects to:

```text
https://example.com/en/
```

## Plugin catalog metadata

The `.traefik.yml` metadata should match the plugin module name:

```yaml
displayName: Dynamic Redirects
iconPath: .assets/icon.png
import: github.com/DoodleScheduling/traefik-dynamic-redirects
summary: Middleware for dynamic redirects based on cookies and Accept-Language.
type: middleware
testData:
  redirects:
    - sourceURL: https://example.com/
      statusCode: 302
      preserveQueryString: true
      authenticatedCookie: Authentication
      authenticatedTarget: https://example.com/home/
      localeCookie: locale
      defaultTarget: https://example.com/en/
      localeTargets:
        de: https://example.com/de/
        es: https://example.com/es/
        fr: https://example.com/fr/
        it: https://example.com/it/
    - sourceURL: https://www.example.com/
      statusCode: 302
      preserveQueryString: true
      authenticatedCookie: Authentication
      authenticatedTarget: https://example.com/home/
      localeCookie: locale
      defaultTarget: https://example.com/en/
      localeTargets:
        de: https://example.com/de/
        es: https://example.com/es/
        fr: https://example.com/fr/
        it: https://example.com/it/
```
