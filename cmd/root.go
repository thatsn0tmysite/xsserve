package cmd

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"sync"

	"xsserve/core"
	"xsserve/database"
	"xsserve/server"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	// Used for flags.
	flags core.Flags

	rootCmd = &cobra.Command{
		Use:   "xsserve",
		Short: "A blind XSS discovery tool (inspired by xsshunter)",
		Long:  `This is a shameless copy of the xsshunter project, in a self-contained, on-demand format. No need to setup a whole server :).`,
		Run: func(cmd *cobra.Command, args []string) {
			//TODO: start those simultaneously and wait for them to clean exit (go, go)
			var wg sync.WaitGroup

			// Get database
			_, err := database.Open(flags.DatabaseURI)
			if err != nil {
				log.Fatal("Error opening database:", err)
			}
			defer database.Close()

			// Setup UI
			if flags.BasicAuth && flags.BasicAuthPass == "" {
				// Generate random password on UI start if Basic Auth is enabled
				p := make([]byte, 25)
				_, err := rand.Read(p)
				if err != nil {
					log.Fatal("Error generating UI credentials:", err)
				}
				flags.BasicAuthPass = hex.EncodeToString(p)
				log.Printf("[UI] Basic authentication: %v %v", flags.BasicAuthUser, flags.BasicAuthPass)
			}
			wg.Add(1)
			log.Printf("[UI] Listening on http://%v:%v", flags.UIAddress, flags.UIPort)
			go func() {
				defer wg.Done()
				server.ServeUI(&flags)
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
				server.ServeXSS(&flags)
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

	rootCmd.PersistentFlags().StringVar(&flags.DatabaseURI, "database-uri", "xsserve.db", "Database URI to use")
	rootCmd.PersistentFlags().StringVarP(&flags.Domain, "domain", "d", "", "Domain name to use")
	rootCmd.PersistentFlags().BoolVar(&flags.IsHTTPS, "https", false, "Serve XSS over HTTPS")
	rootCmd.PersistentFlags().StringVar(&flags.HTTPSCert, "https-cert", "", "Certificate path")
	rootCmd.PersistentFlags().StringVar(&flags.HTTPSKey, "https-key", "", "Key path")
	rootCmd.PersistentFlags().StringVar(&flags.UIAddress, "ui-addr", "127.0.0.1", "Address to host the UI on")
	rootCmd.PersistentFlags().IntVar(&flags.UIPort, "ui-port", 7331, "Port to bind for the UI to")
	rootCmd.PersistentFlags().StringVar(&flags.XSSAddress, "xss-addr", "0.0.0.0", "Address to serve the XSS files on ")
	rootCmd.PersistentFlags().IntVar(&flags.XSSPort, "xss-port", 0, "Port to bind for the XSS server to")
	rootCmd.PersistentFlags().IntVarP(&flags.Verbosity, "verbose", "v", 0, "Verbosity level")
	rootCmd.PersistentFlags().StringVar(&flags.ConfigFile, "config", "", "config file (default is $HOME/.cobra.yaml)")
	rootCmd.PersistentFlags().BoolVar(&flags.BasicAuth, "auth", false, "Enable basic authentication")
	rootCmd.PersistentFlags().StringVar(&flags.BasicAuthUser, "auth-user", "xsserve", "Basic-auth username")
	rootCmd.PersistentFlags().StringVar(&flags.BasicAuthPass, "auth-pass", "", "Basic-auth password")
	rootCmd.PersistentFlags().StringVar(&flags.SeleniumURL, "selenium-url", "http://127.0.0.1:4444/wd/hub", "Selenium node to use")
	rootCmd.PersistentFlags().StringVar(&flags.SeleniumBrowser, "selenium-browser", "firefox", "Selenium driver to use")

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
	viper.BindPFlag("BasicAuth", rootCmd.PersistentFlags().Lookup("auth"))
	viper.BindPFlag("BasicAuthUser", rootCmd.PersistentFlags().Lookup("auth-user"))
	viper.BindPFlag("BasicAuthPass", rootCmd.PersistentFlags().Lookup("auth-pass"))
	viper.BindPFlag("SeleniumURL", rootCmd.PersistentFlags().Lookup("selenium-url"))
	viper.BindPFlag("SeleniumBrowser", rootCmd.PersistentFlags().Lookup("selenium-browser"))

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
