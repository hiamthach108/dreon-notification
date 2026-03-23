package config

import (
	"os"
	"reflect"
	"strconv"

	"github.com/golobby/dotenv"
)

type AppConfig struct {
	App struct {
		Name    string `env:"APP_NAME"`
		Version string `env:"APP_VERSION"`
	}
	Server struct {
		Host     string `env:"HTTP_HOST"`
		Port     string `env:"HTTP_PORT"`
		GRPCPort string `env:"GRPC_PORT"`
	}
	Logger struct {
		Level string `env:"LOG_LEVEL"`
	}

	Cache struct {
		DefaultExpireTimeSec int    `env:"CACHE_DEFAULT_EXPIRE_TIME_SEC"`
		CleanupIntervalHour  int    `env:"CACHE_CLEANUP_INTERVAL_HOUR"`
		RedisHost            string `env:"REDIS_HOST"`
		RedisPort            string `env:"REDIS_PORT"`
		RedisPassword        string `env:"REDIS_PASSWORD"`
		RedisDB              int    `env:"REDIS_DB"`
	}

	Postgres struct {
		ConnectionName string `env:"POSTGRES_CONNECTION_NAME"`
		Host           string `env:"POSTGRES_HOST"`
		Port           int    `env:"POSTGRES_PORT"`
		Username       string `env:"POSTGRES_USERNAME"`
		Password       string `env:"POSTGRES_PASSWORD"`
		DBName         string `env:"POSTGRES_DBNAME"`
		SSL            bool   `env:"POSTGRES_SSL"`
		MaxIdleConns   int    `env:"POSTGRES_MAX_IDLE_CONNS"`
		MaxOpenConns   int    `env:"POSTGRES_MAX_OPEN_CONNS"`
	}

	Notification struct {
		MaxAttempts              int `env:"NOTIFICATION_MAX_ATTEMPTS" default:"3"`
		RetryIntervalSec         int `env:"NOTIFICATION_RETRY_INTERVAL_SEC"`
		RetryBatchSize           int `env:"NOTIFICATION_RETRY_BATCH_SIZE"`
		RetryBackoffInitialSec   int `env:"NOTIFICATION_RETRY_BACKOFF_INITIAL_SEC"`
		RetryBackoffMaxSec       int `env:"NOTIFICATION_RETRY_BACKOFF_MAX_SEC"`
		RetryPublishLeaseSec     int `env:"NOTIFICATION_RETRY_PUBLISH_LEASE_SEC"`
	}

	Email struct {
		Sender       string `env:"EMAIL_SENDER"`
		TemplateDir  string `env:"EMAIL_TEMPLATE_DIR" default:"templates/email"`
		ResendAPIKey string `env:"EMAIL_RESEND_API_KEY"`
	}

	SMS struct {
		TwilioAccountSID string `env:"TWILIO_ACCOUNT_SID"`
		TwilioAuthToken  string `env:"TWILIO_AUTH_TOKEN"`
		TwilioFromNumber string `env:"TWILIO_FROM_NUMBER"` // E.164, e.g. +15551234567
		SMSTemplateDir   string `env:"SMS_TEMPLATE_DIR" default:"templates/sms"`
	}

	RabbitMQ struct {
		URL string `env:"RABBITMQ_URL"`
	}
}

func NewAppConfig() (*AppConfig, error) {
	config := &AppConfig{}

	// Try to load from .env file first (optional)
	file, err := os.Open(".env")
	if err == nil {
		defer func() {
			if closeErr := file.Close(); closeErr != nil {
				// Log error but don't fail if file close fails
				_ = closeErr
			}
		}()
		err = dotenv.NewDecoder(file).Decode(config)
		if err != nil {
			return config, err
		}
	}

	// Load from environment variables (this will override .env values)
	err = loadFromEnv(config)
	if err != nil {
		return config, err
	}

	return config, nil
}

// loadFromEnv loads configuration from environment variables using reflection
func loadFromEnv(config *AppConfig) error {
	v := reflect.ValueOf(config).Elem()
	t := reflect.TypeOf(config).Elem()

	return processStruct(v, t)
}

func processStruct(v reflect.Value, t reflect.Type) error {
	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldType := t.Field(i)

		if field.Kind() == reflect.Struct {
			// Recursively process nested structs
			err := processStruct(field, fieldType.Type)
			if err != nil {
				return err
			}
		} else {
			// Process individual fields with env tags
			envTag := fieldType.Tag.Get("env")
			if envTag != "" {
				envValue := os.Getenv(envTag)
				if envValue != "" {
					err := setFieldValue(field, envValue)
					if err != nil {
						return err
					}
				}
			}
		}
	}
	return nil
}

func setFieldValue(field reflect.Value, value string) error {
	if !field.CanSet() {
		return nil
	}

	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		intVal, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			// If parsing fails, leave as zero value
			return nil
		}
		field.SetInt(intVal)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		uintVal, err := strconv.ParseUint(value, 10, 64)
		if err != nil {
			// If parsing fails, leave as zero value
			return nil
		}
		field.SetUint(uintVal)
	case reflect.Float32, reflect.Float64:
		floatVal, err := strconv.ParseFloat(value, 64)
		if err != nil {
			// If parsing fails, leave as zero value
			return nil
		}
		field.SetFloat(floatVal)
	case reflect.Bool:
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			// If parsing fails, leave as zero value
			return nil
		}
		field.SetBool(boolVal)
	}

	return nil
}
