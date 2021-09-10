package cmd

import (
	"context"
	"fmt"
	"os"
	"os/user"

	"github.com/byxorna/jot/pkg/app"
	//"github.com/byxorna/jot/pkg/model"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
)

var (
	flags = struct {
		ConfigFile   string
		UseAltScreen bool
	}{}

	root = &cobra.Command{
		Use:   "jot",
		Short: "Jot is a terminal based organizational tool",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(cmd *cobra.Command, args []string) error {
			var err error

			user, err := user.Current()
			if err != nil {
				return fmt.Errorf("could not get current user: %w", err)
			}

			a, err := app.New(context.TODO(), flags.ConfigFile, user.Name, flags.UseAltScreen)
			//m, err := model.NewFromConfigFile(context.TODO(), flags.ConfigFile, user.Name, flags.UseAltScreen)
			if err != nil {
				return fmt.Errorf("unable to create program: %w", err)
			}

			if !a.UseAltScreen {
				return tea.NewProgram(a).Start()
			}
			return tea.NewProgram(a, tea.WithAltScreen()).Start()
		},
	}
)

func init() {
	root.PersistentFlags().StringVarP(&flags.ConfigFile, "config", "c", "~/.jot.yaml", "configuration file")
	root.PersistentFlags().BoolVar(&flags.UseAltScreen, "use-alt-screen", true, "use terminal alternate screen buffer")
}

func Execute() {
	err := root.Execute()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
