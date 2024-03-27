package main

import (
	"flag"
	"fmt"
	"log"
	"net/url"
	"os/exec"

	"github.com/svoboond/spinus/internal/conf"
)

func main() {
	localConfPath := flag.String("config", "", "configuration file path")
	flag.Parse()
	config, err := conf.New(*localConfPath)
	if err != nil {
		log.Fatal("could not create config: %w", err)
	}
	postgresUrl := url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", config.Postgres.Host, config.Postgres.Port),
		User:   url.UserPassword(config.Postgres.Username, config.Postgres.Password),
		Path:   config.Postgres.Name,
	}
	cmd := exec.Command("./sqlc", "generate")
	cmd.Env = append(cmd.Env, fmt.Sprintf("POSTGRES_URI=%s", postgresUrl.String()))
	_, err = cmd.Output()
	if err != nil {
		log.Fatal("error calling sqlc: %w", err)
	}
}
