/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	fm "revolution/filemanage"
)

var template string

// projectCmd represents the project command
var projectCmd = &cobra.Command{
	Use:   "project [title]",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {

		if template == "" {
			template = viper.GetString("default_project_template")
		}

		return fm.CreateProject(args[0], template)
	},
}

func init() {
	createCmd.AddCommand(projectCmd)

	projectCmd.PersistentFlags().StringVarP(&template, "template", "t", "default", "specify a template for the project")

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// projectCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// projectCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
