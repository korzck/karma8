//go:build integration

package test

import (
	"context"
	"log"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/stretchr/testify/suite"

	dockertest "github.com/ory/dockertest/v3"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type Suite struct {
	suite.Suite
}

func initDB() *sqlx.DB {
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	resource, err := pool.Run("postgres", "15", []string{
		"POSTGRES_USER=postgres",
		"POSTGRES_PASSWORD=secret",
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	var db *sqlx.DB
	if err := pool.Retry(func() error {
		var err error
		dsn := "postgres://postgres:secret@localhost:" + resource.GetPort("5432/tcp") + "/postgres?sslmode=disable"
		db, err = sqlx.Open("pgx", dsn)
		if err != nil {
			return err
		}

		pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = db.PingContext(pingCtx)
		if err != nil {
			log.Printf("Ping failed, retrying: %v", err)
			return err
		}
		return nil
	}); err != nil {
		log.Fatalf("Could not connect to docker postgres: %s", err)
	}

	_, err = db.Exec("CREATE DATABASE testdb")
	if err != nil {
		log.Printf("Warning: could not create database testdb: %s", err)
	}

	db.Close()
	var serviceDB *sqlx.DB
	if err := pool.Retry(func() error {
		var err error
		dsn := "postgres://postgres:secret@localhost:" + resource.GetPort("5432/tcp") + "/testdb?sslmode=disable"
		serviceDB, err = sqlx.Open("pgx", dsn)
		if err != nil {
			return err
		}

		pingCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err = serviceDB.PingContext(pingCtx)
		if err != nil {
			log.Printf("Ping to testdb failed, retrying: %v", err)
			return err
		}
		return nil
	}); err != nil {
		log.Fatalf("Could not connect to service database testdb: %s", err)
	}

	return serviceDB
}
