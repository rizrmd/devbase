# DevBase Docker Container

Ubuntu-based development container with SSH access, user management, and modern development tools.

## Features

- **Base**: Ubuntu 24.04
- **User**: `dev` (with sudo privileges, no password required)
- **SSH**: Port 2222 on host → 22 in container
- **Web UI**: Port 8080 - User management interface
- **Tools Installed**:
  - Bun (JavaScript runtime)
  - Node.js (LTS)
  - Go 1.23.5
  - Git
  - GitHub CLI (gh)
- **User Persistence**: All created users persist across container restarts

## Quick Start

### 1. Build and Run

```bash
# Build the image
docker build -t devbase .

# Run the container with user data persistence
docker run -d \
  --name devbase \
  --hostname devbase \
  -p 2222:22 \
  -p 8080:8080 \
  -v ./devbase-data:/devbase \
  --restart unless-stopped \
  devbase
```

**Important**: Mount `/devbase` as a volume to persist user data across container restarts. Without this, all users will be lost on restart.

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

## Web User Management Interface

The container includes a web-based user management interface accessible at `http://localhost:8080`.

### Features
- Create, delete, and manage users via web UI
- Change user passwords
- All user data persists in `/devbase/.internal/users.json`
- Users are automatically recreated on container startup

### Accessing the Web UI
1. Navigate to `http://localhost:8080` in your browser
2. Login with admin credentials (default: admin/admin123)
3. Set the admin password first time you login

### Managing Users via Web UI
- **Create User**: Enter username and password (min 3 chars, alphanumeric + underscore/hyphen)
- **Delete User**: Remove user and their home directory
- **Change Password**: Update password for any user

### How Persistence Works
1. User data is stored in `/devbase/.internal/users.json` (in your mounted volume)
2. On container startup, the `user-rebuild` tool reads this file
3. All users are recreated with their saved passwords
4. This happens automatically before SSH and web UI start

### Handling Existing User Directories

If you have existing user directories in `/devbase` that were created before the persistence feature, they won't be automatically restored because we don't have their passwords.

When you restart the container, the startup process will detect these orphaned directories and display a warning:

```
WARNING: Found orphaned home directories:
  - /devbase/username1
  - /devbase/username2
```

To restore these users:
1. Use the web UI at `http://localhost:8080`
2. Create each user with the same username
3. The existing home directory will be preserved and reused
4. Set a new password for the user

**Note**: You need to know the old password to set the same one, or set a new password and inform the user.

## Directory Structure

```
.
├── devbase-data/              # Persisted user data (mount this to /devbase)
│   ├── .internal/
│   │   └── users.json        # User database (auto-generated)
│   └── {username}/           # User home directories
└── Dockerfile                 # Container definition
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
