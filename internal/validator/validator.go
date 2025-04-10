package validator

import (
	"encoding/json"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode/utf8"
)

// Validator contains validation errors for both fields and non-field errors.
type Validator struct {
	NonFieldErrors []string
	FieldErrors    map[string]string
}

// Common regular expressions for validation
var (
	URLRx   = regexp.MustCompile(`^(http|https)://[a-zA-Z0-9][-a-zA-Z0-9]{0,62}(\.[a-zA-Z0-9][-a-zA-Z0-9]{0,62})*(:[0-9]{1,5})?(/[-a-zA-Z0-9(%_+.~=#;:,\\*)]*)?(#[a-zA-Z0-9(%_+.~=#;:,\\*)]*)?`)
	EmailRX = regexp.MustCompile(`^[a-zA-Z0-9.!#$%&'*+/=?^_` + "`" + `{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$`)
	DateRx  = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	TimeRx  = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)(?::([0-5]\d))?$`)
	PhoneRx = regexp.MustCompile(`^\+?[0-9]{1,3}[\s-]?([0-9]{3,4}[\s-]?){2}[0-9]{3,4}$`)
	UUIDRx  = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
)

// Valid returns true if there are no validation errors.
func (v *Validator) Valid() bool {
	return len(v.FieldErrors) == 0 && len(v.NonFieldErrors) == 0
}

// AddNonFieldError adds a non-field-specific error message.
func (v *Validator) AddNonFieldError(message string) {
	v.NonFieldErrors = append(v.NonFieldErrors, message)
}

// AddFieldError adds an error message for a specific field.
// If the field already has an error, it will not be overwritten.
func (v *Validator) AddFieldError(key, message string) {
	if v.FieldErrors == nil {
		v.FieldErrors = make(map[string]string)
	}
	if _, exists := v.FieldErrors[key]; !exists {
		v.FieldErrors[key] = message
	}
}

// CheckField adds an error message for a specific field if the check fails.
func (v *Validator) CheckField(ok bool, key, message string) {
	if !ok {
		v.AddFieldError(key, message)
	}
}

// NotBlank returns true if a value is not an empty string after trimming whitespace.
func NotBlank(value string) bool {
	return strings.TrimSpace(value) != ""
}

// MaxChars returns true if a value contains no more than n characters.
func MaxChars(value string, n int) bool {
	return utf8.RuneCountInString(value) <= n
}

// MinChars returns true if a value contains at least n characters.
func MinChars(value string, n int) bool {
	return utf8.RuneCountInString(value) >= n
}

// PermittedValue returns true if a value is in a list of permitted values.
func PermittedValue[T comparable](value T, permittedValues ...T) bool {
	return slices.Contains(permittedValues, value)
}

// Matches returns true if a value matches a provided compiled regular expression pattern.
func Matches(value string, rx *regexp.Regexp) bool {
	return rx.MatchString(value)
}

// IsNumeric returns true if a string contains only numeric characters.
func IsNumeric(value string) bool {
	for _, r := range value {
		if r < '0' || r > '9' {
			return false
		}
	}
	return len(value) > 0
}

// IsAlpha returns true if a string contains only alphabetic characters.
func IsAlpha(value string) bool {
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') {
			return false
		}
	}
	return len(value) > 0
}

// IsAlphanumeric returns true if a string contains only alphanumeric characters.
func IsAlphanumeric(value string) bool {
	for _, r := range value {
		if (r < 'a' || r > 'z') && (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return len(value) > 0
}

// IsInRange returns true if a value is within a specified numeric range (inclusive).
func IsInRange[T int | int8 | int16 | int32 | int64 | float32 | float64](value, min, max T) bool {
	return value >= min && value <= max
}

// HasPrefix returns true if a string starts with the specified prefix.
func HasPrefix(value, prefix string) bool {
	return strings.HasPrefix(value, prefix)
}

// HasSuffix returns true if a string ends with the specified suffix.
func HasSuffix(value, suffix string) bool {
	return strings.HasSuffix(value, suffix)
}

// IsValidURL returns true if a string is a valid URL.
func IsValidURL(value string) bool {
	if !URLRx.MatchString(value) {
		return false
	}
	_, err := url.ParseRequestURI(value)
	return err == nil
}

// IsValidDate returns true if a string is a valid date in the specified layout.
func IsValidDate(value, layout string) bool {
	_, err := time.Parse(layout, value)
	return err == nil
}

// IsUUID returns true if a string is a valid UUID.
func IsUUID(value string) bool {
	return UUIDRx.MatchString(strings.ToLower(value))
}

// Contains returns true if a string contains the specified substring.
func Contains(value, substr string) bool {
	return strings.Contains(value, substr)
}

// IsOneOf returns true if a string is one of the provided values.
func IsOneOf(value string, options ...string) bool {
	return PermittedValue(value, options...)
}

// Required is equivalent to NotBlank but with a more descriptive name.
func Required(value string) bool {
	return NotBlank(value)
}

// IsJSON validates if a string is valid JSON by attempting to unmarshal it.
func IsJSON(value string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(value), &js) == nil
}
