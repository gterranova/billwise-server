package util

import (
	"github.com/spf13/viper"
)

// Config stores all configuration of the application.
// The values are read by viper from a config file or environment variable.

// Initilize this variable to access the env values
var Config *config

// We will call this in main.go to load the env variables
func InitEnvConfigs() {
	if c, err := LoadConfig("."); err == nil {
		Config = &c
	}
}

type config struct {
	ApiServerAddr string `mapstructure:"API_SERVER_ADDR"`
	ApiSecret     string `mapstructure:"API_SECRET"`
	DBHost        string `mapstructure:"DB_HOST"`
	DBDriver      string `mapstructure:"DB_DRIVER"`
	DBUser        string `mapstructure:"DB_USER"`
	DBPassword    string `mapstructure:"DB_PASSWORD"`
	DBName        string `mapstructure:"DB_NAME"`
	DBPort        string `mapstructure:"DB_PORT"`

	DefaultAdminName     string `mapstructure:"DEFAULT_ADMIN_NAME"`
	DefaultAdminLastname string `mapstructure:"DEFAULT_ADMIN_LASTNAME"`
	DefaultAdminEmail    string `mapstructure:"DEFAULT_ADMIN_EMAIL"`
	DefaultUserDomain    string `mapstructure:"DEFAULT_USER_DOMAIN"`
	DefaultUserPassword  string `mapstructure:"DEFAULT_USER_PASSWORD"`

	TestDBHost     string `mapstructure:"TEST_DB_HOST"`
	TestDBDriver   string `mapstructure:"TEST_DB_DRIVER"`
	TestDBUser     string `mapstructure:"TEST_DB_USER"`
	TestDBPassword string `mapstructure:"TEST_DB_PASSWORD"`
	TestDBName     string `mapstructure:"TEST_DB_NAME"`
	TestDBPort     string `mapstructure:"TEST_DB_PORT"`
}

// LoadConfig reads configuration from file or environment variables.
func LoadConfig(path string) (config config, err error) {
	viper.AddConfigPath(path)
	viper.SetConfigName(".env")
	viper.SetConfigType("env")

	viper.AutomaticEnv()

	err = viper.ReadInConfig()
	if err != nil {
		return
	}

	err = viper.Unmarshal(&config)
	return
}
