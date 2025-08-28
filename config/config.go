package config

import (
	"log"
	"strings"

	"github.com/spf13/viper"
)

// Config holds all configuration for the application
type Config struct {
	Server   ServerConfig   `mapstructure:"server"`
	Redis    RedisConfig    `mapstructure:"redis"`
	Database DatabaseConfig `mapstructure:"database"`
	Logger   LoggerConfig   `mapstructure:"logger"`
}

// ServerConfig holds server-specific configuration
type ServerConfig struct {
	Host string `mapstructure:"host"`
	Env  string `mapstructure:"env"`
	Port string `mapstructure:"port"`
}

// RedisConfig holds redis-specific configuration
type RedisConfig struct {
	Address  string `mapstructure:"address"`
	Password string `mapstructure:"password"`
}

// DatabaseConfig holds database-specific configuration
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         string `mapstructure:"port"`
	User         string `mapstructure:"user"`
	Password     string `mapstructure:"password"`
	DBName       string `mapstructure:"dbname"`
	SSLMode      string `mapstructure:"sslmode"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

// LoggerConfig holds logger-specific configuration
type LoggerConfig struct {
	Level  string `mapstructure:"level"`
	Format string `mapstructure:"format"`
}

// Load reads configuration from environment variables and files
func Load() (*Config, error) {
	cfg := &Config{}

	// Set default values
	viper.SetDefault("server.host", "localhost")
	viper.SetDefault("server.env", "local")
	viper.SetDefault("server.port", "8080")
	viper.SetDefault("database.host", "localhost")
	viper.SetDefault("database.port", "5432")
	viper.SetDefault("database.user", "postgres")
	viper.SetDefault("database.password", "password")
	viper.SetDefault("database.dbname", "explore")
	viper.SetDefault("database.sslmode", "disable")
	viper.SetDefault("database.max_open_conns", 25)
	viper.SetDefault("database.max_idle_conns", 10)
	viper.SetDefault("redis.address", "localhost:6379")
	viper.SetDefault("redis.password", "")
	viper.SetDefault("logger.level", "info")
	viper.SetDefault("logger.format", "json")

	// Read from environment variables
	viper.AutomaticEnv()
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Config file not found, using defaults and environment variables: %v", err)
	}

	// Override with environment variables if set
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	_ = viper.BindEnv("server.host")             // SERVER_HOST
	_ = viper.BindEnv("server.port")             // SERVER_PORT
	_ = viper.BindEnv("database.host")           // DATABASE_HOST
	_ = viper.BindEnv("database.port")           // DATABASE_PORT
	_ = viper.BindEnv("database.user")           // DATABASE_USER
	_ = viper.BindEnv("database.password")       // DATABASE_PASSWORD
	_ = viper.BindEnv("database.dbname")         // DATABASE_DBNAME
	_ = viper.BindEnv("database.sslmode")        // DATABASE_SSLMODE
	_ = viper.BindEnv("database.max_open_conns") // DATABASE_MAX_OPEN_CONNS
	_ = viper.BindEnv("database.max_idle_conns") // DATABASE_MAX_IDLE_CONNS
	_ = viper.BindEnv("logger.level")            // LOGGER_LEVEL
	_ = viper.BindEnv("logger.format")           // LOGGER_FORMAT
	_ = viper.BindEnv("redis.address")           // REDIS_ADDRESS
	_ = viper.BindEnv("redis.password")          // REDIS_PASSWORD

	if err := viper.Unmarshal(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
