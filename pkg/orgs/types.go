package orgs

import (
	"time"

	"github.com/platinummonkey/spoke/pkg/auth"
)

// PlanTier represents subscription plan tiers
type PlanTier string

const (
	PlanFree       PlanTier = "free"
	PlanPro        PlanTier = "pro"
	PlanEnterprise PlanTier = "enterprise"
	PlanCustom     PlanTier = "custom"
)

// OrgStatus represents organization status
type OrgStatus string

const (
	OrgStatusActive    OrgStatus = "active"
	OrgStatusSuspended OrgStatus = "suspended"
	OrgStatusDeleted   OrgStatus = "deleted"
)

// Organization represents an organization with extended fields
type Organization struct {
	ID          int64              `json:"id"`
	Name        string             `json:"name"`
	Slug        string             `json:"slug"`
	DisplayName string             `json:"display_name"`
	Description string             `json:"description,omitempty"`
	OwnerID     *int64             `json:"owner_id,omitempty"`
	PlanTier    PlanTier           `json:"plan_tier"`
	Status      OrgStatus          `json:"status"`
	IsActive    bool               `json:"is_active"`
	Settings    map[string]any     `json:"settings,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// OrgQuotas represents resource quotas for an organization
type OrgQuotas struct {
	ID                       int64     `json:"id"`
	OrgID                    int64     `json:"org_id"`
	MaxModules               int       `json:"max_modules"`
	MaxVersionsPerModule     int       `json:"max_versions_per_module"`
	MaxStorageBytes          int64     `json:"max_storage_bytes"`
	MaxCompileJobsPerMonth   int       `json:"max_compile_jobs_per_month"`
	APIRateLimitPerHour      int       `json:"api_rate_limit_per_hour"`
	CreatedAt                time.Time `json:"created_at"`
	UpdatedAt                time.Time `json:"updated_at"`
}

// OrgUsage represents current usage for an organization
type OrgUsage struct {
	ID                int64     `json:"id"`
	OrgID             int64     `json:"org_id"`
	PeriodStart       time.Time `json:"period_start"`
	PeriodEnd         time.Time `json:"period_end"`
	ModulesCount      int       `json:"modules_count"`
	VersionsCount     int       `json:"versions_count"`
	StorageBytes      int64     `json:"storage_bytes"`
	CompileJobsCount  int       `json:"compile_jobs_count"`
	APIRequestsCount  int64     `json:"api_requests_count"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}

// OrgInvitation represents an invitation to join an organization
type OrgInvitation struct {
	ID         int64     `json:"id"`
	OrgID      int64     `json:"org_id"`
	Email      string    `json:"email"`
	Role       auth.Role `json:"role"`
	Token      string    `json:"token,omitempty"`
	InvitedBy  int64     `json:"invited_by"`
	InvitedAt  time.Time `json:"invited_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	AcceptedAt *time.Time `json:"accepted_at,omitempty"`
	AcceptedBy *int64    `json:"accepted_by,omitempty"`
}

// OrgMember represents an organization member with full details
type OrgMember struct {
	ID             int64     `json:"id"`
	OrganizationID int64     `json:"organization_id"`
	UserID         int64     `json:"user_id"`
	Role           auth.Role `json:"role"`
	InvitedBy      *int64    `json:"invited_by,omitempty"`
	JoinedAt       time.Time `json:"joined_at"`
	CreatedAt      time.Time `json:"created_at"`
	Username       string    `json:"username"`
	Email          string    `json:"email,omitempty"`
	FullName       string    `json:"full_name,omitempty"`
	IsBot          bool      `json:"is_bot"`
}

// CreateOrgRequest represents request to create an organization
type CreateOrgRequest struct {
	Name        string         `json:"name"`
	DisplayName string         `json:"display_name"`
	Description string         `json:"description,omitempty"`
	PlanTier    PlanTier       `json:"plan_tier,omitempty"`
	Settings    map[string]any `json:"settings,omitempty"`
}

// UpdateOrgRequest represents request to update an organization
type UpdateOrgRequest struct {
	DisplayName *string        `json:"display_name,omitempty"`
	Description *string        `json:"description,omitempty"`
	Settings    map[string]any `json:"settings,omitempty"`
}

// InviteMemberRequest represents request to invite a member
type InviteMemberRequest struct {
	Email string    `json:"email"`
	Role  auth.Role `json:"role"`
}

// UpdateMemberRequest represents request to update a member's role
type UpdateMemberRequest struct {
	Role auth.Role `json:"role"`
}

// QuotaExceededError represents a quota exceeded error
type QuotaExceededError struct {
	Resource string
	Current  int64
	Limit    int64
}

func (e *QuotaExceededError) Error() string {
	return "quota exceeded for " + e.Resource
}

// IsQuotaExceeded checks if an error is a quota exceeded error
func IsQuotaExceeded(err error) bool {
	_, ok := err.(*QuotaExceededError)
	return ok
}

// QuotaChecker defines the interface for checking quotas
type QuotaChecker interface {
	CheckModuleQuota(orgID int64) error
	CheckVersionQuota(orgID int64, moduleName string) error
	CheckStorageQuota(orgID int64, additionalBytes int64) error
	CheckCompileJobQuota(orgID int64) error
	CheckAPIRateLimit(orgID int64) error
}

// UsageTracker defines the interface for tracking usage
type UsageTracker interface {
	IncrementModules(orgID int64) error
	IncrementVersions(orgID int64) error
	IncrementStorage(orgID int64, bytes int64) error
	IncrementCompileJobs(orgID int64) error
	IncrementAPIRequests(orgID int64) error
	DecrementModules(orgID int64) error
	DecrementVersions(orgID int64) error
	DecrementStorage(orgID int64, bytes int64) error
}

// Service defines the interface for organization management
type Service interface {
	// Organization CRUD
	CreateOrganization(org *Organization) error
	GetOrganization(id int64) (*Organization, error)
	GetOrganizationBySlug(slug string) (*Organization, error)
	ListOrganizations(userID int64) ([]*Organization, error)
	UpdateOrganization(id int64, updates *UpdateOrgRequest) error
	DeleteOrganization(id int64) error

	// Quota management
	GetQuotas(orgID int64) (*OrgQuotas, error)
	UpdateQuotas(orgID int64, quotas *OrgQuotas) error
	GetDefaultQuotas(planTier PlanTier) *OrgQuotas

	// Usage tracking
	GetUsage(orgID int64) (*OrgUsage, error)
	GetUsageHistory(orgID int64, limit int) ([]*OrgUsage, error)
	ResetUsagePeriod(orgID int64) error

	// Member management
	ListMembers(orgID int64) ([]*OrgMember, error)
	GetMember(orgID, userID int64) (*OrgMember, error)
	AddMember(orgID, userID int64, role auth.Role, invitedBy *int64) error
	UpdateMemberRole(orgID, userID int64, role auth.Role) error
	RemoveMember(orgID, userID int64) error

	// Invitation management
	CreateInvitation(invitation *OrgInvitation) error
	GetInvitation(token string) (*OrgInvitation, error)
	ListInvitations(orgID int64) ([]*OrgInvitation, error)
	AcceptInvitation(token string, userID int64) error
	RevokeInvitation(id int64) error
	CleanupExpiredInvitations() error

	// Embed quota checker and usage tracker
	QuotaChecker
	UsageTracker
}
