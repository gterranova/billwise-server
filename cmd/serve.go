/*
Copyright © 2023 Gianpaolo Terranova <gianpaoloterranova@gmail.com>
*/
package cmd

import (
	"it.terra9/billwise-server/database"
	"it.terra9/billwise-server/routes"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func startServer(serverAddr string) {

	database.Connect()

	app := fiber.New()

	// Add CORS Middleware so the frontend get the cookie
	app.Use(cors.New(cors.Config{
		AllowCredentials: true,
	}))

	routes.Setup(app)

	app.Listen(serverAddr)
}

// serveCmd represents the serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "A brief description of your command",
	Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	Run: func(cmd *cobra.Command, args []string) {
		startServer(viper.GetString("API_SERVER_ADDR"))
	},
}

func init() {
	RootCmd.AddCommand(serveCmd)

	// Here you will define your flags and configuration settings.

	// Cobra supports Persistent Flags which will work for this command
	// and all subcommands, e.g.:
	// serveCmd.PersistentFlags().String("foo", "", "A help for foo")

	// Cobra supports local flags which will only run when this command
	// is called directly, e.g.:
	serveCmd.Flags().Int("port", 8000, "Port to run Application server on")
	viper.BindPFlag("port", serveCmd.Flags().Lookup("port"))
}
