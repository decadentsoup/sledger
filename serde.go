package main

import (
	"os"

	"gopkg.in/yaml.v3"
)

func ReadYAML(path string, out any) {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	decoder.KnownFields(true)

	if err := decoder.Decode(out); err != nil {
		panic(err)
	}
}
