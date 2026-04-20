package framework

const (
	// Claims -> authentication claims
	Claims = "Claims"

	// UID -> authenticated user's public UUID (string)
	UID = "UID"

	// TenantID -> active tenant UUID (string) from access token
	TenantID = "TenantID"

	// UserDBID -> authenticated user's primary key (users.id) from access token
	UserDBID = "UserDBID"

	// File uploaded file from file upload middleware
	File = "@uploaded_file"

	// Limit for get all api
	Limit = "Limit"

	// Page
	Page = "Page"

	// Rate Limit
	RateLimit = "RateLimit"

	// Token -> bearer token
	Token = "Token"

	Role = "Role"

	// Permissions -> effective permission keys from tenant JWT ([]string)
	Permissions = "Permissions"

	// PlatformUserRole -> users.role string from platform JWT (admin / system_manager)
	PlatformUserRole = "PlatformUserRole"
)
