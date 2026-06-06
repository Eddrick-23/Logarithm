//go:build integration

package integration

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Eddrick-23/Logarithm/internal/transport"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/testcontainers/testcontainers-go"
	chmodule "github.com/testcontainers/testcontainers-go/modules/clickhouse" // alias to avoid naming conflict
	natsmodule "github.com/testcontainers/testcontainers-go/modules/nats"     // alias to avoid naming conflict
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	natsContainer, err := natsmodule.Run(ctx, "nats:2.14-alpine", testcontainers.WithCmd("-js"))
	defer func() {
		if err := testcontainers.TerminateContainer(natsContainer); err != nil {
			log.Printf("failed to terminate container: %s", err)
			return
		}
	}()
	if err != nil {
		log.Printf("failed to start container: %s", err)
	}

	natsHost, err := natsContainer.Host(ctx)
	natsPort, err := natsContainer.MappedPort(ctx, "4222/tcp")
	natsUrl = fmt.Sprintf("nats://%s:%s", natsHost, natsPort.Port())

	nc, err := nats.Connect(natsUrl, nats.DrainTimeout(5*time.Second))
	if err != nil {
		log.Printf("failed to connect to raw NATS client: %s", err)
	}

	js, err := jetstream.New(nc)
	if err != nil {
		log.Printf("failed to create to jetstream interface: %s", err)
	}

	_, err = js.CreateStream(ctx, jetstream.StreamConfig{
		Name:     transport.LogStreamName,
		Subjects: []string{"logs.test.>"},
	})
	if err != nil {
		log.Fatalf("failed to create JetStream stream: %s", err)
	}

	user = "clickhouse"
	password = "password"
	dbname = "logarithm"
	dbtablename = "logs"

	clickHouseContainer, err := chmodule.Run(ctx,
		"clickhouse/clickhouse-server:26.3-alpine",
		chmodule.WithUsername(user),
		chmodule.WithPassword(password),
		chmodule.WithDatabase(dbname),
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
