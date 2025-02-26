package env

import (
	"os"
	"strconv"
	"time"
)

type Required bool

const (
	Optional Required = false
	Require  Required = true
)

type Env[T any] struct {
	Key      string
	Fallback T
	Required Required
}

// Get retrieves an environment variable and converts it to the desired type
func (e Env[T]) Get() T {
	value, exists := os.LookupEnv(e.Key)

	if !exists && e.Required {
		panic("Required environment variable " + e.Key + " is not set")
	}

	if !exists {
		return e.Fallback
	}

	var result T
	switch any(e.Fallback).(type) {
	case string:
		result = any(value).(T)
	case int:
		if v, err := strconv.Atoi(value); err == nil {
			result = any(v).(T)
		} else {
			if e.Required {
				panic("Failed to convert environment variable " + e.Key + " to int: " + err.Error())
			}
			result = e.Fallback
		}
	case bool:
		if v, err := strconv.ParseBool(value); err == nil {
			result = any(v).(T)
		} else {
			if e.Required {
				panic("Failed to convert environment variable " + e.Key + " to bool: " + err.Error())
			}
			result = e.Fallback
		}
	case time.Duration:
		if v, err := time.ParseDuration(value); err == nil {
			result = any(v).(T)
		} else {
			if e.Required {
				panic("Failed to convert environment variable " + e.Key + " to duration: " + err.Error())
			}
			result = e.Fallback
		}
	case float64:
		if v, err := strconv.ParseFloat(value, 64); err == nil {
			result = any(v).(T)
		} else {
			if e.Required {
				panic("Failed to convert environment variable " + e.Key + " to float64: " + err.Error())
			}
			result = e.Fallback
		}
	default:
		if e.Required {
			panic("Unsupported type for environment variable " + e.Key)
		}
		result = e.Fallback
	}

	return result
}

// String is a helper function to create a string environment variable configuration
func String(key string, fallback string, required Required) Env[string] {
	return Env[string]{
		Key:      key,
		Fallback: fallback,
		Required: required,
	}
}

// Int is a helper function to create an int environment variable configuration
func Int(key string, fallback int, required Required) Env[int] {
	return Env[int]{
		Key:      key,
		Fallback: fallback,
		Required: required,
	}
}

// Bool is a helper function to create a bool environment variable configuration
func Bool(key string, fallback bool, required Required) Env[bool] {
	return Env[bool]{
		Key:      key,
		Fallback: fallback,
		Required: required,
	}
}

// Duration is a helper function to create a time.Duration environment variable configuration
func Duration(key string, fallback time.Duration, required Required) Env[time.Duration] {
	return Env[time.Duration]{
		Key:      key,
		Fallback: fallback,
		Required: required,
	}
}

// Float64 is a helper function to create a float64 environment variable configuration
func Float64(key string, fallback float64, required Required) Env[float64] {
	return Env[float64]{
		Key:      key,
		Fallback: fallback,
		Required: required,
	}
}
