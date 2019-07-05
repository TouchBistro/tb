package config

import (
	"encoding/json"
	"os"
)

var config *[]Service

type Service struct {
	Name         string `json:"name"`
	IsGithubRepo bool   `json:"repo"`
	Migrations   bool   `json:"migrations"`
	ECR          bool   `json:"ecr"`
	ImageURI     string `json:"imageURI"`
}

func Init(path string) error {
	var err error

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	dec := json.NewDecoder(file)
	err = dec.Decode(&config)
	return err
}

func All() *[]Service {
	return config
}

func BaseImages() []string {
	return []string{
		"touchbistro/alpine-node:10-build",
		"touchbistro/alpine-node:10-runtime",
	}
}
