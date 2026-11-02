package constants

import (
	"os"
	"strconv"
	"strings"
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

// PublicMediaCacheControl caches immutable uploaded images (object names are UUIDs).
const PublicMediaCacheControl = "public, max-age=604800, immutable"

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
	UsersFilename       = "users.json"
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
	MaxImageURLLength  = 2048 // GCS signed URLs can be long; keep below browser/server limits
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
	RateLimitAuthMax      = 5
	RateLimitAuthWindow   = 60
	RateLimitToolsMax     = 30
	RateLimitToolsWindow  = 60
	RateLimitGlobalMax    = 100
	RateLimitGlobalWindow = 60
)

// Request body size limits (in bytes).
// BodyLimitDefault is set to MaxImageUploadBytes + 1 MB multipart overhead
// so that POST /api/private/upload-image can reach the handler's own validation.
const (
	BodyLimitDefault = (MaxImageUploadBytes + 1*1024*1024) // 6 MB
	BodyLimitCerts   = 512 * 1024
)

// Image upload (GCP Storage) limits.
const (
	MaxImageUploadBytes = 5 * 1024 * 1024 // 5 MB
	StorageImagePrefix  = "portfolio-images"
)

// Allowed image MIME types for upload.
var AllowedImageContentTypes = []string{"image/jpeg", "image/png", "image/gif", "image/webp"}

// Default signed URL expiry for reading images.
const DefaultSignedURLExpiryHours = 168 // 7 days

// SignedURLExpiry returns the duration for signed image URLs.
func SignedURLExpiry() time.Duration {
	if val := os.Getenv("SIGNED_URL_EXPIRY_HOURS"); val != "" {
		if hours, err := strconv.Atoi(val); err == nil && hours > 0 {
			return time.Duration(hours) * time.Hour
		}
	}
	return DefaultSignedURLExpiryHours * time.Hour
}

// GCSURLPrefix returns the canonical prefix for our bucket URLs.
func GCSURLPrefix() string {
	bucket := GCSBucketName()
	if bucket == "" {
		return ""
	}
	return "https://storage.googleapis.com/" + bucket + "/"
}

// Server-rendered blog pages (GET /blog/:id, reached through a Firebase Hosting
// rewrite). The page is the static Next.js shell plus per-article meta tags.
const (
	DefaultSiteURL       = "https://www.yonathangutierrez.dev"
	DefaultPublicAPIURL  = "https://api.yonathangutierrez.dev"
	BlogShellPath        = "/blog/articulo/"
	BlogShellTTL         = 60 * time.Second
	BlogPageCacheControl = "public, max-age=0, s-maxage=60"
	BlogSiteName         = "Yonathan Gutierrez"
)

// SiteURL is the canonical public site (SITE_URL overrides it).
func SiteURL() string { return envOrDefault("SITE_URL", DefaultSiteURL) }

// PublicAPIURL is the public base URL of this API, used for og:image links
// (PUBLIC_API_URL overrides it).
func PublicAPIURL() string { return envOrDefault("PUBLIC_API_URL", DefaultPublicAPIURL) }

// BlogShellURL is where the static article shell is fetched from
// (BLOG_SHELL_URL overrides it, e.g. a local Next.js export).
func BlogShellURL() string {
	if v := strings.TrimSpace(os.Getenv("BLOG_SHELL_URL")); v != "" {
		return v
	}
	return SiteURL() + BlogShellPath
}

// envOrDefault returns the trimmed env value without a trailing slash, or fallback.
func envOrDefault(key, fallback string) string {
	if v := strings.TrimRight(strings.TrimSpace(os.Getenv(key)), "/"); v != "" {
		return v
	}
	return fallback
}

// Firestore defaults.
const (
	FirestoreUsersCollection       = "users"
	FirestoreExperiencesCollection = "experiences"
)

// RegistrationEnabled returns whether public user registration is allowed.
// Defaults to false (disabled) in production for security.
func RegistrationEnabled() bool {
	val := strings.ToLower(os.Getenv("REGISTRATION_ENABLED"))
	return val == "true" || val == "1"
}

// GitHub Actions workflow dispatch — used by the redeploy endpoint.
const (
	GitHubOwner    = "C0rporeus"
	GitHubRepo     = "front-end"
	GitHubWorkflow = "ci.yml"
	GitHubRef      = "main"
	GitHubPATEnv   = "GITHUB_DISPATCH_PAT"
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

// GCSBucketName returns the GCP Storage bucket name for image uploads from env.
// If empty, the upload endpoint will return 503 (service not configured).
func GCSBucketName() string {
	return os.Getenv("GCS_BUCKET_NAME")
}
