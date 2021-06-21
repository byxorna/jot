package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/user"
	"time"

	"github.com/byxorna/jot/pkg/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	flags = struct {
		ConfigFile string
	}{}

	root = &cobra.Command{
		Use:   "jot",
		Short: "Jot is a terminal based organizational tool",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error
			var fileName string

			if len(args) > 0 {
				fileName = args[0]
			}

			user, err := user.Current()
			if err != nil {
				return errors.New("could not get current user")
			}

			m, err := model.NewFromConfigFile(flags.ConfigFile, user.Name)
			if err != nil {
				return err
			}

			p := tea.NewProgram(m, tea.WithAltScreen())
			err = p.Start()
			return err
		},
	}
)

func init() {
	root.PersistentFlags().StringVarP(&flags.ConfigFile, "config", "c", "~/.jot.yaml", "configuration file")
}

func Execute() {
	err := root.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
