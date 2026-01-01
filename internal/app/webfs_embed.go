package app

import "embed"

//go:embed all:web/dist
var webFSEmbed embed.FS

func init() {
	WebFS = webFSEmbed
}
