/*
Copyright © 2026 Micromachine
*/
package main

import (
	"log/slog"

	"micromachine.dev/cmd-utils/cmd"
	"micromachine.dev/cmd-utils/lib/utils"
)

func main() {
	slog.SetDefault(slog.New(utils.NewColorHandler()))
	cmd.Execute()
}
