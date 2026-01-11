package io

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
)

// EnsureDir makes sure the parent directory exists.
func EnsureDir(dir string) error {
	if dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}

// WriteFileAtomic writes data to a temporary file before renaming it to the target path.
func WriteFileAtomic(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	if err := EnsureDir(dir); err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, perm); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// WriteJSONAtomic serializes v to JSON and writes it atomically.
func WriteJSONAtomic(path string, v interface{}, indent bool) error {
	var data []byte
	var err error
	if indent {
		data, err = json.MarshalIndent(v, "", "  ")
	} else {
		data, err = json.Marshal(v)
	}
	if err != nil {
		return err
	}
	return WriteFileAtomic(path, data, 0o644)
}

// ReadJSONFile loads JSON data from path into v.
func ReadJSONFile(path string, v interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, v)
}
