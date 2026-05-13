package main

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
	"github.com/joho/godotenv"
)

// This script migrates all workflow definitions from 'intermediate' or 'agent' node types to the new 'prompt' type.
func main() {
	_ = godotenv.Load()
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://phaxa:phaxa_secret@localhost:5433/phaxa?sslmode=disable"
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}
	defer db.Close()

	// Update yaml_source: agent -> prompt, intermediate -> prompt
	db.Exec("UPDATE workflow_definitions SET yaml_source = REPLACE(yaml_source, 'type: agent', 'type: prompt')")
	db.Exec("UPDATE workflow_definitions SET yaml_source = REPLACE(yaml_source, 'type: intermediate', 'type: prompt')")

	// Update definition JSONB: agent -> prompt, intermediate -> prompt
	db.Exec("UPDATE workflow_definitions SET definition = REPLACE(definition::text, '\"type\": \"agent\"', '\"type\": \"prompt\"')::jsonb")
	db.Exec("UPDATE workflow_definitions SET definition = REPLACE(definition::text, '\"type\": \"intermediate\"', '\"type\": \"prompt\"')::jsonb")

	fmt.Println("Successfully migrated all workflows to the 'prompt' node type.")
}
