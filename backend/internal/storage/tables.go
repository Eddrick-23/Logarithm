package storage

import (
	"fmt"
	"regexp"
)

const (
	TableLogs            = "logs"
	TableMetrics         = "metrics"    // 1 second bucket
	TableMetrics1m       = "metrics_1m" // 1 minute bucket
	TableServiceRegistry = "service_registry"
)

// add new constants here
var allTables = []string{
	TableLogs,
	TableMetrics,
	TableMetrics1m,
	TableServiceRegistry,
}

var validTableName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// verify that the table name is valid
func isValidTableName(name string) bool {
	return validTableName.MatchString(name)
}

// map table to dbName.table
func initTables(dbName string) (map[string]string, error) {
	tables := make(map[string]string, len(allTables))
	for _, t := range allTables {
		if !isValidTableName(t) {
			return nil, fmt.Errorf("invalid table name: %s", t)
		}
		tables[t] = dbName + "." + t
	}
	return tables, nil
}
