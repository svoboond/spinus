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
		log.Fatalf("could not create config: %v", err)
	}
	postgresUrl := url.URL{
		Scheme: "postgres",
		Host:   fmt.Sprintf("%s:%d", config.Postgres.Host, config.Postgres.Port),
		User:   url.UserPassword(config.Postgres.Username, config.Postgres.Password),
		Path:   config.Postgres.Name,
	}
	cmd := exec.Command("./sqlc", "generate")
	cmd.Env = append(cmd.Env, fmt.Sprintf("POSTGRES_URI=%s", postgresUrl.String()))
	stdoutStderr, err := cmd.CombinedOutput()
	if err != nil {
		log.Fatalf("error calling sqlc:\n%s", stdoutStderr)
	}
}
