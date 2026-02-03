# DevBase Docker Container

Ubuntu-based development container with SSH access and modern development tools.

## Features

- **Base**: Latest Ubuntu
- **User**: `dev` (with sudo privileges, no password required)
- **SSH**: Port 2222 on host → 22 in container
- **Tools Installed**:
  - Bun (JavaScript runtime)
  - Node.js (LTS)
  - Go 1.23.5
  - Git
  - GitHub CLI (gh)

## Quick Start

### 1. Build and Run

```bash
# Build the image
docker build -t devbase .

# Run the container
docker run -d \
  --name devbase \
  --hostname devbase \
  -p 2222:22 \
  -v ./home-data:/home/dev \
  -v ~/.ssh:/home/dev/.ssh-host:ro \
  --restart unless-stopped \
  devbase
```

### 2. Set Up SSH Access

#### Option A: Password Authentication (Default)

The container starts with password authentication enabled. You can set a password for the `dev` user:

```bash
docker exec -it devbase passwd dev
```

Then SSH in:
```bash
ssh -p 2222 dev@localhost
```

#### Option B: SSH Key Authentication (Recommended)

1. Copy your public key to the container:
```bash
# From your host machine
cat ~/.ssh/id_rsa.pub | docker exec -i devbase tee /home/dev/.ssh/authorized_keys > /dev/null
```

2. Set proper permissions:
```bash
docker exec -it devbase chown -R dev:dev /home/dev/.ssh
docker exec -it devbase chmod 700 /home/dev/.ssh
docker exec -it devbase chmod 600 /home/dev/.ssh/authorized_keys
```

3. Disable password authentication (optional, for better security):
```bash
docker exec -it devbase sed -i 's/PasswordAuthentication yes/PasswordAuthentication no/' /etc/ssh/sshd_config
docker restart devbase
```

4. SSH in without password:
```bash
ssh -p 2222 dev@localhost
```

## Directory Structure

```
.
├── home-data/       # Persists dev user's home directory
└── Dockerfile       # Container definition
```

## Using GitHub CLI (gh)

To authenticate with GitHub:

```bash
# Inside the container
gh auth login
```

Or copy your existing gh credentials:
```bash
# From host (if you have gh config)
cp ~/.config/gh ./home-data/.config/gh -r
```

## Development Tips

### Add your Git config
```bash
git config --global user.name "Your Name"
git config --global user.email "your@email.com"
```

### Access host's SSH keys from container
Your host SSH keys are mounted at `/home/dev/.ssh-host` (read-only).

### Stop the container
```bash
docker stop devbase
docker rm devbase
```

### Restart the container
```bash
docker restart devbase
```

### Execute commands without SSH
```bash
docker exec -it devbase bash
```

## Available Versions

- **Node.js**: Latest LTS (check with `node --version`)
- **Bun**: Latest (check with `bun --version`)
- **Go**: 1.23.5 (check with `go version`)
- **Git**: Latest from Ubuntu repos (check with `git --version`)
- **GitHub CLI**: Latest (check with `gh --version`)
