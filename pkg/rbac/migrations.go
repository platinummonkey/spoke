package rbac

import (
	"context"
	"database/sql"
	"fmt"
)

// Migration represents a database migration
type Migration struct {
	Version     int
	Description string
	SQL         string
}

// GetMigrations returns all RBAC migrations
func GetMigrations() []Migration {
	return []Migration{
		{
			Version:     1,
			Description: "Create roles table",
			SQL: `
				CREATE TABLE IF NOT EXISTS roles (
					id BIGSERIAL PRIMARY KEY,
					name VARCHAR(255) NOT NULL,
					display_name VARCHAR(255) NOT NULL,
					description TEXT,
					organization_id BIGINT REFERENCES organizations(id) ON DELETE CASCADE,
					permissions JSONB NOT NULL DEFAULT '[]',
					parent_role_id BIGINT REFERENCES roles(id) ON DELETE SET NULL,
					is_built_in BOOLEAN DEFAULT FALSE,
					is_custom BOOLEAN DEFAULT TRUE,
					created_at TIMESTAMP NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
					created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
					UNIQUE(name, organization_id)
				);

				CREATE INDEX idx_roles_organization_id ON roles(organization_id);
				CREATE INDEX idx_roles_parent_role_id ON roles(parent_role_id);
				CREATE INDEX idx_roles_name ON roles(name);
				CREATE INDEX idx_roles_is_built_in ON roles(is_built_in);
			`,
		},
		{
			Version:     2,
			Description: "Create user_roles table",
			SQL: `
				CREATE TABLE IF NOT EXISTS user_roles (
					id BIGSERIAL PRIMARY KEY,
					user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
					scope VARCHAR(50) NOT NULL,
					resource_id VARCHAR(255),
					organization_id BIGINT REFERENCES organizations(id) ON DELETE CASCADE,
					granted_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
					granted_at TIMESTAMP NOT NULL DEFAULT NOW(),
					expires_at TIMESTAMP,
					UNIQUE(user_id, role_id, scope, COALESCE(resource_id, ''), COALESCE(organization_id, 0))
				);

				CREATE INDEX idx_user_roles_user_id ON user_roles(user_id);
				CREATE INDEX idx_user_roles_role_id ON user_roles(role_id);
				CREATE INDEX idx_user_roles_organization_id ON user_roles(organization_id);
				CREATE INDEX idx_user_roles_scope ON user_roles(scope);
				CREATE INDEX idx_user_roles_resource_id ON user_roles(resource_id);
				CREATE INDEX idx_user_roles_expires_at ON user_roles(expires_at);
			`,
		},
		{
			Version:     3,
			Description: "Create teams table",
			SQL: `
				CREATE TABLE IF NOT EXISTS teams (
					id BIGSERIAL PRIMARY KEY,
					organization_id BIGINT NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
					name VARCHAR(255) NOT NULL,
					display_name VARCHAR(255) NOT NULL,
					description TEXT,
					created_at TIMESTAMP NOT NULL DEFAULT NOW(),
					updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
					created_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
					UNIQUE(organization_id, name)
				);

				CREATE INDEX idx_teams_organization_id ON teams(organization_id);
				CREATE INDEX idx_teams_name ON teams(name);
			`,
		},
		{
			Version:     4,
			Description: "Create team_members table",
			SQL: `
				CREATE TABLE IF NOT EXISTS team_members (
					id BIGSERIAL PRIMARY KEY,
					team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
					user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					role_id BIGINT REFERENCES roles(id) ON DELETE SET NULL,
					added_at TIMESTAMP NOT NULL DEFAULT NOW(),
					added_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
					UNIQUE(team_id, user_id)
				);

				CREATE INDEX idx_team_members_team_id ON team_members(team_id);
				CREATE INDEX idx_team_members_user_id ON team_members(user_id);
			`,
		},
		{
			Version:     5,
			Description: "Create team_roles table",
			SQL: `
				CREATE TABLE IF NOT EXISTS team_roles (
					id BIGSERIAL PRIMARY KEY,
					team_id BIGINT NOT NULL REFERENCES teams(id) ON DELETE CASCADE,
					role_id BIGINT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
					scope VARCHAR(50) NOT NULL,
					resource_id VARCHAR(255),
					organization_id BIGINT REFERENCES organizations(id) ON DELETE CASCADE,
					granted_by BIGINT REFERENCES users(id) ON DELETE SET NULL,
					granted_at TIMESTAMP NOT NULL DEFAULT NOW(),
					UNIQUE(team_id, role_id, scope, COALESCE(resource_id, ''), COALESCE(organization_id, 0))
				);

				CREATE INDEX idx_team_roles_team_id ON team_roles(team_id);
				CREATE INDEX idx_team_roles_role_id ON team_roles(role_id);
				CREATE INDEX idx_team_roles_organization_id ON team_roles(organization_id);
				CREATE INDEX idx_team_roles_scope ON team_roles(scope);
				CREATE INDEX idx_team_roles_resource_id ON team_roles(resource_id);
			`,
		},
		{
			Version:     6,
			Description: "Create permission_cache table",
			SQL: `
				CREATE TABLE IF NOT EXISTS permission_cache (
					user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
					permission VARCHAR(255) NOT NULL,
					scope VARCHAR(50) NOT NULL,
					resource_id VARCHAR(255),
					organization_id BIGINT,
					allowed BOOLEAN NOT NULL,
					expires_at TIMESTAMP NOT NULL,
					created_at TIMESTAMP NOT NULL DEFAULT NOW(),
					PRIMARY KEY (user_id, permission, scope, COALESCE(resource_id, ''), COALESCE(organization_id, 0))
				);

				CREATE INDEX idx_permission_cache_expires_at ON permission_cache(expires_at);
				CREATE INDEX idx_permission_cache_user_id ON permission_cache(user_id);
			`,
		},
	}
}

// RunMigrations executes all pending migrations
func RunMigrations(ctx context.Context, db *sql.DB) error {
	// Create migration tracking table
	_, err := db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS rbac_migrations (
			version INT PRIMARY KEY,
			description TEXT NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get applied migrations
	rows, err := db.QueryContext(ctx, "SELECT version FROM rbac_migrations ORDER BY version")
	if err != nil {
		return fmt.Errorf("failed to query migrations: %w", err)
	}

	appliedVersions := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			rows.Close()
			return fmt.Errorf("failed to scan migration version: %w", err)
		}
		appliedVersions[version] = true
	}
	rows.Close()

	// Run pending migrations
	migrations := GetMigrations()
	for _, migration := range migrations {
		if appliedVersions[migration.Version] {
			continue
		}

		fmt.Printf("Running migration %d: %s\n", migration.Version, migration.Description)

		// Start transaction
		tx, err := db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to start transaction: %w", err)
		}

		// Execute migration
		if _, err := tx.ExecContext(ctx, migration.SQL); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to execute migration %d: %w", migration.Version, err)
		}

		// Record migration
		if _, err := tx.ExecContext(ctx,
			"INSERT INTO rbac_migrations (version, description) VALUES ($1, $2)",
			migration.Version, migration.Description,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %d: %w", migration.Version, err)
		}

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %d: %w", migration.Version, err)
		}

		fmt.Printf("Migration %d completed successfully\n", migration.Version)
	}

	return nil
}

// InitializeBuiltInRoles creates built-in roles if they don't exist
func InitializeBuiltInRoles(ctx context.Context, store *Store) error {
	builtInRoles := BuiltInRoles()

	for _, role := range builtInRoles {
		// Check if role already exists
		existing, err := store.GetRoleByName(ctx, role.Name, nil)
		if err == nil && existing != nil {
			// Role already exists, skip
			continue
		}

		// Create the role
		role.CreatedAt = role.UpdatedAt
		if err := store.CreateRole(ctx, &role); err != nil {
			return fmt.Errorf("failed to create built-in role %s: %w", role.Name, err)
		}

		fmt.Printf("Created built-in role: %s\n", role.Name)
	}

	return nil
}

// RollbackMigration rolls back a specific migration
func RollbackMigration(ctx context.Context, db *sql.DB, version int) error {
	// This is a placeholder for rollback logic
	// In production, you would maintain separate rollback SQL for each migration
	return fmt.Errorf("rollback not implemented for version %d", version)
}
