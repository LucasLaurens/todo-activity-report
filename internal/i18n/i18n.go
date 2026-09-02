// Package i18n provides locale-aware translations loaded from embedded JSON
// catalogs (fr, en). French is the default.
package i18n

import (
	"embed"
	"encoding/json"
	"fmt"
	"strings"
)

//go:embed locales/*.json
var localesFS embed.FS

type Locale string

const (
	French  Locale = "fr"
	English Locale = "en"
)

const DefaultLocale = French

// ParseLocale maps a free-form value (env, flag) to a supported locale,
// falling back to the default when unknown or empty.
func ParseLocale(value string) Locale {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "en", "en_us", "en-us", "english":
		return English
	case "fr", "fr_fr", "fr-fr", "french", "":
		return French
	default:
		return DefaultLocale
	}
}

// Translator resolves message keys for one locale, falling back to the default
// locale, then to the key itself.
type Translator struct {
	locale   Locale
	messages map[string]string
	fallback map[string]string
}

func NewTranslator(locale Locale) Translator {
	return Translator{
		locale:   locale,
		messages: load(locale),
		fallback: load(DefaultLocale),
	}
}

func (t Translator) Locale() Locale {
	return t.locale
}

// T returns the message for key, formatted with args when provided.
func (t Translator) T(key string, args ...any) string {
	message, ok := t.messages[key]
	if !ok {
		message, ok = t.fallback[key]
	}
	if !ok {
		return key
	}
	if len(args) > 0 {
		return fmt.Sprintf(message, args...)
	}
	return message
}

func load(locale Locale) map[string]string {
	data, err := localesFS.ReadFile("locales/" + string(locale) + ".json")
	if err != nil {
		return map[string]string{}
	}
	messages := make(map[string]string)
	_ = json.Unmarshal(data, &messages)
	return messages
}
