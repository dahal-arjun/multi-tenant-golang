package framework

import (
	"github.com/spf13/viper"
)

type Env struct {
	LogLevel    string `mapstructure:"LOG_LEVEL"`
	ServerPort  string `mapstructure:"SERVER_PORT"`
	Environment string `mapstructure:"ENVIRONMENT"`

	DBUsername string `mapstructure:"DB_USER"`
	DBPassword string `mapstructure:"DB_PASS"`
	DBHost     string `mapstructure:"DB_HOST"`
	DBPort     string `mapstructure:"DB_PORT"`
	DBName string `mapstructure:"DB_NAME"`

	JWTSecret           string `mapstructure:"JWT_SECRET"`
	JWTAccessTTLMinutes int    `mapstructure:"JWT_ACCESS_TTL_MINUTES"`
	JWTRefreshTTLDays   int    `mapstructure:"JWT_REFRESH_TTL_DAYS"`

	// PasswordResetTTLMinutes is how long forgot-password tokens remain valid.
	PasswordResetTTLMinutes int `mapstructure:"PASSWORD_RESET_TTL_MINUTES"`

	// LoginPickTenantTTLMinutes is how long the post-login tenant-picker JWT remains valid.
	LoginPickTenantTTLMinutes int `mapstructure:"LOGIN_PICK_TENANT_TTL_MINUTES"`

	// EmailVerificationTTLMinutes is how long verification links and the initial login grace window last.
	EmailVerificationTTLMinutes int `mapstructure:"EMAIL_VERIFICATION_TTL_MINUTES"`
	// PublicAppURL is the browser-facing origin for verification and invite links (no trailing slash required).
	PublicAppURL string `mapstructure:"PUBLIC_APP_URL"`
	// EmailVerificationPath is appended to PublicAppURL for verification deep links (e.g. /verify).
	EmailVerificationPath string `mapstructure:"EMAIL_VERIFICATION_PATH"`
	// TenantInvitePath is appended to PublicAppURL for invite acceptance deep links (e.g. /accept-invite).
	TenantInvitePath string `mapstructure:"TENANT_INVITE_PATH"`
	// TenantInviteTTLMinutes is how long tenant invitation tokens remain valid.
	TenantInviteTTLMinutes int `mapstructure:"TENANT_INVITE_TTL_MINUTES"`

	SentryDSN          string `mapstructure:"SENTRY_DSN"`
	MaxMultipartMemory int64  `mapstructure:"MAX_MULTIPART_MEMORY"`
	StorageBucketName  string `mapstructure:"STORAGE_BUCKET_NAME"`

	TimeZone      string `mapstructure:"TIMEZONE"`
	AdminEmail    string `mapstructure:"ADMIN_EMAIL"`
	AdminPassword string `mapstructure:"ADMIN_PASSWORD"`

}

var globalEnv = Env{
	MaxMultipartMemory: 10 << 20, // 10 MB
}

func GetEnv() Env {
	return globalEnv
}

func NewEnv(logger Logger) *Env {
	viper.AutomaticEnv()
	viper.SetConfigFile(".env")

	err := viper.ReadInConfig()
	if err != nil {
		logger.Fatal("cannot read cofiguration", err)
	}

	viper.SetDefault("TIMEZONE", "UTC")
	viper.SetDefault("JWT_ACCESS_TTL_MINUTES", 15)
	viper.SetDefault("JWT_REFRESH_TTL_DAYS", 30)
	viper.SetDefault("PASSWORD_RESET_TTL_MINUTES", 60)
	viper.SetDefault("LOGIN_PICK_TENANT_TTL_MINUTES", 10)
	viper.SetDefault("EMAIL_VERIFICATION_TTL_MINUTES", 15)
	viper.SetDefault("PUBLIC_APP_URL", "http://localhost:3000")
	viper.SetDefault("EMAIL_VERIFICATION_PATH", "/verify")
	viper.SetDefault("TENANT_INVITE_PATH", "/accept-invite")
	viper.SetDefault("TENANT_INVITE_TTL_MINUTES", 10080)

	err = viper.Unmarshal(&globalEnv)
	if err != nil {
		logger.Fatal("environment cant be loaded: ", err)
	}

	if globalEnv.JWTSecret == "" {
		if globalEnv.Environment == "production" {
			logger.Fatal("JWT_SECRET is required in production")
		}
		globalEnv.JWTSecret = "dev-insecure-jwt-secret-change-me"
		logger.Warn("JWT_SECRET not set; using insecure default for non-production")
	}

	return &globalEnv
}
