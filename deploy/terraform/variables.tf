# ── Proxmox ──────────────────────────────────────────────
variable "proxmox_api_url" {
  description = "Proxmox API endpoint"
  type        = string
  sensitive   = true
}

variable "proxmox_api_token_id" {
  description = "Proxmox API token ID (user@realm!token-name)"
  type        = string
  sensitive   = true
}

variable "proxmox_api_token_secret" {
  description = "Proxmox API token secret"
  type        = string
  sensitive   = true
}

variable "proxmox_node" {
  description = "Proxmox node name"
  type        = string
  default     = "pve"
}

variable "proxmox_tls_insecure" {
  description = "Skip TLS verification for self-signed certs"
  type        = bool
  default     = true
}

variable "proxmox_host" {
  description = "IP or hostname of the Proxmox node (for SSH + pct push)"
  type        = string
}

variable "proxmox_ssh_user" {
  description = "SSH user for the Proxmox node"
  type        = string
  default     = "root"
}

variable "proxmox_ssh_password" {
  description = "SSH password for the Proxmox node"
  type        = string
  sensitive   = true
}

# ── Container ────────────────────────────────────────────
variable "kronize_ct_id" {
  description = "Kronize container ID"
  type        = number
  default     = 109
}

variable "ct_root_password" {
  description = "Root password for the container"
  type        = string
  sensitive   = true
}

variable "template_file" {
  description = "CT template path on Proxmox storage"
  type        = string
  default     = "local:vztmpl/debian-13-standard_13.1-2_amd64.tar.zst"
}

variable "cpu_cores" {
  description = "Number of CPU cores"
  type        = number
  default     = 2
}

variable "memory_mb" {
  description = "RAM in MB"
  type        = number
  default     = 512
}

variable "swap_mb" {
  description = "Swap in MB"
  type        = number
  default     = 512
}

variable "disk_size" {
  description = "Root disk size, e.g. '5G'"
  type        = string
  default     = "8G"
}

variable "storage_pool" {
  description = "Proxmox storage pool for container disk"
  type        = string
  default     = "local-lvm"
}

variable "bridge" {
  description = "Network bridge interface"
  type        = string
  default     = "vlab01"
}

# ── Docker registry ──────────────────────────────────────
variable "registry_url" {
  description = "Docker registry host"
  type        = string
  sensitive   = true
}

variable "registry_username" {
  description = "Registry username"
  type        = string
  sensitive   = true
}

variable "registry_password" {
  description = "Registry password"
  type        = string
  sensitive   = true
}

# ── App ──────────────────────────────────────────────────
variable "app_port" {
  description = "Port kronize listens on"
  type        = number
  default     = 8080
}

variable "jwt_secret" {
  description = "JWT signing secret for kronize"
  type        = string
  sensitive   = true
}

# ── Traefik ──────────────────────────────────────────────
variable "domain" {
  description = "Domain for kronize. Empty string = port-based routing (no DNS)."
  type        = string
}

variable "traefik_config_dir" {
  description = "Traefik file-provider config directory"
  type        = string
  default     = "/etc/traefik/conf.d"
}

variable "tls_cert_file" {
  description = "Path to TLS certificate on Traefik host"
  type        = string
}

variable "tls_key_file" {
  description = "Path to TLS private key on Traefik host"
  type        = string
}

variable "traefik_ct_id" {
  description = "Traefik container ID"
  type        = number
  default     = 103
}
