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
  default     = "gcr.io/cloudrun/hello"
}

variable "storage_bucket_name" {
  description = "Globally unique name for the Cloud Storage bucket that holds product and marketing images."
  type        = string
}

variable "min_instances" {
  description = "Minimum number of Cloud Run instances (0 keeps the free tier idle-cost at zero)."
  type        = number
  default     = 0
}

variable "max_instances" {
  description = "Maximum number of Cloud Run instances."
  type        = number
  default     = 2
}
