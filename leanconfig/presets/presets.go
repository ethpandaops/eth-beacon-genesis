package presets

import (
	"embed"
)

// preset configs
//
//go:embed *.yaml
var PresetsFS embed.FS
