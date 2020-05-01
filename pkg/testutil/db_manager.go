package testutil

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"

	"github.com/golang-migrate/migrate"
	"github.com/golang-migrate/migrate/database/postgres"
	_ "github.com/golang-migrate/migrate/source/file"
	"github.com/lib/pq"
	_ "github.com/stretchr/testify"
	"gopkg.in/khaiql/dbcleaner.v2"
	"gopkg.in/khaiql/dbcleaner.v2/engine"

	"github.com/Bnei-Baruch/gxydb-api/common"
	"github.com/Bnei-Baruch/gxydb-api/models"
	"github.com/Bnei-Baruch/gxydb-api/pkg/stringutil"
)

type TestDBManager struct {
	DB        *sql.DB
	DBCleaner dbcleaner.DbCleaner
	testDB    string
}

func (m *TestDBManager) InitTestDB() error {
	//boil.DebugMode = true

	m.testDB = fmt.Sprintf("test_%s", strings.ToLower(stringutil.GenerateName(5)))
	fmt.Println("Initializing test DB: ", m.testDB)

	// Open connection
	db, err := sql.Open("postgres", common.Config.DBUrl)
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

	// Connect to temp database and run migrations
	dsn, err := m.replaceDBName(m.testDB)
	if err != nil {
		return err
	}
	if err := m.runMigrations(dsn); err != nil {
		return err
	}

	// Re-connect to test DB since migrator closes his connection
	m.DB, err = sql.Open("postgres", dsn)
	if err != nil {
		return err
	}

	// Initialize DB cleaner
	m.DBCleaner = dbcleaner.New()
	m.DBCleaner.SetEngine(engine.NewPostgresEngine(dsn))

	return nil
}

func (m *TestDBManager) DestroyTestDB() error {
	fmt.Println("Destroying testDB: ", m.testDB)

	// Close DB cleaner
	if err := m.DBCleaner.Close(); err != nil {
		return err
	}

	// Close temp DB
	if err := m.DB.Close(); err != nil {
		return err
	}

	// Connect to main dev DB
	db, err := sql.Open("postgres", common.Config.DBUrl)
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

func (m *TestDBManager) AllTables() []string {
	v := reflect.ValueOf(models.TableNames)
	t := v.Type()
	tables := make([]string, 0)
	for i := 0; i < t.NumField(); i++ {
		name := t.Field(i).Name
		value := v.FieldByName(name).Interface()
		if value.(string) != models.TableNames.SchemaMigrations {
			tables = append(tables, value.(string))
		}
	}
	return tables
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

	_, filename, _, _ := runtime.Caller(0)
	rel := filepath.Join(filepath.Dir(filename), "..", "..", "migrations")
	migrator, err := migrate.NewWithDatabaseInstance(fmt.Sprintf("file://%s", rel), "postgres", driver)
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
	paramsStr, err := pq.ParseURL(common.Config.DBUrl)
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
