package auth

import (
	"clean-architecture/domain/constants"
	"clean-architecture/domain/models"
	"clean-architecture/domain/tenantroles"
	"clean-architecture/pkg/errorz"
	"clean-architecture/pkg/framework"
	"clean-architecture/pkg/jwtutil"
	"clean-architecture/pkg/types"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Service handles registration, login, and token lifecycle.
type Service struct {
	repo   Repository
	env    *framework.Env
	logger framework.Logger
	perms  tenantroles.PermCalculator

	resendMu   sync.Mutex
	resendHits map[string][]time.Time
}

// NewService constructs the auth service.
func NewService(repo Repository, env *framework.Env, logger framework.Logger, perms tenantroles.PermCalculator) *Service {
	return &Service{repo: repo, env: env, logger: logger, perms: perms}
}

// Register creates user, tenant, owner membership, and returns tokens.
// Intended for bootstrap or admin flows: the email is marked verified immediately (no verification email).
func (s *Service) Register(req RegisterRequest) (*TokenResponse, error) {
	if len(req.Password) < 8 {
		return nil, errorz.ErrWeakPassword
	}
	existing, err := s.repo.FindUserByEmail(strings.ToLower(strings.TrimSpace(req.Email)))
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errorz.ErrEmailTaken
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	email := strings.ToLower(strings.TrimSpace(req.Email))
	var out *TokenResponse
	err = s.repo.Transaction(func(tx *gorm.DB) error {
		slug, err := allocateSlugTx(tx, req.TenantName)
		if err != nil {
			return err
		}
		u := &models.User{
			Email:           email,
			PasswordHash:    string(hash),
			IsActive:        true,
			IsEmailVerified: true,
		}
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		actor := u.ID
		t := &models.Tenant{
			Name: strings.TrimSpace(req.TenantName),
			Slug: slug,
			AuditFields: models.AuditFields{
				CreatedByID: &actor,
				UpdatedByID: &actor,
			},
		}
		if err := tx.Create(t).Error; err != nil {
			return err
		}
		m := &models.TenantMembership{
			UserID:   u.ID,
			TenantID: t.ID,
			Role:     constants.TenantRoleOwner,
			AuditFields: models.AuditFields{
				CreatedByID: &actor,
				UpdatedByID: &actor,
			},
		}
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		tid := t.ID
		if err := tx.Model(u).Updates(map[string]any{
			"tenant_id":     tid,
			"created_by_id": actor,
			"updated_by_id": actor,
		}).Error; err != nil {
			return err
		}
		tr, err := s.issueTokensTx(tx, u, m)
		if err != nil {
			return err
		}
		out = tr
		return nil
	})
	return out, err
}

// Login validates credentials and returns tenant choices plus a short-lived pick_tenant_token
// (including when the user has no tenants yet, for CreateTenantWithPickToken).
// Call TenantSession with that token and a chosen tenant_id to receive access and refresh tokens.
func (s *Service) Login(req LoginRequest) (*LoginDiscoveryResponse, error) {
	email := strings.ToLower(strings.TrimSpace(req.Email))
	u, err := s.repo.FindUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errorz.ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errorz.ErrInvalidCredentials
	}

	if !u.IsEmailVerified {
		deadline := effectiveVerificationDeadline(u, s.emailVerificationTTL())
		if deadline != nil && time.Now().UTC().After(*deadline) {
			if u.IsActive {
				_ = s.repo.Model(u).Update("is_active", false).Error
			}
			return nil, errorz.ErrVerificationExpired
		}
		return nil, errorz.ErrEmailNotVerified
	}

	if !u.IsActive {
		return nil, errorz.ErrInvalidCredentials
	}

	memberships, err := s.repo.ListMembershipsForUser(u.ID)
	if err != nil {
		return nil, err
	}
	choices := make([]TenantChoice, 0, len(memberships))
	for i := range memberships {
		m := &memberships[i]
		t, terr := s.repo.FindTenantByID(m.TenantID)
		if terr != nil {
			return nil, terr
		}
		if t == nil {
			continue
		}
		choices = append(choices, TenantChoice{
			TenantID: m.TenantID.String(),
			Name:     t.Name,
			Slug:     t.Slug,
			Role:     string(m.Role),
		})
	}

	out := &LoginDiscoveryResponse{
		User: UserResponse{
			UUID:      u.UUID.String(),
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Role:      string(u.Role),
		},
		Tenants: choices,
	}
	ttl := time.Duration(s.env.LoginPickTenantTTLMinutes) * time.Minute
	if s.env.LoginPickTenantTTLMinutes <= 0 {
		ttl = 10 * time.Minute
	}
	pick, err := jwtutil.SignTenantPick([]byte(s.env.JWTSecret), u.UUID.String(), int64(u.ID), ttl)
	if err != nil {
		return nil, err
	}
	out.PickTenantToken = pick
	out.PickTenantExpiresIn = int64(ttl.Seconds())

	if constants.IsPlatformStaff(u.Role) {
		pt, err := s.issuePlatformTokensOutsideTx(u)
		if err != nil {
			return nil, err
		}
		if pt != nil {
			out.PlatformAccessToken = &pt.AccessToken
			out.PlatformRefreshToken = &pt.RefreshToken
			exp := pt.ExpiresIn
			out.PlatformExpiresIn = &exp
		}
	}
	return out, nil
}

// TenantSession exchanges pick_tenant_token (Authorization: Bearer) + tenant_id for API tokens.
func (s *Service) TenantSession(pickToken string, tenantIDStr string) (*TokenResponse, error) {
	claims, err := jwtutil.ParseTenantPick([]byte(s.env.JWTSecret), pickToken)
	if err != nil {
		return nil, errorz.ErrInvalidPickToken
	}
	uid, err := types.ShouldParseUUID(claims.Subject)
	if err != nil {
		return nil, errorz.ErrInvalidPickToken
	}
	u, err := s.repo.FindUserByUUID(uid)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.IsActive || !u.IsEmailVerified || u.ID != uint(claims.UserDBID) {
		return nil, errorz.ErrInvalidPickToken
	}
	tid, err := types.ShouldParseUUID(tenantIDStr)
	if err != nil {
		return nil, err
	}
	m, err := s.repo.FindMembership(u.ID, tid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errorz.ErrMembershipNotFound
	}

	var out *TokenResponse
	err = s.repo.Transaction(func(tx *gorm.DB) error {
		tr, err := s.issueTokensTx(tx, u, m)
		if err != nil {
			return err
		}
		out = tr
		return nil
	})
	return out, err
}

// Refresh rotates refresh token and returns new pair.
func (s *Service) Refresh(refreshPlain string) (*TokenResponse, error) {
	sum := sha256.Sum256([]byte(refreshPlain))
	hash := hex.EncodeToString(sum[:])
	row, err := s.repo.FindValidRefreshByHash(hash)
	if err != nil {
		return nil, err
	}
	if row == nil {
		return nil, errorz.ErrInvalidRefreshToken
	}
	u, err := s.repo.FindUserByID(row.UserID)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.IsActive {
		return nil, errorz.ErrInvalidRefreshToken
	}

	if row.TenantID == nil {
		if !constants.IsPlatformStaff(u.Role) {
			return nil, errorz.ErrInvalidRefreshToken
		}
		var out *TokenResponse
		err = s.repo.Transaction(func(tx *gorm.DB) error {
			if err := s.repo.RevokeRefresh(tx, row.ID); err != nil {
				return err
			}
			tr, err := s.issuePlatformTokensTx(tx, u)
			if err != nil {
				return err
			}
			out = tr
			return nil
		})
		return out, err
	}

	tid := *row.TenantID
	m, err := s.repo.FindMembership(u.ID, tid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errorz.ErrInvalidRefreshToken
	}

	var out *TokenResponse
	err = s.repo.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.RevokeRefresh(tx, row.ID); err != nil {
			return err
		}
		tr, err := s.issueTokensTx(tx, u, m)
		if err != nil {
			return err
		}
		out = tr
		return nil
	})
	return out, err
}

// Logout revokes a refresh token (idempotent).
func (s *Service) Logout(refreshPlain string) error {
	sum := sha256.Sum256([]byte(refreshPlain))
	hash := hex.EncodeToString(sum[:])
	row, err := s.repo.FindValidRefreshByHash(hash)
	if err != nil {
		return err
	}
	if row == nil {
		return nil
	}
	return s.repo.RevokeRefresh(s.repo.DB, row.ID)
}

// ForgotPassword creates a reset token. Response is uniform for unknown emails; in local env the raw token is returned for testing.
func (s *Service) ForgotPassword(email string) (*ForgotPasswordResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	u, err := s.repo.FindUserByEmail(email)
	if err != nil {
		return nil, err
	}
	msg := "If an account exists for this email, password reset instructions have been issued."
	out := &ForgotPasswordResponse{Message: msg}
	if u == nil || !u.IsActive {
		return out, nil
	}

	ttl := time.Duration(s.env.PasswordResetTTLMinutes) * time.Minute
	if s.env.PasswordResetTTLMinutes <= 0 {
		ttl = 60 * time.Minute
	}
	plain, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(plain))
	hash := hex.EncodeToString(sum[:])
	err = s.repo.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.InvalidatePendingPasswordResets(tx, u.ID); err != nil {
			return err
		}
		pr := &models.PasswordResetToken{
			UserID:    u.ID,
			TokenHash: hash,
			ExpiresAt: time.Now().UTC().Add(ttl),
		}
		return s.repo.CreatePasswordResetToken(tx, pr)
	})
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(s.env.Environment, "local") {
		out.ResetToken = &plain
	}
	return out, nil
}

// ResetPassword validates a reset token and sets a new password; revokes refresh tokens.
func (s *Service) ResetPassword(token string, newPassword string) error {
	if len(newPassword) < 8 {
		return errorz.ErrWeakPassword
	}
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])
	row, err := s.repo.FindValidPasswordResetByHash(hash)
	if err != nil {
		return err
	}
	if row == nil {
		return errorz.ErrInvalidResetToken
	}
	u, err := s.repo.FindUserByID(row.UserID)
	if err != nil {
		return err
	}
	if u == nil || !u.IsActive {
		return errorz.ErrInvalidResetToken
	}
	hashPass, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	actor := u.ID
	return s.repo.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(u).Updates(map[string]any{
			"password_hash": string(hashPass),
			"updated_by_id": actor,
		}).Error; err != nil {
			return err
		}
		if err := s.repo.MarkPasswordResetUsed(tx, row.ID); err != nil {
			return err
		}
		return s.repo.RevokeAllRefreshTokensForUser(tx, u.ID)
	})
}

// ChangePassword verifies the current password and sets a new one; revokes refresh tokens.
func (s *Service) ChangePassword(userID uint, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return errorz.ErrWeakPassword
	}
	u, err := s.repo.FindUserByID(userID)
	if err != nil {
		return err
	}
	if u == nil {
		return errorz.ErrRecordNotFound
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(currentPassword)); err != nil {
		return errorz.ErrWrongCurrentPassword
	}
	hashPass, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return s.repo.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(u).Updates(map[string]any{
			"password_hash": string(hashPass),
			"updated_by_id": userID,
		}).Error; err != nil {
			return err
		}
		return s.repo.RevokeAllRefreshTokensForUser(tx, userID)
	})
}

// UpdateMe patches profile fields for the authenticated user in the current tenant context.
func (s *Service) UpdateMe(userUUIDStr, tenantIDStr string, actorID uint, req UpdateMeRequest) (*MeResponse, error) {
	updates := map[string]any{}
	if req.FirstName != nil {
		updates["first_name"] = *req.FirstName
	}
	if req.LastName != nil {
		updates["last_name"] = *req.LastName
	}
	if req.FirstNameJa != nil {
		updates["first_name_ja"] = *req.FirstNameJa
	}
	if req.LastNameJa != nil {
		updates["last_name_ja"] = *req.LastNameJa
	}
	if len(updates) == 0 {
		return s.Me(userUUIDStr, tenantIDStr)
	}
	updates["updated_by_id"] = actorID

	uid, err := types.ShouldParseUUID(userUUIDStr)
	if err != nil {
		return nil, err
	}
	tid, err := types.ShouldParseUUID(tenantIDStr)
	if err != nil {
		return nil, err
	}
	u, err := s.repo.FindUserByUUID(uid)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errorz.ErrRecordNotFound
	}
	m, err := s.repo.FindMembership(u.ID, tid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errorz.ErrMembershipNotFound
	}
	if err := s.repo.Model(u).Updates(updates).Error; err != nil {
		return nil, err
	}
	return s.Me(userUUIDStr, tenantIDStr)
}

// Me returns the current user and tenant context.
func (s *Service) Me(userUUIDStr string, tenantIDStr string) (*MeResponse, error) {
	uid, err := types.ShouldParseUUID(userUUIDStr)
	if err != nil {
		return nil, err
	}
	tid, err := types.ShouldParseUUID(tenantIDStr)
	if err != nil {
		return nil, err
	}
	u, err := s.repo.FindUserByUUID(uid)
	if err != nil {
		return nil, err
	}
	if u == nil {
		return nil, errorz.ErrRecordNotFound
	}
	m, err := s.repo.FindMembership(u.ID, tid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errorz.ErrMembershipNotFound
	}
	permKeys, err := s.perms.EffectivePermissionKeys(m)
	if err != nil {
		return nil, err
	}
	return &MeResponse{
		User: UserResponse{
			UUID:      u.UUID.String(),
			Email:     u.Email,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Role:      string(u.Role),
		},
		TenantID:      tid.String(),
		TenantRole:    string(m.Role),
		Permissions:   permKeys,
		IsActive:      u.IsActive,
		EmailVerified: u.IsEmailVerified,
	}, nil
}

func (s *Service) issueTokensTx(tx *gorm.DB, u *models.User, m *models.TenantMembership) (*TokenResponse, error) {
	accessTTL := time.Duration(s.env.JWTAccessTTLMinutes) * time.Minute
	refreshTTL := time.Duration(s.env.JWTRefreshTTLDays) * 24 * time.Hour

	permKeys, err := s.perms.EffectivePermissionKeys(m)
	if err != nil {
		return nil, err
	}
	trid := ""
	if m.TenantRoleID != nil {
		trid = m.TenantRoleID.String()
	}
	access, err := jwtutil.SignAccess([]byte(s.env.JWTSecret), u.UUID.String(), int64(u.ID), m.TenantID.String(), string(m.Role), trid, permKeys, accessTTL)
	if err != nil {
		return nil, err
	}
	plain, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(plain))
	hash := hex.EncodeToString(sum[:])
	actor := u.ID
	tidCopy := m.TenantID
	rt := &models.RefreshToken{
		UserID:    u.ID,
		TenantID:  &tidCopy,
		TokenHash: hash,
		ExpiresAt: time.Now().UTC().Add(refreshTTL),
		AuditFields: models.AuditFields{
			CreatedByID: &actor,
			UpdatedByID: &actor,
		},
	}
	if err := s.repo.CreateRefreshToken(tx, rt); err != nil {
		return nil, err
	}
	return &TokenResponse{
		AccessToken:  access,
		RefreshToken: plain,
		TokenType:    "Bearer",
		ExpiresIn:    int64(accessTTL.Seconds()),
	}, nil
}

func (s *Service) issuePlatformTokensTx(tx *gorm.DB, u *models.User) (*TokenResponse, error) {
	accessTTL := time.Duration(s.env.JWTAccessTTLMinutes) * time.Minute
	refreshTTL := time.Duration(s.env.JWTRefreshTTLDays) * 24 * time.Hour
	access, err := jwtutil.SignPlatformAccess([]byte(s.env.JWTSecret), u.UUID.String(), int64(u.ID), string(u.Role), accessTTL)
	if err != nil {
		return nil, err
	}
	plain, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(plain))
	hash := hex.EncodeToString(sum[:])
	actor := u.ID
	rt := &models.RefreshToken{
		UserID:    u.ID,
		TenantID:  nil,
		TokenHash: hash,
		ExpiresAt: time.Now().UTC().Add(refreshTTL),
		AuditFields: models.AuditFields{
			CreatedByID: &actor,
			UpdatedByID: &actor,
		},
	}
	if err := s.repo.CreateRefreshToken(tx, rt); err != nil {
		return nil, err
	}
	return &TokenResponse{
		AccessToken:  access,
		RefreshToken: plain,
		TokenType:    "Bearer",
		ExpiresIn:    int64(accessTTL.Seconds()),
	}, nil
}

func (s *Service) issuePlatformTokensOutsideTx(u *models.User) (*TokenResponse, error) {
	var out *TokenResponse
	err := s.repo.Transaction(func(tx *gorm.DB) error {
		tr, err := s.issuePlatformTokensTx(tx, u)
		if err != nil {
			return err
		}
		out = tr
		return nil
	})
	return out, err
}

// Signup creates an unverified user without a tenant and emails (or returns) a verification token.
func (s *Service) Signup(req SignupRequest) (*SignupResponse, error) {
	if len(req.Password) < 8 {
		return nil, errorz.ErrWeakPassword
	}
	email := strings.ToLower(strings.TrimSpace(req.Email))
	existing, err := s.repo.FindUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, errorz.ErrEmailTaken
	}
	hashPass, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	ttl := s.emailVerificationTTL()
	deadline := time.Now().UTC().Add(ttl)
	var out *SignupResponse
	err = s.repo.Transaction(func(tx *gorm.DB) error {
		u := &models.User{
			Email:                     email,
			PasswordHash:              string(hashPass),
			IsActive:                  true,
			IsEmailVerified:           false,
			EmailVerificationDeadline: &deadline,
		}
		if err := tx.Create(u).Error; err != nil {
			return err
		}
		actor := u.ID
		if err := tx.Model(u).Updates(map[string]any{
			"created_by_id": actor,
			"updated_by_id": actor,
		}).Error; err != nil {
			return err
		}
		plain, err := randomToken(32)
		if err != nil {
			return err
		}
		sum := sha256.Sum256([]byte(plain))
		hash := hex.EncodeToString(sum[:])
		if err := s.repo.InvalidatePendingEmailVerifications(tx, u.ID); err != nil {
			return err
		}
		ev := &models.EmailVerificationToken{
			UserID:    u.ID,
			TokenHash: hash,
			ExpiresAt: time.Now().UTC().Add(ttl),
		}
		if err := s.repo.CreateEmailVerificationToken(tx, ev); err != nil {
			return err
		}
		out = &SignupResponse{
			Message: "If this email is new, check your inbox to verify your account.",
		}
		if strings.EqualFold(s.env.Environment, "local") {
			out.VerificationToken = &plain
		}
		return nil
	})
	return out, err
}

// VerifyEmail marks the user verified using a one-time token from the verification email.
func (s *Service) VerifyEmail(token string) error {
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])
	row, err := s.repo.FindValidEmailVerificationByHash(hash)
	if err != nil {
		return err
	}
	if row == nil {
		return errorz.ErrInvalidVerificationToken
	}
	u, err := s.repo.FindUserByID(row.UserID)
	if err != nil {
		return err
	}
	if u == nil {
		return errorz.ErrInvalidVerificationToken
	}
	actor := u.ID
	return s.repo.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.InvalidatePendingEmailVerifications(tx, u.ID); err != nil {
			return err
		}
		return tx.Model(u).Updates(map[string]any{
			"is_email_verified":           true,
			"is_active":                   true,
			"email_verification_deadline": nil,
			"updated_by_id":               actor,
		}).Error
	})
}

// ResendVerification issues a new verification link for an unverified account (rate-limited).
func (s *Service) ResendVerification(email, clientIP string) (*ResendVerificationResponse, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	key := email + "||" + strings.TrimSpace(clientIP)
	if !s.recordResend(key) {
		return nil, errorz.ErrResendVerificationLimited
	}
	msg := "If an account exists for this email and it is still unverified, a new verification link has been issued."
	out := &ResendVerificationResponse{Message: msg}
	u, err := s.repo.FindUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if u == nil || u.IsEmailVerified {
		return out, nil
	}
	ttl := s.emailVerificationTTL()
	deadline := time.Now().UTC().Add(ttl)
	plain, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	tokHash := sha256.Sum256([]byte(plain))
	hash := hex.EncodeToString(tokHash[:])
	actor := u.ID
	err = s.repo.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.InvalidatePendingEmailVerifications(tx, u.ID); err != nil {
			return err
		}
		ev := &models.EmailVerificationToken{
			UserID:    u.ID,
			TokenHash: hash,
			ExpiresAt: time.Now().UTC().Add(ttl),
		}
		if err := s.repo.CreateEmailVerificationToken(tx, ev); err != nil {
			return err
		}
		return tx.Model(u).Updates(map[string]any{
			"is_active":                   true,
			"email_verification_deadline": deadline,
			"updated_by_id":               actor,
		}).Error
	})
	if err != nil {
		return nil, err
	}
	if strings.EqualFold(s.env.Environment, "local") {
		out.VerificationToken = &plain
	}
	return out, nil
}

// CreateTenantWithPickToken creates the first organization for a verified user with no memberships.
func (s *Service) CreateTenantWithPickToken(pickToken, tenantName string) (*TokenResponse, error) {
	claims, err := jwtutil.ParseTenantPick([]byte(s.env.JWTSecret), pickToken)
	if err != nil {
		return nil, errorz.ErrInvalidPickToken
	}
	uid, err := types.ShouldParseUUID(claims.Subject)
	if err != nil {
		return nil, errorz.ErrInvalidPickToken
	}
	u, err := s.repo.FindUserByUUID(uid)
	if err != nil {
		return nil, err
	}
	if u == nil || !u.IsActive || u.ID != uint(claims.UserDBID) || !u.IsEmailVerified {
		return nil, errorz.ErrInvalidPickToken
	}
	memberships, err := s.repo.ListMembershipsForUser(u.ID)
	if err != nil {
		return nil, err
	}
	if len(memberships) > 0 {
		return nil, errorz.ErrForbiddenAccess
	}
	name := strings.TrimSpace(tenantName)
	if name == "" {
		return nil, errorz.ErrBadRequest.JoinError("tenant_name is required")
	}
	var out *TokenResponse
	err = s.repo.Transaction(func(tx *gorm.DB) error {
		slug, err := allocateSlugTx(tx, name)
		if err != nil {
			return err
		}
		actor := u.ID
		t := &models.Tenant{
			Name: name,
			Slug: slug,
			AuditFields: models.AuditFields{
				CreatedByID: &actor,
				UpdatedByID: &actor,
			},
		}
		if err := tx.Create(t).Error; err != nil {
			return err
		}
		m := &models.TenantMembership{
			UserID:   u.ID,
			TenantID: t.ID,
			Role:     constants.TenantRoleOwner,
			AuditFields: models.AuditFields{
				CreatedByID: &actor,
				UpdatedByID: &actor,
			},
		}
		if err := tx.Create(m).Error; err != nil {
			return err
		}
		tid := t.ID
		if err := tx.Model(u).Updates(map[string]any{
			"tenant_id":     tid,
			"updated_by_id": actor,
		}).Error; err != nil {
			return err
		}
		tr, err := s.issueTokensTx(tx, u, m)
		if err != nil {
			return err
		}
		out = tr
		return nil
	})
	return out, err
}

// CreateTenantInvitation adds a pending invite; caller must be owner or admin of the tenant.
func (s *Service) CreateTenantInvitation(actorID uint, tenantIDStr string, req CreateTenantInviteRequest) (*TenantInviteResponse, error) {
	roleStr := req.Role
	inviteEmail := req.Email
	role, err := parseTenantRole(roleStr)
	if err != nil {
		return nil, err
	}
	tid, err := types.ShouldParseUUID(tenantIDStr)
	if err != nil {
		return nil, err
	}
	t, err := s.repo.FindTenantByID(tid)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errorz.ErrTenantNotFound
	}
	m, err := s.repo.FindMembership(actorID, tid)
	if err != nil {
		return nil, err
	}
	if m == nil {
		return nil, errorz.ErrMembershipNotFound
	}
	if m.Role != constants.TenantRoleOwner && m.Role != constants.TenantRoleAdmin {
		return nil, errorz.ErrTenantInviteForbidden
	}
	email := strings.ToLower(strings.TrimSpace(inviteEmail))
	if email == "" {
		return nil, errorz.ErrBadRequest.JoinError("email is required")
	}
	existingUser, err := s.repo.FindUserByEmail(email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		existingM, err := s.repo.FindMembership(existingUser.ID, tid)
		if err != nil {
			return nil, err
		}
		if existingM != nil {
			return nil, errorz.ErrInviteAlreadyMember
		}
	}
	var customRoleID *types.BinaryUUID
	if req.TenantRoleID != nil && strings.TrimSpace(*req.TenantRoleID) != "" {
		rid, err := types.ShouldParseUUID(strings.TrimSpace(*req.TenantRoleID))
		if err != nil {
			return nil, err
		}
		tr, err := s.repo.FindTenantRoleByID(rid)
		if err != nil {
			return nil, err
		}
		if tr == nil || tr.TenantID != tid {
			return nil, errorz.ErrBadRequest.JoinError("tenant_role_id is not valid for this tenant")
		}
		customRoleID = &rid
	}
	ttl := s.tenantInviteTTL()
	plain, err := randomToken(32)
	if err != nil {
		return nil, err
	}
	tokHash := sha256.Sum256([]byte(plain))
	hash := hex.EncodeToString(tokHash[:])
	var out *TenantInviteResponse
	err = s.repo.Transaction(func(tx *gorm.DB) error {
		if err := s.repo.RevokePendingTenantInvitations(tx, tid, email); err != nil {
			return err
		}
		inv := &models.TenantInvitation{
			TenantID:        tid,
			Email:           email,
			Role:            role,
			TenantRoleID:    customRoleID,
			TokenHash:       hash,
			InvitedByUserID: actorID,
			ExpiresAt:       time.Now().UTC().Add(ttl),
		}
		if err := s.repo.CreateTenantInvitation(tx, inv); err != nil {
			return err
		}
		out = &TenantInviteResponse{Message: "Invitation sent."}
		if strings.EqualFold(s.env.Environment, "local") {
			out.InviteToken = &plain
		}
		return nil
	})
	return out, err
}

// AcceptInvite creates or authenticates a user and attaches the invited membership.
func (s *Service) AcceptInvite(token, password string) (*TokenResponse, error) {
	if len(password) < 8 {
		return nil, errorz.ErrWeakPassword
	}
	sum := sha256.Sum256([]byte(token))
	hash := hex.EncodeToString(sum[:])
	inv, err := s.repo.FindValidTenantInvitationByHash(hash)
	if err != nil {
		return nil, err
	}
	if inv == nil {
		return nil, errorz.ErrInvalidInviteToken
	}
	t, err := s.repo.FindTenantByID(inv.TenantID)
	if err != nil {
		return nil, err
	}
	if t == nil {
		return nil, errorz.ErrInvalidInviteToken
	}
	email := inv.Email
	u, err := s.repo.FindUserByEmail(email)
	if err != nil {
		return nil, err
	}
	var out *TokenResponse
	err = s.repo.Transaction(func(tx *gorm.DB) error {
		if u == nil {
			hashPass, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
			if err != nil {
				return err
			}
			newUser := &models.User{
				Email:           email,
				PasswordHash:    string(hashPass),
				IsActive:        true,
				IsEmailVerified: true,
			}
			if err := tx.Create(newUser).Error; err != nil {
				return err
			}
			actor := newUser.ID
			if err := tx.Model(newUser).Updates(map[string]any{
				"tenant_id":     inv.TenantID,
				"created_by_id": actor,
				"updated_by_id": actor,
			}).Error; err != nil {
				return err
			}
			u = newUser
		} else {
			if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
				return errorz.ErrInvalidCredentials
			}
			var dup models.TenantMembership
			err := tx.Where("user_id = ? AND tenant_id = ?", u.ID, inv.TenantID).First(&dup).Error
			if err == nil {
				return errorz.ErrInviteAlreadyMember
			}
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				return err
			}
			actor := u.ID
			if err := tx.Model(u).Updates(map[string]any{
				"is_email_verified":           true,
				"is_active":                   true,
				"email_verification_deadline": nil,
				"tenant_id":                   inv.TenantID,
				"updated_by_id":               actor,
			}).Error; err != nil {
				return err
			}
		}
		member := &models.TenantMembership{
			UserID:       u.ID,
			TenantID:     inv.TenantID,
			Role:         inv.Role,
			TenantRoleID: inv.TenantRoleID,
			AuditFields: models.AuditFields{
				CreatedByID: &u.ID,
				UpdatedByID: &u.ID,
			},
		}
		if err := tx.Create(member).Error; err != nil {
			return err
		}
		if err := s.repo.MarkTenantInvitationAccepted(tx, inv.ID); err != nil {
			return err
		}
		tr, err := s.issueTokensTx(tx, u, member)
		if err != nil {
			return err
		}
		out = tr
		return nil
	})
	return out, err
}

func (s *Service) emailVerificationTTL() time.Duration {
	if s.env.EmailVerificationTTLMinutes <= 0 {
		return 15 * time.Minute
	}
	return time.Duration(s.env.EmailVerificationTTLMinutes) * time.Minute
}

func (s *Service) tenantInviteTTL() time.Duration {
	if s.env.TenantInviteTTLMinutes <= 0 {
		return 7 * 24 * time.Hour
	}
	return time.Duration(s.env.TenantInviteTTLMinutes) * time.Minute
}

func effectiveVerificationDeadline(u *models.User, ttl time.Duration) *time.Time {
	if u.IsEmailVerified {
		return nil
	}
	if u.EmailVerificationDeadline != nil {
		t := u.EmailVerificationDeadline.UTC()
		return &t
	}
	if u.CreatedAt.IsZero() {
		return nil
	}
	d := u.CreatedAt.UTC().Add(ttl)
	return &d
}

func parseTenantRole(s string) (constants.TenantRole, error) {
	r := constants.TenantRole(strings.ToLower(strings.TrimSpace(s)))
	switch r {
	case constants.TenantRoleOwner, constants.TenantRoleAdmin, constants.TenantRoleMember:
		return r, nil
	default:
		return "", errorz.ErrInvalidTenantRole
	}
}

func (s *Service) recordResend(key string) bool {
	const maxPerWindow = 3
	const window = time.Hour
	s.resendMu.Lock()
	defer s.resendMu.Unlock()
	if s.resendHits == nil {
		s.resendHits = make(map[string][]time.Time)
	}
	now := time.Now()
	var kept []time.Time
	for _, ts := range s.resendHits[key] {
		if now.Sub(ts) < window {
			kept = append(kept, ts)
		}
	}
	if int64(len(kept)) >= int64(maxPerWindow) {
		return false
	}
	kept = append(kept, now)
	s.resendHits[key] = kept
	return true
}

func allocateSlugTx(tx *gorm.DB, name string) (string, error) {
	base := slugify(name)
	for i := 0; i < 100; i++ {
		candidate := base
		if i > 0 {
			candidate = fmt.Sprintf("%s-%d", base, i)
		}
		var count int64
		if err := tx.Model(&models.Tenant{}).Where("slug = ?", candidate).Count(&count).Error; err != nil {
			return "", err
		}
		if count == 0 {
			return candidate, nil
		}
	}
	return "", errors.New("could not allocate unique tenant slug")
}

func slugify(s string) string {
	var b strings.Builder
	s = strings.ToLower(strings.TrimSpace(s))
	prevDash := false
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		case r == ' ' || r == '-' || r == '_':
			if !prevDash && b.Len() > 0 {
				b.WriteRune('-')
				prevDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "tenant"
	}
	return out
}

func randomToken(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
