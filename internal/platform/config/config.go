// Package config lê configuração de ambiente. Sem arquivo, sem framework:
// tudo vem de env var para o mesmo binário rodar em docker-compose hoje e em
// qualquer orquestrador depois.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

func String(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}

// MustString aborta o boot se faltar configuração crítica (segredo de JWT,
// DSN do banco). Falhar no start é melhor que falhar na primeira requisição.
func MustString(key string) string {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		panic("config: variável de ambiente obrigatória ausente: " + key)
	}
	return v
}

func Bool(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return def
	}
	return b
}

func Int(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	i, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return i
}

func Float(key string, def float64) float64 {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return def
	}
	return f
}

func Duration(key string, def time.Duration) time.Duration {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
