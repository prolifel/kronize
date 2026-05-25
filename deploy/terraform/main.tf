terraform {
  required_version = ">= 1.5"
  required_providers {
    proxmox = {
      source  = "Telmate/proxmox"
      version = "3.0.2-rc07"
    }
    local = {
      source  = "hashicorp/local"
      version = "2.5.2"
    }
    null = {
      source  = "hashicorp/null"
      version = "3.2.3"
    }
  }
}

provider "proxmox" {
  pm_user         = "${var.proxmox_ssh_user}@pam"
  pm_password     = var.proxmox_ssh_password
  pm_api_url      = var.proxmox_api_url
  pm_tls_insecure = var.proxmox_tls_insecure
  pm_debug        = true
}

# ── LXC Container ────────────────────────────────────────
resource "proxmox_lxc" "kronize" {
  target_node  = var.proxmox_node
  hostname     = "kronize"
  vmid         = var.kronize_ct_id
  ostemplate   = var.template_file
  password     = var.ct_root_password
  unprivileged = false # Docker requires privileged
  tags         = "10.10.10.${var.kronize_ct_id}"

  cores  = var.cpu_cores
  memory = var.memory_mb
  swap   = var.swap_mb

  rootfs {
    storage = var.storage_pool
    size    = var.disk_size
  }

  network {
    name   = "eth0"
    bridge = var.bridge
    ip     = "10.10.10.${var.kronize_ct_id}/24"
    gw     = "10.10.10.1"
  }

  features {
    nesting = true
    keyctl  = true
  }
}

# ── Traefik dynamic config ───────────────────────────────
resource "local_file" "traefik_config" {
  content = templatefile("${path.module}/templates/traefik.yml.tftpl", {
    domain        = var.domain
    container_ip  = split("/", proxmox_lxc.kronize.network[0].ip)[0]
    app_port      = var.app_port
    tls_cert_file = var.domain != "" ? var.tls_cert_file : ""
    tls_key_file  = var.domain != "" ? var.tls_key_file : ""
  })
  filename = "${path.module}/.gen/kronize.yml"
}

# ── Container provisioning ───────────────────────────────
resource "null_resource" "container_setup" {
  depends_on = [proxmox_lxc.kronize]

  triggers = {
    registry_url      = var.registry_url
    registry_username = nonsensitive(sha256(var.registry_username))
    registry_password = nonsensitive(sha256(var.registry_password))
    jwt_secret        = nonsensitive(sha256(var.jwt_secret))
    app_port          = var.app_port
  }

  connection {
    type     = "ssh"
    user     = var.proxmox_ssh_user
    password = var.proxmox_ssh_password
    host     = var.proxmox_host
  }

  provisioner "file" {
    content = templatefile("${path.module}/templates/setup.sh.tftpl", {
      registry_url      = var.registry_url
      registry_username = var.registry_username
      registry_password = var.registry_password
      app_port          = var.app_port
      jwt_secret        = var.jwt_secret
      domain            = var.domain
    })
    destination = "/tmp/kronize-setup.sh"
  }

  provisioner "remote-exec" {
    inline = [
      "pct start ${var.kronize_ct_id} || true",
      "until pct status ${var.kronize_ct_id} | grep -q running; do sleep 1; done",
      "sleep 3",
      "pct exec ${var.kronize_ct_id} -- mkdir -p /opt/kronize/data /opt/kronize/scripts",
      "pct push ${var.kronize_ct_id} /tmp/kronize-setup.sh /tmp/kronize-setup.sh",
      "pct exec ${var.kronize_ct_id} -- chmod +x /tmp/kronize-setup.sh",
      "pct exec ${var.kronize_ct_id} -- /tmp/kronize-setup.sh"
    ]
  }
}

# ── Deploy config to Traefik container ───────────────────
resource "null_resource" "traefik_deploy" {
  count = var.domain != "" ? 1 : 0

  depends_on = [local_file.traefik_config]

  triggers = {
    config_sha = sha256(local_file.traefik_config.content)
    cert_sha   = try(filemd5("${path.module}/../certs/${var.domain}.cer"), "")
    key_sha    = try(filemd5("${path.module}/../certs/${var.domain}.key"), "")
  }

  connection {
    type     = "ssh"
    user     = var.proxmox_ssh_user
    password = var.proxmox_ssh_password
    host     = var.proxmox_host
  }

  provisioner "file" {
    content     = local_file.traefik_config.content
    destination = "/tmp/kronize.yml"
  }

  provisioner "file" {
    source      = "${path.module}/../certs/${var.domain}.cer"
    destination = "/tmp/kronize.crt"
  }

  provisioner "file" {
    source      = "${path.module}/../certs/${var.domain}.key"
    destination = "/tmp/kronize.key"
  }

  provisioner "remote-exec" {
    inline = [
      "pct start ${var.traefik_ct_id} || true",
      "until pct status ${var.traefik_ct_id} | grep -q running; do sleep 1; done",
      "sleep 2",
      "pct exec ${var.traefik_ct_id} -- mkdir -p ${dirname(var.tls_cert_file)}",
      "pct push ${var.traefik_ct_id} /tmp/kronize.yml ${var.traefik_config_dir}/kronize.yml",
      "pct push ${var.traefik_ct_id} /tmp/kronize.crt ${var.tls_cert_file}",
      "pct push ${var.traefik_ct_id} /tmp/kronize.key ${var.tls_key_file}",
      "rm /tmp/kronize.yml /tmp/kronize.crt /tmp/kronize.key"
    ]
  }
}
