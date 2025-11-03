package assets

import (
	"embed"
)

//go:embed *.png
var EmbeddedAssets embed.FS
