package auth

// RegisterRequest creates a user, tenant, and owner membership.
type RegisterRequest struct {
	Email      string `json:"email" binding:"required,email"`
	Password   string `json:"password" binding:"required,min=8"`
	TenantName string `json:"tenant_name" binding:"required,min=1,max=255"`
}

// SignupRequest creates an unverified user without a tenant (self-serve onboarding).
type SignupRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
}

// SignupResponse confirms signup; verification_token is only set in ENVIRONMENT=local.
type SignupResponse struct {
	Message             string  `json:"message"`
	VerificationToken *string `json:"verification_token,omitempty"`
}

// VerifyEmailRequest completes email verification with a token from the email (or local dev response).
type VerifyEmailRequest struct {
	Token string `json:"token" binding:"required"`
}

// ResendVerificationRequest asks for a new verification email.
type ResendVerificationRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ResendVerificationResponse is uniform for unknown emails; verification_token may be set in local dev.
type ResendVerificationResponse struct {
	Message             string  `json:"message"`
	VerificationToken *string `json:"verification_token,omitempty"`
}

// CreateTenantRequest creates the first tenant using a pick_tenant_token from login.
type CreateTenantRequest struct {
	TenantName string `json:"tenant_name" binding:"required,min=1,max=255"`
}

// AcceptInviteRequest completes an invitation with token and password (existing users must supply their current password).
type AcceptInviteRequest struct {
	Token    string `json:"token" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

// CreateTenantInviteRequest is sent by a tenant owner/admin to invite by email.
type CreateTenantInviteRequest struct {
	Email string `json:"email" binding:"required,email"`
	Role  string `json:"role" binding:"required"`
}

// TenantInviteResponse confirms the invite; invite_token is only set in ENVIRONMENT=local.
type TenantInviteResponse struct {
	Message     string  `json:"message"`
	InviteToken *string `json:"invite_token,omitempty"`
}

// LoginRequest authenticates with email and password. Response lists tenants; use pick_tenant_token with POST /auth/tenant-session.
type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// TenantChoice is one organization the user may open a session for.
type TenantChoice struct {
	TenantID string `json:"tenant_id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	Role     string `json:"role"`
}

// LoginDiscoveryResponse is returned from POST /auth/login (before choosing a tenant).
type LoginDiscoveryResponse struct {
	User                  UserResponse   `json:"user"`
	Tenants               []TenantChoice `json:"tenants"`
	PickTenantToken       string         `json:"pick_tenant_token"`
	PickTenantExpiresIn   int64          `json:"pick_tenant_expires_in"`
}

// TenantSessionRequest exchanges pick_tenant_token + tenant_id for access and refresh tokens.
type TenantSessionRequest struct {
	TenantID string `json:"tenant_id" binding:"required,uuid"`
}

// RefreshRequest exchanges a refresh token for new tokens.
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// LogoutRequest revokes a refresh token.
type LogoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// TokenResponse is returned from register, login, and refresh.
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
}

// UserResponse is a safe subset of user fields for /auth/me.
type UserResponse struct {
	UUID      string `json:"uuid"`
	Email     string `json:"email"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Role      string `json:"role"`
}

// MeResponse combines user and tenant context.
type MeResponse struct {
	User          UserResponse `json:"user"`
	TenantID      string       `json:"tenant_id"`
	TenantRole    string       `json:"tenant_role"`
	IsActive      bool         `json:"is_active"`
	EmailVerified bool         `json:"is_email_verified"`
}

// ForgotPasswordRequest starts a password reset (token is emailed in production; see devguide).
type ForgotPasswordRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// ForgotPasswordResponse is always returned with the same shape; reset_token is only set in local dev.
type ForgotPasswordResponse struct {
	Message    string  `json:"message"`
	ResetToken *string `json:"reset_token,omitempty"`
}

// ResetPasswordRequest completes recovery with a token from forgot-password.
type ResetPasswordRequest struct {
	Token       string `json:"token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8"`
}

// ChangePasswordRequest updates password for the authenticated user.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// UpdateMeRequest patches profile fields; omit or null to leave unchanged.
type UpdateMeRequest struct {
	FirstName   *string `json:"first_name"`
	LastName    *string `json:"last_name"`
	FirstNameJa *string `json:"first_name_ja"`
	LastNameJa  *string `json:"last_name_ja"`
}
