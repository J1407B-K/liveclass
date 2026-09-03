// Package skills embeds the built-in agent skill definitions so runtime
// behavior does not depend on the process working directory.
package skills

import "embed"

// Files contains every built-in skill definition.
//
//go:embed *.md
var Files embed.FS
