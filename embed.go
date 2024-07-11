package flightlesssomething

import "embed"

//go:embed templates/*
var templatesFS embed.FS

//go:embed static/*
var staticFS embed.FS
