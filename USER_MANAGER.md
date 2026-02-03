# User Manager Docker Setup

## Quick Start

### Build and Run
```bash
# Build the image
docker build -t devbase:latest .

# Run the container
docker run -d \
  --name devbase \
  -p 2222:2222 \
  -p 8080:8080 \
  -e ADMIN_PASSWORD=your_secure_password \
  devbase:latest
```

### Using Docker Compose
```bash
docker-compose up -d
```

## Access

Once running:
- **Web UI**: http://localhost:8080
- **SSH**: `ssh -p 2222 username@localhost`

## Default Credentials

- **Admin UI**: `admin` / `admin123` (change immediately!)
- **Dev User**: `dev` / `devbase123!@#`

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `ADMIN_PASSWORD` | `admin123` | Admin password for web UI |
| `DB_PATH` | `/var/lib/devbase/users.db` | SQLite database path |
| `LISTEN_ADDR` | `:8080` | Web UI listen address |
| `BASE_HOME_DIR` | `/devbase` | User home directory base |
| `SSH_PORT` | `2222` | SSH port (for reference) |

## Testing

```bash
# Build the image
docker build -t devbase:test .

# Run container
docker run -d --name devbase-test -p 8080:8080 -p 2222:2222 devbase:test

# Check logs
docker logs -f devbase-test

# Access the web UI
open http://localhost:8080

# SSH in as dev user
ssh -p 2222 dev@localhost
# Password: devbase123!@#

# Create a test user via the web UI, then test SSH
ssh -p 2222 testuser@localhost

# Cleanup
docker stop devbase-test && docker rm devbase-test
```

## Container Startup

The container starts three services:
1. **SSH Server** on port 2222 (via `/usr/sbin/sshd`)
2. **User Manager** on port 8080 (via `/usr/local/bin/usermgr`)
3. **Cloudflare Warp** (optional, via `warp-svc`)

All processes run in background with the container kept alive by `sleep infinity`.
