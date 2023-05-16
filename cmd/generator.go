/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"

	fm "revolution/filemanage"
)

// generatorCmd represents the generator command
var generatorCmd = &cobra.Command{
	Use:   "generator [title]",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		if err := fm.CreateGenerator(args[0]); err != nil {
			return err
		}

		return nil
	},
}

func init() {
	createCmd.AddCommand(generatorCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// generatorCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// generatorCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
