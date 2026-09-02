variable "gcp_project_id" {
  description = "GCP project ID that owns the Cloud Run service and Cloud Storage bucket."
  type        = string
}

variable "gcp_region" {
  description = "GCP region for Cloud Run and Cloud Storage."
  type        = string
  default     = "us-central1"
}

variable "service_name" {
  description = "Name of the Cloud Run service running the BrewOps backend."
  type        = string
  default     = "brew-ops-backend"
}

variable "container_image" {
  description = "Fully qualified container image to deploy (e.g. from Artifact Registry)."
  type        = string
  default     = "us-central1-docker.pkg.dev/brewops-demo/brewops-demo/backend:latest"
}

variable "enable_gcs_bucket" {
  description = "Whether to create the GCS bucket for product/marketing images. This deploy runs with storage_backend = \"local\", so the bucket is unneeded and defaults to off — set true only when switching storage_backend to \"gcs\"."
  type        = bool
  default     = false
}

variable "storage_bucket_name" {
  description = "Globally unique name for the Cloud Storage bucket that holds product and marketing images. Only required when enable_gcs_bucket = true."
  type        = string
  default     = ""
}

variable "min_instances" {
  description = "Minimum number of Cloud Run instances (0 keeps the free tier idle-cost at zero)."
  type        = number
  default     = 0
}

variable "max_instances" {
  description = "Maximum number of Cloud Run instances. Kept low as a safety cap against an unexpected traffic spike on a public demo."
  type        = number
  default     = 3
}

# ---------------------------------------------------------------------------
# Backend runtime environment variables — mirrors backend/config/config.go.
# Sensitive values have no default: they must come from terraform.tfvars
# (gitignored) and are never hardcoded in main.tf.
# ---------------------------------------------------------------------------

variable "database_url" {
  description = "Postgres connection string (Supabase pooler). Sensitive."
  type        = string
  sensitive   = true
}

variable "jwt_secret" {
  description = "Secret used to sign/verify JWT access and refresh tokens. Sensitive."
  type        = string
  sensitive   = true
}

variable "jwt_access_token_ttl" {
  description = "JWT access token lifetime (Go duration string)."
  type        = string
  default     = "8h"
}

variable "jwt_refresh_token_ttl" {
  description = "JWT refresh token lifetime (Go duration string)."
  type        = string
  default     = "720h"
}

variable "app_env" {
  description = "APP_ENV for the backend. \"production\" fails closed on dev-only routes (e.g. POST /auth/register)."
  type        = string
  default     = "production"
}

variable "demo_mode" {
  description = "Enables demo-only surface area (demo-reset endpoint, AI usage quotas). See CLAUDE.md's DEMO MODE section."
  type        = bool
  default     = true
}

variable "demo_admin_email" {
  description = "Email for the single demo admin account, upserted by seed/reset. Sensitive."
  type        = string
  sensitive   = true
}

variable "demo_admin_password" {
  description = "Password for the single demo admin account, upserted by seed/reset. Sensitive."
  type        = string
  sensitive   = true
}

variable "demo_reset_token" {
  description = "Shared secret compared against the X-Demo-Reset-Token header on POST /admin/demo-reset. Required when demo_mode is true. Sensitive."
  type        = string
  sensitive   = true
}

variable "storage_backend" {
  description = "Image storage backend: \"local\" (./local-storage/ + public_base_url) or \"gcs\"."
  type        = string
  default     = "local"
}

variable "public_base_url" {
  description = "Base URL LocalDiskStorageClient prefixes onto filenames to build public image URLs. Only used when storage_backend is \"local\". Update this to the deployed Cloud Run URL after the first apply (chicken-and-egg: the URL isn't known before that)."
  type        = string
  default     = "http://localhost:8080"
}

variable "openai_api_key" {
  description = "OpenAI API key used by POST /inventory/suggest. Sensitive."
  type        = string
  sensitive   = true
}

variable "cors_allowed_origins" {
  description = "Allowed CORS origin for the frontend. Placeholder until the frontend has a real deployed URL — update then."
  type        = string
  default     = "http://localhost:5173"
}

variable "seed_assets_dir" {
  description = "Absolute path demoseed.Reset copies seed images from inside the container. Must match exactly where the Dockerfile's COPY seed-assets /seed-assets lands — the backend's own default (\"./seed-assets\") is relative to the process's working directory, which Cloud Run does not guarantee equals the source tree root."
  type        = string
  default     = "/seed-assets"
}

variable "demo_reset_schedule" {
  description = "Cron schedule (Cloud Scheduler format) for the automatic demo reset."
  type        = string
  default     = "0 */6 * * *" # every 6 hours
}
