/*
Copyright © 2023 NAME HERE <EMAIL ADDRESS>
*/
package cmd

import (
	"log"
	"revolution/component"

	"github.com/spf13/cobra"
)

// modifierCmd represents the modifier command
var modifierCmd = &cobra.Command{
	Use:   "modifier",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {

		if err := component.CreateComponent(args[0], "modifier"); err != nil {
			log.Fatalln(err)
		}
	},
}

func init() {
	createCmd.AddCommand(modifierCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// modifierCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	// modifierCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
