package models

type ErrorPageI18nSettings struct {
	DefaultLang  string   `json:"default_lang"`
	LangMode     string   `json:"lang_mode"` // browser | fixed
	EnabledLangs []string `json:"enabled_langs"`
}

type ErrorPageDefinition struct {
	Template string                       `json:"template"`
	Strings  map[string]map[string]string `json:"strings"`
}
