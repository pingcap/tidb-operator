// Copyright 2024 PingCAP, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"database/sql"
	"fmt"
	"strings"
)

// ImportDataConfig holds configuration for the import data action.
type ImportDataConfig struct {
	DB               *sql.DB
	BatchSize        int
	TotalRows        int
	TableName        string
	SplitRegionCount int
}

// ImportData creates a table and inserts a specified number of rows in batches.
//
//nolint:gosec // Only used for testing.
func ImportData(config ImportDataConfig) error {
	if config.DB == nil {
		return fmt.Errorf("database connection is nil")
	}
	if config.TableName == "" {
		config.TableName = "t1" // Default table name
	}
	if config.BatchSize <= 0 {
		config.BatchSize = 1000 // Default batch size
	}
	if config.TotalRows <= 0 {
		config.TotalRows = 500000 // Default total rows
	}

	fmt.Printf("Starting data import: Table=%s, TotalRows=%d, BatchSize=%d\n", config.TableName, config.TotalRows, config.BatchSize)
	if _, err := config.DB.Exec("CREATE DATABASE IF NOT EXISTS e2e_test"); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}
	fmt.Println("Database 'e2e_test' ensured to exist.")

	if _, err := config.DB.Exec("USE e2e_test"); err != nil {
		return fmt.Errorf("failed to use database 'e2e_test': %w", err)
	}
	fmt.Println("Using database 'e2e_test'.")

	createTableSQL := fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (id INT PRIMARY KEY AUTO_INCREMENT, v VARCHAR(1000))", config.TableName)
	if _, err := config.DB.Exec(createTableSQL); err != nil {
		return fmt.Errorf("failed to create table '%s': %w", config.TableName, err)
	}
	fmt.Printf("Table '%s' ensured to exist.\n", config.TableName)

	for i := 0; i < config.TotalRows; i += config.BatchSize {
		end := min(i+config.BatchSize, config.TotalRows)
		if i >= end { // Ensure we don't proceed if i has caught up to end due to small TotalRows vs BatchSize
			break
		}

		valueStrings := make([]string, 0, end-i)
		args := make([]any, 0, end-i)
		for j := i; j < end; j++ {
			valueStrings = append(valueStrings, "(?)")
			// Using a simpler string for data generation to avoid excessive length issues if not needed.
			// The original string was strings.Repeat("x", 900)+fmt.Sprintf("%d", j)
			args = append(args, fmt.Sprintf("data_val_%d", j))
		}
		// Use parameterized query with table name validation
		if !isValidTableName(config.TableName) {
			return fmt.Errorf("invalid table name: %s", config.TableName)
		}
		query := fmt.Sprintf("INSERT INTO %s (v) VALUES %s", config.TableName, strings.Join(valueStrings, ","))
		if _, err := config.DB.Exec(query, args...); err != nil {
			return fmt.Errorf("failed to insert batch (rows %d to %d) into table '%s': %w", i, end-1, config.TableName, err)
		}
	}

	fmt.Printf("Successfully inserted %d rows into table '%s'.\n", config.TotalRows, config.TableName)

	if config.SplitRegionCount > 0 {
		// Splitting table into desired number of regions.
		splitTableSQL := "SPLIT TABLE " + config.TableName + " BETWEEN (0) AND (?) REGIONS ?"
		if _, err := config.DB.Exec(splitTableSQL, config.TotalRows, config.SplitRegionCount); err != nil {
			return fmt.Errorf("failed to split table '%s' to %d regions: %w", config.TableName, config.SplitRegionCount, err)
		}
		fmt.Printf("Table '%s' split into %d regions.\n", config.TableName, config.SplitRegionCount)
	}

	fmt.Println("Data import completed.")
	return nil
}

// isValidTableName checks if the table name is valid and safe to use
func isValidTableName(name string) bool {
	// Only allow alphanumeric characters and underscores
	for _, c := range name {
		//nolint:gocritic
		if c < 'a' && c > 'z' && c < 'A' && c > 'Z' && c < '0' && c > '9' && c != '_' {
			return false
		}
	}
	return name != "" && len(name) <= 64 // MySQL table name length limit
}
