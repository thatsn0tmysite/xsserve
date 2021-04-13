package cmd

import (
	"fmt"
	"log"
	"sync"

	"xsserve/database"
	"xsserve/server"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Command line flags
type Flags struct {
	DatabaseURI string // MongoDB database URI
	Database    string // Database name
	Domain      string // Domain name to use
	IsHTTPS     bool   // Serve XSS over HTTPS?
	HTTPSCert   string // Certificate path
	HTTPSKey    string // Key path
	UIAddress   string // Address to host the UI on (defaults to 127.0.0.1)
	UIPort      int    // Port to bind for the UI to (defaults to 7331)
	XSSAddress  string // Address to serve the XSS files on (defaults to 0.0.0.0)
	XSSPort     int    // Port to bind for the XSS server to (defaults to 8443 if IsHTTPS, 8080 otherwise)
	ConfigFile  string // Viper configuration file to use
	Verbosity   int    // Verbosity level (defaults to 0, >4 is debug)
}

var (
	// Used for flags.
	flags Flags

	rootCmd = &cobra.Command{
		Use:   "xsserve",
		Short: "A blind XSS discovery tool (inspired by xsshunter)",
		Long:  `This is a shameless copy of the xsshunter project, in a self-contained, on-demand format. No need to setup a whole server :).`,
		Run: func(cmd *cobra.Command, args []string) {
			//TODO: start those simultaneously and wait for them to clean exit (go, go)
			var wg sync.WaitGroup

			// Get database
			err := database.Open(flags.DatabaseURI, flags.Database)
			if err != nil {
				log.Fatal("Error opening databse:", err)
			}
			log.Println("Successfully connected to database:", flags.DatabaseURI, flags.Database)
			// Setup UI
			wg.Add(1)
			log.Printf("[UI] Listening on http://%v:%v", flags.UIAddress, flags.UIPort)
			go func() {
				defer wg.Done()
				server.ServeUI(flags.UIAddress, flags.UIPort)
			}()

			// Setup HTTPS
			scheme := "http"
			if flags.IsHTTPS {
				scheme = "https"
				if flags.XSSPort == 0 {
					flags.XSSPort = 8443
				}
			}
			if flags.XSSPort == 0 {
				flags.XSSPort = 8080
			}
			wg.Add(1)
			log.Printf("[XSS] Listening on %v://%v:%v", scheme, flags.XSSAddress, flags.XSSPort)
			go func() {
				defer wg.Done()
				server.ServeXSS(flags.XSSAddress, flags.XSSPort, flags.IsHTTPS, flags.HTTPSCert, flags.HTTPSKey)
			}()
			wg.Wait()
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringVar(&flags.DatabaseURI, "database-uri", "mongodb://127.0.0.1:27017", "MongoDB database URI")
	rootCmd.PersistentFlags().StringVar(&flags.Database, "database", "xsserve_db", "MongoDB database name")
	rootCmd.PersistentFlags().StringVarP(&flags.Domain, "domain", "d", "127.0.0.1", "Domain name to use")
	rootCmd.PersistentFlags().BoolVar(&flags.IsHTTPS, "https", false, "Serve XSS over HTTPS")
	rootCmd.PersistentFlags().StringVar(&flags.HTTPSCert, "https-cert", "", "Certificate path")
	rootCmd.PersistentFlags().StringVar(&flags.HTTPSKey, "https-key", "", "Key path")
	rootCmd.PersistentFlags().StringVar(&flags.UIAddress, "ui-addr", "127.0.0.1", "Address to host the UI on")
	rootCmd.PersistentFlags().IntVar(&flags.UIPort, "ui-port", 7331, "Port to bind for the UI to")
	rootCmd.PersistentFlags().StringVar(&flags.XSSAddress, "xss-addr", "0.0.0.0", "Address to serve the XSS files on ")
	rootCmd.PersistentFlags().IntVar(&flags.XSSPort, "xss-port", 0, "Port to bind for the XSS server to")
	rootCmd.PersistentFlags().IntVarP(&flags.Verbosity, "verbose", "v", 0, "Verbosity level")
	rootCmd.PersistentFlags().StringVar(&flags.ConfigFile, "config", "", "config file (default is $HOME/.cobra.yaml)")

	//	viper.SetDefault("author", "NAME HERE <EMAIL ADDRESS>")
	//	viper.SetDefault("license", "apache")

	viper.BindPFlag("DatabaseURI", rootCmd.PersistentFlags().Lookup("database-uri"))
	viper.BindPFlag("Database", rootCmd.PersistentFlags().Lookup("database"))
	viper.BindPFlag("Domain", rootCmd.PersistentFlags().Lookup("domain"))
	viper.BindPFlag("IsHTTPS", rootCmd.PersistentFlags().Lookup("https"))
	viper.BindPFlag("HTTPSCertificate", rootCmd.PersistentFlags().Lookup("https-cert"))
	viper.BindPFlag("HTTPSKey", rootCmd.PersistentFlags().Lookup("https-key"))
	viper.BindPFlag("UIAddress", rootCmd.PersistentFlags().Lookup("ui-addr"))
	viper.BindPFlag("UIPort", rootCmd.PersistentFlags().Lookup("ui-port"))
	viper.BindPFlag("XSSAddress", rootCmd.PersistentFlags().Lookup("xss-addr"))
	viper.BindPFlag("XSSPort", rootCmd.PersistentFlags().Lookup("xss-port"))
	viper.BindPFlag("Verbosity", rootCmd.PersistentFlags().Lookup("verbose"))

	// rootCmd.AddCommand(addCmd)
	// rootCmd.AddCommand(initCmd)
}

func initConfig() {
	if flags.ConfigFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(flags.ConfigFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".cobra" (without extension).
		viper.AddConfigPath(fmt.Sprintf("%v/.config/%v", home, "xsserve"))
		viper.SetConfigName("config.json")
	}

	viper.AutomaticEnv()

	if err := viper.ReadInConfig(); err == nil {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
	}
}
