package storage

import (
	"fmt"
	"regexp"
)

const (
	TableLogs    = "logs"
	TableMetrics = "metrics"
)

var validTableName = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// verify that the table name is valid
func isValidTableName(name string) bool {
	return validTableName.MatchString(name)
}

// map table to dbName.table
func initTables(dbName string) (map[string]string, error) {
	names := []string{TableLogs, TableMetrics}
	tables := make(map[string]string, len(names))
	for _, t := range names {
		if !isValidTableName(t) {
			return nil, fmt.Errorf("invalid table name: %s", t)
		}
		tables[t] = dbName + "." + t
	}
	return tables, nil
}
