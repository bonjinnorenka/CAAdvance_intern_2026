package main

import (
	"errors"
	"fmt"
	"testing"

	"github.com/go-sql-driver/mysql"
)

func TestIsDuplicateKey(t *testing.T) {
	dup := &mysql.MySQLError{Number: 1062, Message: "Duplicate entry '001_init.sql' for key 'schema_migrations.PRIMARY'"}
	if !isDuplicateKey(dup) {
		t.Fatal("expected duplicate-key error to match")
	}
	if !isDuplicateKey(fmt.Errorf("record migration: %w", dup)) {
		t.Fatal("expected wrapped duplicate-key error to match")
	}

	other := &mysql.MySQLError{Number: 1064, Message: "You have an error in your SQL syntax"}
	if isDuplicateKey(other) {
		t.Fatal("did not expect a syntax error to match")
	}
	if isDuplicateKey(errors.New("connection refused")) {
		t.Fatal("did not expect a generic error to match")
	}
}
