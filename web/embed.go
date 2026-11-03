// Package web embeds the templates and static files served by the application.
// It is one of only two Go files outside cmd/ and internal/, because Go's embed
// feature can include only files at or below the embedding file's directory
// (Web-App Development Stack v2.0, Section 6).
package web

import "embed"

// Files holds web/templates and web/static.
//
//go:embed templates static
var Files embed.FS
