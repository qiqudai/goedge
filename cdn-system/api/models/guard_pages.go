package models

// GuardPageDefinition stores CC challenge HTML template and localized strings.
type GuardPageDefinition struct {
	Template string                       `json:"template"`
	Strings  map[string]map[string]string `json:"strings"`
}
