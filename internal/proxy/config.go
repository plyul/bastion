package proxy

import (
	"fmt"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"os"
)

type ConfigStruct struct {
	Log struct {
		Level string `yaml:"level"`
	}
	API struct {
		URL             string `yaml:"url"`
		CertificateFile string `yaml:"certificateFile"`
	}
	OIDC struct {
		Issuer       string `yaml:"issuer"`
		ClientID     string `yaml:"clientID"`
		ClientSecret string `yaml:"clientSecret"`
	}
	BindAddress       string `yaml:"bindAddress"`
	GuardedNetwork    string `yaml:"guardedNetwork"`
	ConnectTimeoutSec int    `yaml:"connectTimeout"`
}

func LoadConfiguration() ConfigStruct {
	var config ConfigStruct
	var needHelp bool
	var configFileName string

	pflag.StringVar(&configFileName, "config", "", "Configuration file name (parameters in configuration file have priority over command line args)")
	pflag.BoolVar(&needHelp, "help", false, "Show available configuration options")

	pflag.StringVar(&config.Log.Level, "log-level", "info", "Logging level")

	pflag.StringVar(&config.API.URL, "api-url", "", "Bastion server URL (mandatory)")
	pflag.StringVar(&config.API.CertificateFile, "api-cert", "", "Certificate file name used for connection to Bastion server (mandatory)")

	pflag.StringVar(&config.OIDC.Issuer, "oidc-issuer", "", "OIDC authorization server URL  (mandatory)")
	pflag.StringVar(&config.OIDC.ClientID, "oidc-client-id", "", "OIDC client ID  (mandatory)")
	pflag.StringVar(&config.OIDC.ClientSecret, "oidc-client-secret", "", "OIDC client secret  (mandatory)")

	pflag.StringVar(&config.BindAddress, "bind-address", "0.0.0.0:2200", "The IP address and port on which to listen for HTTPS requests")
	pflag.StringVar(&config.GuardedNetwork, "network", "", "Network this proxy serves (mandatory)")
	pflag.IntVar(&config.ConnectTimeoutSec, "connect-timeout", 5, "Timeout connecting to target hosts, seconds")

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
		if config.API.URL == "" ||
			config.API.CertificateFile == "" ||
			config.OIDC.Issuer == "" ||
			config.OIDC.ClientID == "" ||
			config.OIDC.ClientSecret == "" ||
			config.GuardedNetwork == "" {
			fmt.Println("Missing mandatory argument(s)")
			pflag.Usage()
			os.Exit(1)
		}
	}

	return config
}
