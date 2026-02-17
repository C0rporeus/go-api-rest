package constants

import (
	"os"
	"strconv"
	"time"
)

// Visibility values for experiences/skills.
const (
	VisibilityPublic  = "public"
	VisibilityPrivate = "private"
)

// DynamoDB defaults.
const (
	DefaultDynamoDBTable = "users"
	DynamoDBEmailIndex   = "email-index"
)

// DNS defaults.
const DefaultDNSTimeout = 5 * time.Second

// DNSBLProviders is the list of DNS blacklist providers used for IP checks.
var DNSBLProviders = []string{
	"zen.spamhaus.org",
	"bl.spamcop.net",
	"b.barracudacentral.org",
	"dnsbl.sorbs.net",
	"spam.dnsbl.sorbs.net",
	"cbl.abuseat.org",
}

// Cache-Control header for public collection responses.
const PublicCollectionCacheControl = "public, max-age=60, stale-while-revalidate=300"

// Certificate generation defaults.
const (
	DefaultCertCommonName   = "localhost"
	DefaultCertOrganization = "PortfolioTools"
	DefaultCertValidDays    = 365
	DefaultCertPassword     = "changeit"
	DefaultCertKeyBits      = 2048
)

// Skill tags used to identify skill-type experiences.
var SkillTags = []string{
	"skill", "skills",
	"habilidad", "habilidades",
	"capacidad", "capacidades",
}

// File permissions.
const (
	DirPermission  = 0o755
	FilePermission = 0o644
)

// API route paths.
const APIContactPath = "/api/contact"

// Data storage defaults.
const (
	DefaultDataDir      = "data"
	ExperiencesFilename = "experiences.json"
	DataDirEnvVar       = "PORTFOLIO_DATA_DIR"
)

// Input validation limits.
const (
	MaxTitleLength     = 200
	MaxSummaryLength   = 500
	MaxBodyLength      = 50000
	MaxTagLength       = 50
	MaxTagCount        = 20
	MaxImageURLCount   = 10
	MaxBase64InputSize = 1_000_000
	MaxCertCommonName  = 100
	MaxCertOrgLength   = 100
	MaxCertValidDays   = 3650
	MinPasswordLength  = 8
	MaxPasswordLength  = 128
	MaxEmailLength     = 254
	MaxMessageLength   = 500
	MaxContactName     = 100
	MinContactName     = 2
)

// Rate limiting defaults.
const (
	RateLimitAuthMax      = 10
	RateLimitAuthWindow   = 60
	RateLimitToolsMax     = 30
	RateLimitToolsWindow  = 60
	RateLimitGlobalMax    = 100
	RateLimitGlobalWindow = 60
)

// Request body size limits (in bytes).
const (
	BodyLimitDefault = 1 * 1024 * 1024
	BodyLimitCerts   = 512 * 1024
)

// DefaultJWTExpiryHours is the fallback JWT token expiry in hours.
const DefaultJWTExpiryHours = 24

// JWTExpiryDuration reads JWT_EXPIRY_HOURS from env with a fallback.
func JWTExpiryDuration() time.Duration {
	if val := os.Getenv("JWT_EXPIRY_HOURS"); val != "" {
		if hours, err := strconv.Atoi(val); err == nil && hours > 0 {
			return time.Duration(hours) * time.Hour
		}
	}
	return DefaultJWTExpiryHours * time.Hour
}

// TableName returns the DynamoDB table name from env or the default.
func TableName() string {
	if name := os.Getenv("DYNAMO_DB_TABLE"); name != "" {
		return name
	}
	return DefaultDynamoDBTable
}
