package constants

import (
	"sort"
	"strings"
)

// Permission is a curated capability string (registry in code).
type Permission string

const (
	PermissionWidgetsRead  Permission = "widgets.read"
	PermissionWidgetsWrite Permission = "widgets.write"
	PermissionRolesManage  Permission = "roles.manage"
	PermissionInvitesManage Permission = "invites.manage"
	PermissionMembersManage Permission = "members.manage"
	PermissionPlatformRead  Permission = "platform.read"
)

// AllPermissions returns every defined permission (used for owner / superuser baseline).
func AllPermissions() []Permission {
	return []Permission{
		PermissionWidgetsRead,
		PermissionWidgetsWrite,
		PermissionRolesManage,
		PermissionInvitesManage,
		PermissionMembersManage,
		PermissionPlatformRead,
	}
}

// BuiltinPermissionKeys returns permissions implied by legacy owner/admin/member roles.
func BuiltinPermissionKeys(role TenantRole) []string {
	switch role {
	case TenantRoleOwner:
		return PermissionStrings(AllPermissions())
	case TenantRoleAdmin:
		return PermissionStrings([]Permission{
			PermissionWidgetsRead,
			PermissionWidgetsWrite,
			PermissionRolesManage,
			PermissionInvitesManage,
			PermissionMembersManage,
		})
	case TenantRoleMember:
		return PermissionStrings([]Permission{PermissionWidgetsRead})
	default:
		return nil
	}
}

// PermissionStrings converts to string slice.
func PermissionStrings(p []Permission) []string {
	out := make([]string, len(p))
	for i := range p {
		out[i] = string(p[i])
	}
	return out
}

// IsValidPermission returns true if key is in the curated registry.
func IsValidPermission(key string) bool {
	key = strings.TrimSpace(key)
	for _, p := range AllPermissions() {
		if string(p) == key {
			return true
		}
	}
	return false
}

// MergePermissionKeys returns sorted unique keys (builtin ∪ custom).
func MergePermissionKeys(builtin, custom []string) []string {
	seen := make(map[string]struct{})
	for _, k := range builtin {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		seen[k] = struct{}{}
	}
	for _, k := range custom {
		k = strings.TrimSpace(k)
		if k == "" || !IsValidPermission(k) {
			continue
		}
		seen[k] = struct{}{}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
