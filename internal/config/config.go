package config

import (
	"sync"

	"github.com/spf13/viper"
)

type Config struct {
	ServerPort                   string `mapstructure:"SERVER_PORT"`
	DatabaseURL                  string `mapstructure:"DATABASE_URL"`
	UserServiceURL               string `mapstructure:"USER_SERVICE_URL"`
	KafkaBroker                  string `mapstructure:"KAFKA_BROKER"`
	KafkaTopicPush               string `mapstructure:"KAFKA_TOPIC_PUSH"`
	MigrationURL                 string `mapstructure:"MIGRATION_URL"`
	RedisAddr                    string `mapstructure:"REDIS_ADDR"`
	JwtSecret                    string `mapstructure:"JWT_SECRET_KEY"`
	KafkaTopicPersistence        string `mapstructure:"KAFKA_TOPIC_PERSISTENCE"`
	KafkaTopicNotification       string `mapstructure:"KAFKA_TOPIC_NOTIFICATIONS"`
	KafkaDBWorkerConsumerGroupID string `mapstructure:"KAFKA_DB_WORKER_CONSUMER_GROUP_ID"`
}

var (
	instance *Config
	once     sync.Once
)

func LoadConfig() (*Config, error) {
	var err error
	once.Do(func() {
		viper.SetConfigFile(".env")
		viper.AutomaticEnv()

		if err = viper.ReadInConfig(); err != nil {
			return
		}

		instance = &Config{}
		err = viper.Unmarshal(instance)
	})
	return instance, err
}

func Get() *Config {
	if instance == nil {
		panic("Config has not been initialized. Call LoadConfig first.")
	}
	return instance
}
