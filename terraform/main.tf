terraform {
  required_version = ">= 1.9"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 6.0"
    }
  }
}

provider "google" {
  project = var.gcp_project_id
  region  = var.gcp_region
}

# Cloud Storage bucket for product and marketing images. The backend never
# stores binary image data in Postgres — only the public URL. Only created
# when enable_gcs_bucket = true; this demo deploy runs with
# STORAGE_BACKEND=local, which never touches GCS at all.
resource "google_storage_bucket" "images" {
  count = var.enable_gcs_bucket ? 1 : 0

  name                        = var.storage_bucket_name
  location                    = var.gcp_region
  storage_class               = "STANDARD"
  uniform_bucket_level_access = true

  cors {
    origin          = ["*"]
    method          = ["GET"]
    response_header = ["Content-Type"]
    max_age_seconds = 3600
  }
}

resource "google_storage_bucket_iam_member" "public_read" {
  count = var.enable_gcs_bucket ? 1 : 0

  bucket = google_storage_bucket.images[0].name
  role   = "roles/storage.objectViewer"
  member = "allUsers"
}

# Cloud Run service hosting the Fiber backend. min_instances = 0 lets the
# service scale to zero, which is what keeps this inside the free tier.
resource "google_cloud_run_v2_service" "backend" {
  name     = var.service_name
  location = var.gcp_region

  template {
    scaling {
      min_instance_count = var.min_instances
      max_instance_count = var.max_instances
    }

    containers {
      image = var.container_image

      ports {
        container_port = 8080
      }

      resources {
        limits = {
          cpu    = "1"
          memory = "512Mi"
        }
      }

      env {
        name  = "APP_ENV"
        value = var.app_env
      }
      env {
        name  = "DATABASE_URL"
        value = var.database_url
      }
      env {
        name  = "JWT_SECRET"
        value = var.jwt_secret
      }
      env {
        name  = "JWT_ACCESS_TOKEN_TTL"
        value = var.jwt_access_token_ttl
      }
      env {
        name  = "JWT_REFRESH_TOKEN_TTL"
        value = var.jwt_refresh_token_ttl
      }
      env {
        name  = "DEMO_MODE"
        value = var.demo_mode ? "true" : "false"
      }
      env {
        name  = "DEMO_ADMIN_EMAIL"
        value = var.demo_admin_email
      }
      env {
        name  = "DEMO_ADMIN_PASSWORD"
        value = var.demo_admin_password
      }
      env {
        name  = "DEMO_RESET_TOKEN"
        value = var.demo_reset_token
      }
      env {
        name  = "STORAGE_BACKEND"
        value = var.storage_backend
      }
      env {
        name  = "PUBLIC_BASE_URL"
        value = var.public_base_url
      }
      env {
        name  = "OPENAI_API_KEY"
        value = var.openai_api_key
      }
      env {
        name  = "CORS_ALLOWED_ORIGINS"
        value = var.cors_allowed_origins
      }
      env {
        name  = "SEED_ASSETS_DIR"
        value = var.seed_assets_dir
      }
    }
  }
}

resource "google_cloud_run_v2_service_iam_member" "public_invoker" {
  name     = google_cloud_run_v2_service.backend.name
  location = google_cloud_run_v2_service.backend.location
  role     = "roles/run.invoker"
  member   = "allUsers"
}

# Periodically calls the demo-reset endpoint so the public demo cleans
# itself up without manual intervention. The URI is built from the Cloud
# Run service's own .uri attribute (never hardcoded, never a separate
# variable) so Terraform's dependency graph keeps it correct automatically.
resource "google_cloud_scheduler_job" "demo_reset" {
  name      = "brewops-demo-reset"
  region    = var.gcp_region
  schedule  = var.demo_reset_schedule
  time_zone = "America/Mexico_City"

  http_target {
    http_method = "POST"
    uri         = "${google_cloud_run_v2_service.backend.uri}/api/v1/admin/demo-reset"
    body        = base64encode("{}")

    headers = {
      "Content-Type"       = "application/json"
      "X-Demo-Reset-Token" = var.demo_reset_token
    }
  }
}
