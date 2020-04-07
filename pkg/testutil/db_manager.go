package testutil

import (
	"database/sql"
	"fmt"
	"os"
	"strings"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	"github.com/lib/pq"

	_ "github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	_ "github.com/stretchr/testify"

	"github.com/Bnei-Baruch/gxydb-api/pkg/stringutil"
)

type TestDBManager struct {
	DB     *sql.DB
	testDB string
}

func (m *TestDBManager) InitTestDB() error {
	m.testDB = fmt.Sprintf("test_%s", strings.ToLower(stringutil.GenerateName(5)))
	fmt.Println("Initializing test DB: ", m.testDB)

	// Open connection
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		return err
	}

	// Create a new temporary test database
	if _, err := db.Exec(fmt.Sprintf("CREATE DATABASE %s", m.testDB)); err != nil {
		return err
	}

	// Close first connection
	if err := db.Close(); err != nil {
		return err
	}

	// Connect to temp database
	dsn, err := m.replaceDBName(m.testDB)
	if err != nil {
		return err
	}

	if err := m.runMigrations(dsn); err != nil {
		return err
	}

	m.DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return err
	}

	return nil
}

func (m *TestDBManager) DestroyTestDB() error {
	fmt.Println("Destroying testDB: ", m.testDB)

	// Close temp DB
	err := m.DB.Close()
	if err != nil {
		return err
	}

	// Connect to DB
	db, err := sql.Open("postgres", os.Getenv("DB_URL"))
	if err != nil {
		return err
	}

	// Drop test DB
	_, err = db.Exec(fmt.Sprintf("DROP DATABASE %s", m.testDB))
	if err != nil {
		return err
	}

	return nil
}

func (m *TestDBManager) runMigrations(dsn string) error {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return err
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return err
	}

	migrator, err := migrate.NewWithDatabaseInstance("file://migrations", "postgres", driver)
	if err != nil {
		return err
	}

	if err := migrator.Up(); err != nil {
		return err
	}

	srcErr, dbErr := migrator.Close()
	if srcErr != nil {
		return err
	}
	if dbErr != nil {
		return err
	}

	return nil
}

func (m *TestDBManager) replaceDBName(tempName string) (string, error) {
	paramsStr, err := pq.ParseURL(os.Getenv("DB_URL"))
	if err != nil {
		return "", err
	}
	params := strings.Split(paramsStr, " ")
	found := false
	for i := range params {
		if strings.HasPrefix(params[i], "dbname") {
			params[i] = fmt.Sprintf("dbname=%s", tempName)
			found = true
			break
		}
	}
	if !found {
		params = append(params, fmt.Sprintf("dbname=%s", tempName))
	}
	return strings.Join(params, " "), nil
}
