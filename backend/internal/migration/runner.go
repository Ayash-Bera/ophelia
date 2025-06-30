package migration

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Ayash-Bera/ophelia/backend/internal/database"
	"github.com/sirupsen/logrus"
)

type Runner struct {
	dbManager *database.Manager
	logger    *logrus.Logger
}

func NewRunner(dbManager *database.Manager, logger *logrus.Logger) *Runner {
	return &Runner{
		dbManager: dbManager,
		logger:    logger,
	}
}

// RunMigrations executes all pending migrations
func (r *Runner) RunMigrations(migrationsPath string) error {
	r.logger.Info("Starting database migrations...")

	// First run GORM auto-migrations
	if err := r.dbManager.Migrate(); err != nil {
		return fmt.Errorf("GORM auto-migration failed: %w", err)
	}

	// Then run SQL migrations
	if err := r.runSQLMigrations(migrationsPath); err != nil {
		return fmt.Errorf("SQL migrations failed: %w", err)
	}

	r.logger.Info("Database migrations completed successfully")
	return nil
}

func (r *Runner) runSQLMigrations(migrationsPath string) error {
	files, err := ioutil.ReadDir(migrationsPath)
	if err != nil {
		return fmt.Errorf("failed to read migrations directory: %w", err)
	}

	var sqlFiles []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".sql") {
			sqlFiles = append(sqlFiles, file.Name())
		}
	}

	sort.Strings(sqlFiles) // Ensure migrations run in order

	for _, fileName := range sqlFiles {
		if err := r.runSQLFile(filepath.Join(migrationsPath, fileName)); err != nil {
			return fmt.Errorf("failed to run migration %s: %w", fileName, err)
		}
		r.logger.WithField("file", fileName).Info("Migration executed successfully")
	}

	return nil
}

func (r *Runner) runSQLFile(filePath string) error {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return err
	}

	// For PostgreSQL, we need to handle dollar-quoted strings properly
	// Simple approach: execute the entire file as one statement if it contains $
	sqlContent := string(content)

	if strings.Contains(sqlContent, "$") {
		r.logger.WithField("file", filepath.Base(filePath)).Debug("Executing SQL file with dollar-quoted functions")

		// Remove comments but keep the structure intact
		cleanedSQL := r.removeComments(sqlContent)

		if err := r.dbManager.DB.Exec(cleanedSQL).Error; err != nil {
			return fmt.Errorf("failed to execute %s: %w", filepath.Base(filePath), err)
		}
		return nil
	}

	// For simple SQL files, split by statements
	statements := r.splitSQLStatements(sqlContent)

	for i, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}

		r.logger.WithFields(logrus.Fields{
			"file":      filepath.Base(filePath),
			"statement": i + 1,
		}).Debug("Executing SQL statement")

		if err := r.dbManager.DB.Exec(stmt).Error; err != nil {
			return fmt.Errorf("failed to execute statement %d in %s: %w", i+1, filepath.Base(filePath), err)
		}
	}

	return nil
}

// removeComments removes SQL comments while preserving structure
func (r *Runner) removeComments(sql string) string {
	lines := strings.Split(sql, "\n")
	var result []string

	for _, line := range lines {
		// Remove comment lines but keep empty lines for structure
		if strings.HasPrefix(strings.TrimSpace(line), "--") {
			continue
		}
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// splitSQLStatements splits SQL content into individual statements
func (r *Runner) splitSQLStatements(sql string) []string {
	// Remove comments and split by semicolon
	lines := strings.Split(sql, "\n")
	var cleanedLines []string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Skip comment lines and empty lines
		if line != "" && !strings.HasPrefix(line, "--") {
			cleanedLines = append(cleanedLines, line)
		}
	}

	// Join back and split by semicolon
	cleanedSQL := strings.Join(cleanedLines, " ")
	statements := strings.Split(cleanedSQL, ";")

	var result []string
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt != "" {
			result = append(result, stmt)
		}
	}

	return result
}
