package abacus

import "embed"

//go:embed all:web/dist
var Frontend embed.FS

//go:embed migrations
var Migrations embed.FS
