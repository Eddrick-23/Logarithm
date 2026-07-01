//go:build integration

package integration

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	chmodule "github.com/testcontainers/testcontainers-go/modules/clickhouse" // alias to avoid naming conflict
	natsmodule "github.com/testcontainers/testcontainers-go/modules/nats"     // alias to avoid naming conflict
	"github.com/testcontainers/testcontainers-go/wait"
)

const (
	user     = "clickhouse"
	password = "password"
	dbName   = "logarithm"
)

// Populated by TestMain before any tests run
// Ports populated by docker
var (
	dbAddr  string
	natsUrl string
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	natsContainer, err := natsmodule.Run(
		ctx,
		"nats:2.14-alpine",
		testcontainers.WithCmd("-js"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("Server is ready"),
		),
	)

	if err != nil {
		log.Fatalf("failed to start container: %s", err)
	}

	defer func() {
		if err := testcontainers.TerminateContainer(natsContainer); err != nil {
			log.Printf("failed to terminate container: %s", err)
			return
		}
	}()

	natsHost, err := natsContainer.Host(ctx)
	if err != nil {
		log.Fatalf("failed to get nats host: %s", err)
	}

	natsPort, err := natsContainer.MappedPort(ctx, "4222/tcp")
	if err != nil {
		log.Fatalf("failed to get nats port: %s", err)
	}

	natsUrl = fmt.Sprintf("nats://%s:%s", natsHost, natsPort.Port())

	clickHouseContainer, err := chmodule.Run(ctx,
		"clickhouse/clickhouse-server:26.3-alpine",
		chmodule.WithUsername(user),
		chmodule.WithPassword(password),
		chmodule.WithDatabase(dbName),
		chmodule.WithInitScripts(filepath.Join("testdata", "init-db.sql")),
	)

	defer func() {
		if clickHouseContainer != nil {
			if err := testcontainers.TerminateContainer(clickHouseContainer); err != nil {
				fmt.Printf("failed to terminate container: %s", err)
			}
		}
	}()
	if err != nil {
		fmt.Printf("failed to start container: %s", err)
		return
	}

	clickhouseHost, err := clickHouseContainer.Host(ctx)
	if err != nil {
		fmt.Printf("failed to get host: %v", err)
	}
	clickhousePort, err := clickHouseContainer.MappedPort(ctx, "9000/tcp")
	if err != nil {
		fmt.Printf("failed to get port: %v", err)
	}

	dbAddr = fmt.Sprintf("%s:%s", clickhouseHost, clickhousePort.Port())
	exitVal := m.Run()

	os.Exit(exitVal)
}
