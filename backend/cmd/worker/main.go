package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/Eddrick-23/Logarithm/internal/transport"
)
func main() {
	// config := config.LoadConfig()
	// _, err := storage.NewClickHouseStore(context.Background(),
	// 	config.DBAddress,
	// 	config.DBName,
	// 	config.DBTableName,
	// 	config.DBUser,
	// 	config.DBPassword)

	// if err != nil {
	// 	fmt.Println(err)
	// }

	handler := slog.NewTextHandler(os.Stdout, nil)
	logger := slog.New(handler)
	ctx := context.Background()	
	nb, err := transport.NewNatsBroker(ctx, logger, "nats://localhost:4222")

	if err != nil {
		fmt.Println(err)
	}
	defer nb.Close()

	// listen any subject starting with logs.<...> e.g. logs.<serviceName>
	_, err = nb.CreateStream(ctx, "TEST_STREAM", "logs.>")

	if err != nil {
		fmt.Println(err)
	}

}
