package server

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
)

type ConfigStruct struct {
	Log struct {
		Level         string
		PrintRequests bool
	}
	TLS struct {
		CertificateFile string
		KeyFile         string
	}
	OIDC struct {
		Issuer                       string
		ClientID                     string
		ClientSecret                 string
		RedirectURL                  string
		SessionTTLSeconds            int
		AllowedConfidentialClientIDs []string `yaml:"AllowedConfidentialClientIDs,flow"`
	}
	Web struct {
		StaticContentDir string
		TemplatesDir     string
		SessionsDir      string
	}
	Datastore struct {
		DataSourceName string
	}
	BindAddress string
}

func LoadConfiguration() ConfigStruct {
	var config ConfigStruct
	var needHelp bool
	var configFileName string

	pflag.StringVar(&configFileName, "config", "", "Configuration file name (parameters in configuration file have priority over command line args)")
	pflag.BoolVar(&needHelp, "help", false, "Show available configuration options")

	pflag.StringVar(&config.Log.Level, "log-level", "info", "Logging level")
	pflag.BoolVar(&config.Log.PrintRequests, "log-requests", false, "Log all HTTP requests (default false)")

	pflag.StringVar(&config.TLS.CertificateFile, "tls-cert-file", "", "File name of TLS Certificate (mandatory)")
	pflag.StringVar(&config.TLS.KeyFile, "tls-key-file", "", "File name of TLS private key (mandatory)")

	pflag.StringVar(&config.OIDC.Issuer, "oidc-issuer", "", "OIDC authorization server URL (mandatory)")
	pflag.StringVar(&config.OIDC.ClientID, "oidc-client-id", "", "OIDC client ID (mandatory)")
	pflag.StringVar(&config.OIDC.ClientSecret, "oidc-client-secret", "", "OIDC client secret (mandatory)")
	pflag.StringVar(&config.OIDC.RedirectURL, "oidc-redirect-url", "", "OIDC redirect URL (mandatory)")
	pflag.IntVar(&config.OIDC.SessionTTLSeconds, "oidc-session-ttl", 3600, "Duration of session, in seconds")
	pflag.StringArrayVar(&config.OIDC.AllowedConfidentialClientIDs, "oidc-allowed-client-id", nil, "Allowed confidential client (mandatory)")

	pflag.StringVar(&config.Web.StaticContentDir, "web-static", "", "Path to static web content for web frontend (mandatory)")
	pflag.StringVar(&config.Web.TemplatesDir, "web-templates", "", "Path to HTML template files (mandatory)")
	pflag.StringVar(&config.Web.SessionsDir, "web-sessions-dir", "/srv/bastion/sessions", "Path where to store user web sessions")

	pflag.StringVar(&config.Datastore.DataSourceName, "datastore-dsn", "", "Data Source Name to connect to (mandatory)")

	pflag.StringVar(&config.BindAddress, "bind-address", "0.0.0.0:1443", "The IP address and port on which to listen for HTTPS requests")

	pflag.Parse()
	err := viper.BindPFlags(pflag.CommandLine)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}

	if needHelp {
		pflag.Usage()
		os.Exit(0)
	}
	if configFileName != "" {
		viper.SetConfigName(configFileName)
		viper.SetConfigType("yaml")
		viper.AddConfigPath(".")
		fmt.Printf("Reading configuration from %s... ", configFileName)
		err := viper.ReadInConfig()
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		err = viper.Unmarshal(&config)
		if err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
		fmt.Println("success")
	} else {
		fmt.Println("No configuration file name provided. Will use command line arguments")
		if config.TLS.CertificateFile == "" ||
			config.TLS.KeyFile == "" ||
			config.OIDC.Issuer == "" ||
			config.OIDC.ClientID == "" ||
			config.OIDC.ClientSecret == "" ||
			config.OIDC.RedirectURL == "" ||
			config.OIDC.AllowedConfidentialClientIDs == nil ||
			config.Web.StaticContentDir == "" ||
			config.Web.TemplatesDir == "" ||
			config.Datastore.DataSourceName == "" {
			fmt.Println("Missing mandatory argument(s)")
			pflag.Usage()
			os.Exit(1)
		}
	}

	return config
}
