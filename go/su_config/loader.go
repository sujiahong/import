package su_config

import (
	"encoding/json"
	"errors"
	"os"
)

func Load(path string, out any) error {
	if out == nil {
		return errors.New("nil config output")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, out); err != nil {
		return err
	}
	return validate(out)
}

func MustLoad(path string, out any) {
	if err := Load(path, out); err != nil {
		panic(err)
	}
}

func LoadWithEnv(path, prefix string, out any) error {
	if err := Load(path, out); err != nil {
		return err
	}
	if err := LoadEnv(prefix, out); err != nil {
		return err
	}
	return validate(out)
}
