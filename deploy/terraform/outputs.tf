output "container_ip" {
  description = "IP assigned to kronize container"
  value       = proxmox_lxc.kronize.network[0].ip
}

output "container_id" {
  description = "Kronize container ID"
  value       = proxmox_lxc.kronize.vmid
}

output "domain" {
  description = "Domain kronize is served at"
  value       = var.domain
}

output "traefik_config_path" {
  description = "Path to generated Traefik dynamic config (local copy)"
  value       = local_file.traefik_config.filename
}

output "traefik_config_content" {
  description = "Raw Traefik dynamic config YAML"
  value       = nonsensitive(local_file.traefik_config.content)
}
