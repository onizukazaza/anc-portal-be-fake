package postgres

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
)

func MigrateUp(databaseURL, migrationsPath string) error {
	m, err := newMigrate(databaseURL, migrationsPath)
	if err != nil {
		return err
	}
	defer closeMigrate(m)

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up failed: %w", err)
	}
	return nil
}

func MigrateDown(databaseURL, migrationsPath string) error {
	m, err := newMigrate(databaseURL, migrationsPath)
	if err != nil {
		return err
	}
	defer closeMigrate(m)

	if err := m.Down(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate down failed: %w", err)
	}
	return nil
}

func MigrateSteps(databaseURL, migrationsPath string, steps int) error {
	m, err := newMigrate(databaseURL, migrationsPath)
	if err != nil {
		return err
	}
	defer closeMigrate(m)

	if err := m.Steps(steps); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate steps failed: %w", err)
	}
	return nil
}

func ShowMigrationVersion(databaseURL, migrationsPath string) error {
	m, err := newMigrate(databaseURL, migrationsPath)
	if err != nil {
		return err
	}
	defer closeMigrate(m)

	version, dirty, err := m.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			fmt.Println("migration version: none (no migrations applied yet)")
			return nil
		}
		return fmt.Errorf("get migration version failed: %w", err)
	}

	fmt.Printf("migration version: %d (dirty=%v)\n", version, dirty)
	return nil
}

func ForceMigrationVersion(databaseURL, migrationsPath string, version int) error {
	m, err := newMigrate(databaseURL, migrationsPath)
	if err != nil {
		return err
	}
	defer closeMigrate(m)

	if err := m.Force(version); err != nil {
		return fmt.Errorf("force migration version failed: %w", err)
	}
	return nil
}

func newMigrate(databaseURL, migrationsPath string) (*migrate.Migrate, error) {
	m, err := migrate.New("file://"+migrationsPath, databaseURL)
	if err != nil {
		return nil, fmt.Errorf("create migrate instance failed: %w", err)
	}
	return m, nil
}

func closeMigrate(m *migrate.Migrate) {
	srcErr, dbErr := m.Close()
	if srcErr != nil {
		fmt.Printf("warning: close source failed: %v\n", srcErr)
	}
	if dbErr != nil {
		fmt.Printf("warning: close database failed: %v\n", dbErr)
	}
}
