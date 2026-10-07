// Command taskd is the terminal view for taskd.
package main

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"

	"github.com/Qaid-Danial/taskd/internal/store"
	"github.com/Qaid-Danial/taskd/internal/tui"
)

func main() {
	cfg, err := store.ConfigFromEnv()
	if err != nil {
		fmt.Fprintln(os.Stderr, "taskd:", err)
		os.Exit(1)
	}

	if _, err := tea.NewProgram(tui.New(store.New(cfg))).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "taskd:", err)
		os.Exit(1)
	}
}