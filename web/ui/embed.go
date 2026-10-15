package ui

import "embed"

// FS embeds index.html and all static assets for the Web UI.
//
//go:embed index.html static/*
var FS embed.FS
