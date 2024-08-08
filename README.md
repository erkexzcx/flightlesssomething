# FlightlessSomething

[flightlessmango.com](https://flightlessmango.com/) website clone, written in Go.

Yes, there is a lot of crappy copypasta html/css/js code. As long as it works! 🤷

# Features

* Written in Go:
  * Fast performance
  * Multithreaded
  * Single, statically linked binary
* Uses `gin` web framework
* Uses `gorm` ORM (Can be easily ported to other databases)

## Features that will NOT be included

* TLS/SSL/ACME - use reverse proxy (I suggest [Caddy](https://github.com/caddyserver/caddy))

# Development

To run this code locally, setup `go`, open this project and run this:

```bash
go run cmd/flightlesssomething/main.go -data-dir data -discord-client-id xxxxxxxxxxxxxxxxxxx -discord-client-secret xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx -discord-redirect-url 'http://127.0.0.1:8080/login/callback' -session-secret xxxxxxxxxxxxxxxxxxxxxxxx -openai-api-key xxxxxxxxxxxxxxxxxxxxxxxx
```

Then open in browser: http://127.0.0.1:8080/
