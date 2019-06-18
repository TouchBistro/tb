package config

import (
  "os"
  "encoding/json"
)

var config *Config

type Config []struct {
  Name string `json:"name"`
  Repo bool `json:"repo"`
  Migrations bool `json:"migrations"`
  ECR bool `json:"ecr"`
  ImageURI string `json:"imageURI"`
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

func All() *Config {
  return config
}
