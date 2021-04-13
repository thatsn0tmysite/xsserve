package server

import "embed"

//go:embed resources/ui/*.tmpl
var FS embed.FS

//go:embed resources/ui/css resources/ui/js resources/ui/images resources/xss
var StaticFS embed.FS
