output "cloud_run_url" {
  description = "Public URL of the deployed Cloud Run backend service."
  value       = google_cloud_run_v2_service.backend.uri
}

output "storage_bucket_name" {
  description = "Name of the Cloud Storage bucket holding product and marketing images. Empty when enable_gcs_bucket = false (the default — this deploy uses STORAGE_BACKEND=local)."
  value       = try(google_storage_bucket.images[0].name, "")
}
