package auth

import (
	"clean-architecture/domain/constants"
	"clean-architecture/domain/models"
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
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Service handles registration, login, and token lifecycle.
type Service struct {
	repo   Repository
	env    *framework.Env
	logger framework.Logger
}

// NewService constructs the auth service.
func NewService(repo Repository, env *framework.Env, logger framework.Logger) *Service {
	return &Service{repo: repo, env: env, logger: logger}
}

// Register creates user, tenant, owner membership, and returns tokens.
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
			IsEmailVerified: false,
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
		tr, err := s.issueTokensTx(tx, u, t.ID, string(constants.TenantRoleOwner))
		if err != nil {
			return err
		}
		out = tr
		return nil
	})
	return out, err
}

// Login validates credentials and returns tenant choices plus a short-lived pick_tenant_token.
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
	if len(choices) == 0 {
		return out, nil
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
	if u == nil || !u.IsActive || u.ID != uint(claims.UserDBID) {
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
		tr, err := s.issueTokensTx(tx, u, tid, string(m.Role))
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
	m, err := s.repo.FindMembership(u.ID, row.TenantID)
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
		tr, err := s.issueTokensTx(tx, u, row.TenantID, string(m.Role))
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
		IsActive:      u.IsActive,
		EmailVerified: u.IsEmailVerified,
	}, nil
}

func (s *Service) issueTokensTx(tx *gorm.DB, u *models.User, tenantID types.BinaryUUID, tenantRole string) (*TokenResponse, error) {
	accessTTL := time.Duration(s.env.JWTAccessTTLMinutes) * time.Minute
	refreshTTL := time.Duration(s.env.JWTRefreshTTLDays) * 24 * time.Hour

	access, err := jwtutil.SignAccess([]byte(s.env.JWTSecret), u.UUID.String(), int64(u.ID), tenantID.String(), tenantRole, accessTTL)
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
		TenantID:  tenantID,
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
