package main

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	tu "github.com/panyam/templar/utils"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start an HTTP server to serve templates",
	Long: `Start an HTTP server that serves Templar templates.

Config file options (serve section):
  serve:
    addr: ":8080"
    templates:
      - ./templates
      - ../shared/templates
    static:
      - /css:./styles
      - /js:./scripts

Examples:
  templar serve -t templates -s /static:./public
  templar serve --addr :8080 -t templates -t ../shared/templates
  templar serve -t templates -s /css:./styles -s /js:./scripts`,
	Run: func(cmd *cobra.Command, args []string) {
		addr := viper.GetString("serve.addr")
		templateDirs := viper.GetStringSlice("serve.templates")
		staticDirs := viper.GetStringSlice("serve.static")

		b := tu.BasicServer{
			TemplateDirs: templateDirs,
			StaticDirs:   staticDirs,
		}
		b.Serve(nil, addr)
	},
}

func init() {
	serveCmd.Flags().StringP("addr", "a", ":7777", "Address where the HTTP server will run")
	serveCmd.Flags().StringArrayP("template", "t", nil, "Template directories to load templates from (can be repeated)")
	serveCmd.Flags().StringArrayP("static", "s", nil, "Static directories in format <http_prefix>:<local_folder> (can be repeated)")

	// Bind flags to viper
	viper.BindPFlag("serve.addr", serveCmd.Flags().Lookup("addr"))
	viper.BindPFlag("serve.templates", serveCmd.Flags().Lookup("template"))
	viper.BindPFlag("serve.static", serveCmd.Flags().Lookup("static"))

	// Set defaults
	viper.SetDefault("serve.addr", ":7777")
}
