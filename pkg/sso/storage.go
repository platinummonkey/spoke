package sso

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

// Storage handles SSO provider configuration storage
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new SSO storage
func NewStorage(db *sql.DB) *Storage {
	return &Storage{db: db}
}

// CreateProvider creates a new SSO provider configuration
func (s *Storage) CreateProvider(config *ProviderConfig) error {
	// Marshal configs to JSON
	var samlConfigJSON, oauth2ConfigJSON, oidcConfigJSON, groupMappingJSON, attrMappingJSON []byte
	var err error

	if config.SAMLConfig != nil {
		samlConfigJSON, err = json.Marshal(config.SAMLConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal SAML config: %w", err)
		}
	}

	if config.OAuth2Config != nil {
		oauth2ConfigJSON, err = json.Marshal(config.OAuth2Config)
		if err != nil {
			return fmt.Errorf("failed to marshal OAuth2 config: %w", err)
		}
	}

	if config.OIDCConfig != nil {
		oidcConfigJSON, err = json.Marshal(config.OIDCConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal OIDC config: %w", err)
		}
	}

	if len(config.GroupMapping) > 0 {
		groupMappingJSON, err = json.Marshal(config.GroupMapping)
		if err != nil {
			return fmt.Errorf("failed to marshal group mapping: %w", err)
		}
	}

	attrMappingJSON, err = json.Marshal(config.AttributeMapping)
	if err != nil {
		return fmt.Errorf("failed to marshal attribute mapping: %w", err)
	}

	err = s.db.QueryRow(`
		INSERT INTO sso_providers (
			name, provider_type, provider_name, enabled, auto_provision, default_role,
			saml_config, oauth2_config, oidc_config, group_mapping, attribute_mapping,
			created_at, updated_at
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, NOW(), NOW())
		RETURNING id
	`, config.Name, config.ProviderType, config.ProviderName, config.Enabled,
		config.AutoProvision, config.DefaultRole, samlConfigJSON, oauth2ConfigJSON,
		oidcConfigJSON, groupMappingJSON, attrMappingJSON).Scan(&config.ID)

	return err
}

// GetProvider retrieves a provider by name
func (s *Storage) GetProvider(name string) (*ProviderConfig, error) {
	var (
		samlConfigJSON    []byte
		oauth2ConfigJSON  []byte
		oidcConfigJSON    []byte
		groupMappingJSON  []byte
		attrMappingJSON   []byte
	)

	config := &ProviderConfig{}
	err := s.db.QueryRow(`
		SELECT id, name, provider_type, provider_name, enabled, auto_provision, default_role,
			saml_config, oauth2_config, oidc_config, group_mapping, attribute_mapping,
			created_at, updated_at
		FROM sso_providers
		WHERE name = $1
	`, name).Scan(
		&config.ID, &config.Name, &config.ProviderType, &config.ProviderName,
		&config.Enabled, &config.AutoProvision, &config.DefaultRole,
		&samlConfigJSON, &oauth2ConfigJSON, &oidcConfigJSON,
		&groupMappingJSON, &attrMappingJSON,
		&config.CreatedAt, &config.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	if len(samlConfigJSON) > 0 {
		config.SAMLConfig = &SAMLConfig{}
		if err := json.Unmarshal(samlConfigJSON, config.SAMLConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal SAML config: %w", err)
		}
	}

	if len(oauth2ConfigJSON) > 0 {
		config.OAuth2Config = &OAuth2Config{}
		if err := json.Unmarshal(oauth2ConfigJSON, config.OAuth2Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal OAuth2 config: %w", err)
		}
	}

	if len(oidcConfigJSON) > 0 {
		config.OIDCConfig = &OIDCConfig{}
		if err := json.Unmarshal(oidcConfigJSON, config.OIDCConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal OIDC config: %w", err)
		}
	}

	if len(groupMappingJSON) > 0 {
		if err := json.Unmarshal(groupMappingJSON, &config.GroupMapping); err != nil {
			return nil, fmt.Errorf("failed to unmarshal group mapping: %w", err)
		}
	}

	if err := json.Unmarshal(attrMappingJSON, &config.AttributeMapping); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attribute mapping: %w", err)
	}

	return config, nil
}

// GetProviderByID retrieves a provider by ID
func (s *Storage) GetProviderByID(id int64) (*ProviderConfig, error) {
	var (
		samlConfigJSON    []byte
		oauth2ConfigJSON  []byte
		oidcConfigJSON    []byte
		groupMappingJSON  []byte
		attrMappingJSON   []byte
	)

	config := &ProviderConfig{}
	err := s.db.QueryRow(`
		SELECT id, name, provider_type, provider_name, enabled, auto_provision, default_role,
			saml_config, oauth2_config, oidc_config, group_mapping, attribute_mapping,
			created_at, updated_at
		FROM sso_providers
		WHERE id = $1
	`, id).Scan(
		&config.ID, &config.Name, &config.ProviderType, &config.ProviderName,
		&config.Enabled, &config.AutoProvision, &config.DefaultRole,
		&samlConfigJSON, &oauth2ConfigJSON, &oidcConfigJSON,
		&groupMappingJSON, &attrMappingJSON,
		&config.CreatedAt, &config.UpdatedAt)

	if err != nil {
		return nil, err
	}

	// Unmarshal JSON fields
	if len(samlConfigJSON) > 0 {
		config.SAMLConfig = &SAMLConfig{}
		if err := json.Unmarshal(samlConfigJSON, config.SAMLConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal SAML config: %w", err)
		}
	}

	if len(oauth2ConfigJSON) > 0 {
		config.OAuth2Config = &OAuth2Config{}
		if err := json.Unmarshal(oauth2ConfigJSON, config.OAuth2Config); err != nil {
			return nil, fmt.Errorf("failed to unmarshal OAuth2 config: %w", err)
		}
	}

	if len(oidcConfigJSON) > 0 {
		config.OIDCConfig = &OIDCConfig{}
		if err := json.Unmarshal(oidcConfigJSON, config.OIDCConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal OIDC config: %w", err)
		}
	}

	if len(groupMappingJSON) > 0 {
		if err := json.Unmarshal(groupMappingJSON, &config.GroupMapping); err != nil {
			return nil, fmt.Errorf("failed to unmarshal group mapping: %w", err)
		}
	}

	if err := json.Unmarshal(attrMappingJSON, &config.AttributeMapping); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attribute mapping: %w", err)
	}

	return config, nil
}

// ListProviders lists all SSO providers
func (s *Storage) ListProviders(enabledOnly bool) ([]*ProviderConfig, error) {
	query := `
		SELECT id, name, provider_type, provider_name, enabled, auto_provision, default_role,
			saml_config, oauth2_config, oidc_config, group_mapping, attribute_mapping,
			created_at, updated_at
		FROM sso_providers
	`
	if enabledOnly {
		query += " WHERE enabled = true"
	}
	query += " ORDER BY name"

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var providers []*ProviderConfig
	for rows.Next() {
		var (
			samlConfigJSON    []byte
			oauth2ConfigJSON  []byte
			oidcConfigJSON    []byte
			groupMappingJSON  []byte
			attrMappingJSON   []byte
		)

		config := &ProviderConfig{}
		err := rows.Scan(
			&config.ID, &config.Name, &config.ProviderType, &config.ProviderName,
			&config.Enabled, &config.AutoProvision, &config.DefaultRole,
			&samlConfigJSON, &oauth2ConfigJSON, &oidcConfigJSON,
			&groupMappingJSON, &attrMappingJSON,
			&config.CreatedAt, &config.UpdatedAt)
		if err != nil {
			return nil, err
		}

		// Unmarshal JSON fields
		if len(samlConfigJSON) > 0 {
			config.SAMLConfig = &SAMLConfig{}
			if err := json.Unmarshal(samlConfigJSON, config.SAMLConfig); err != nil {
				return nil, fmt.Errorf("failed to unmarshal SAML config: %w", err)
			}
		}

		if len(oauth2ConfigJSON) > 0 {
			config.OAuth2Config = &OAuth2Config{}
			if err := json.Unmarshal(oauth2ConfigJSON, config.OAuth2Config); err != nil {
				return nil, fmt.Errorf("failed to unmarshal OAuth2 config: %w", err)
			}
		}

		if len(oidcConfigJSON) > 0 {
			config.OIDCConfig = &OIDCConfig{}
			if err := json.Unmarshal(oidcConfigJSON, config.OIDCConfig); err != nil {
				return nil, fmt.Errorf("failed to unmarshal OIDC config: %w", err)
			}
		}

		if len(groupMappingJSON) > 0 {
			if err := json.Unmarshal(groupMappingJSON, &config.GroupMapping); err != nil {
				return nil, fmt.Errorf("failed to unmarshal group mapping: %w", err)
			}
		}

		if err := json.Unmarshal(attrMappingJSON, &config.AttributeMapping); err != nil {
			return nil, fmt.Errorf("failed to unmarshal attribute mapping: %w", err)
		}

		providers = append(providers, config)
	}

	return providers, rows.Err()
}

// UpdateProvider updates an existing provider
func (s *Storage) UpdateProvider(config *ProviderConfig) error {
	// Marshal configs to JSON
	var samlConfigJSON, oauth2ConfigJSON, oidcConfigJSON, groupMappingJSON, attrMappingJSON []byte
	var err error

	if config.SAMLConfig != nil {
		samlConfigJSON, err = json.Marshal(config.SAMLConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal SAML config: %w", err)
		}
	}

	if config.OAuth2Config != nil {
		oauth2ConfigJSON, err = json.Marshal(config.OAuth2Config)
		if err != nil {
			return fmt.Errorf("failed to marshal OAuth2 config: %w", err)
		}
	}

	if config.OIDCConfig != nil {
		oidcConfigJSON, err = json.Marshal(config.OIDCConfig)
		if err != nil {
			return fmt.Errorf("failed to marshal OIDC config: %w", err)
		}
	}

	if len(config.GroupMapping) > 0 {
		groupMappingJSON, err = json.Marshal(config.GroupMapping)
		if err != nil {
			return fmt.Errorf("failed to marshal group mapping: %w", err)
		}
	}

	attrMappingJSON, err = json.Marshal(config.AttributeMapping)
	if err != nil {
		return fmt.Errorf("failed to marshal attribute mapping: %w", err)
	}

	_, err = s.db.Exec(`
		UPDATE sso_providers
		SET provider_type = $1, provider_name = $2, enabled = $3, auto_provision = $4,
			default_role = $5, saml_config = $6, oauth2_config = $7, oidc_config = $8,
			group_mapping = $9, attribute_mapping = $10, updated_at = NOW()
		WHERE id = $11
	`, config.ProviderType, config.ProviderName, config.Enabled, config.AutoProvision,
		config.DefaultRole, samlConfigJSON, oauth2ConfigJSON, oidcConfigJSON,
		groupMappingJSON, attrMappingJSON, config.ID)

	return err
}

// DeleteProvider deletes a provider
func (s *Storage) DeleteProvider(name string) error {
	_, err := s.db.Exec(`DELETE FROM sso_providers WHERE name = $1`, name)
	return err
}

// ProviderExists checks if a provider with the given name exists
func (s *Storage) ProviderExists(name string) (bool, error) {
	var exists bool
	err := s.db.QueryRow(`SELECT EXISTS(SELECT 1 FROM sso_providers WHERE name = $1)`, name).Scan(&exists)
	return exists, err
}
