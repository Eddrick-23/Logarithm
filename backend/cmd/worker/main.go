package main

import (
	"context"
	"fmt"

	"github.com/Eddrick-23/OrbitalTest/internal/transport"
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
	ctx := context.Background()	
	nb, err := transport.NewNatsBroker(ctx, "nats://localhost:4222")

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
