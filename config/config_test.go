package config

import "testing"

func TestLoad(t *testing.T) {
	cfg := Load("../config.yaml")
	t.Log(cfg)
}
