# DevBase User Manager

A web-based user management system for DevBase that allows admin to create and manage users with home directories in `/devbase/{username}`, each having SSH access and sudo permissions.

## Features

- **Web UI**: Simple HTML interface using Tailwind CSS
- **Admin Authentication**: Session-based authentication with secure cookies
- **User Management**: Create, view, update, and delete users
- **SSH Key Management**: Add and manage SSH public keys for users
- **System Integration**: Automatically creates Linux users with home directories and sudo access
- **Password Management**: Generate secure passwords, change admin password

## Quick Start

### Build

```bash
make build
```

### Run Locally

```bash
# Set environment variables (optional)
export ADMIN_PASSWORD=your_secure_password
export DB_PATH=/var/lib/devbase/users.db
export LISTEN_ADDR=:8080

# Run the application
sudo ./build/usermgr
```

Note: The application requires sudo privileges because it executes system commands (`useradd`, `userdel`, `chpasswd`, etc.) to manage Linux users.

### Access

Open your browser and navigate to:
- http://localhost:8080

**Default credentials:**
- Username: `admin`
- Password: `admin123`

**Important**: Change the default password after first login!

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `ADMIN_PASSWORD` | Default admin password | `admin123` |
| `DB_PATH` | Path to SQLite database | `/devbase/.internal/users.db` |
| `LISTEN_ADDR` | Server address | `:8080` |
| `BASE_HOME_DIR` | Base directory for user homes | `/devbase` |
| `SSH_PORT` | SSH port | `2222` |
| `SESSION_SECRET` | Session encryption key | `devbase-session-secret-change-me` |
| `SESSION_DURATION` | Session timeout in hours | `1` |

## Usage

### Creating a User

1. Navigate to **Users** → **Create User**
2. Fill in the form:
   - **Username**: 3-32 characters, must start with a letter
   - **Password**: Minimum 8 characters (use Generate button for random password)
   - **Email**: Optional
3. Click **Create User**

The system will:
- Create a Linux user with home directory `/devbase/{username}`
- Add the user to the `sudo` group
- Set the password
- Save user metadata to the database

### SSH Access

Users can SSH into the system using:
```bash
ssh -p 2222 username@hostname
```

### Adding SSH Keys

1. Go to **Users** → select a user
2. Scroll to **SSH Keys** section
3. Paste the public key and optionally add a name
4. Click **Add SSH Key**

### Deleting a User

1. Navigate to **Users**
2. Click **Delete** next to the user
3. Confirm the deletion

The system will:
- Remove the Linux user
- Archive the home directory (with `userdel -r`)
- Remove from database

## API Endpoints

The application provides JSON API endpoints:

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/api/users` | List all users |
| `POST` | `/api/users` | Create a user |
| `GET` | `/api/users/:username` | Get user details |
| `DELETE` | `/api/users/:username` | Delete a user |

## Security Considerations

1. **Passwords**: Admin passwords are hashed with bcrypt (cost 12)
2. **Sessions**: Secure, httpOnly cookies with configurable timeout
3. **Sudo Access**: All created users get sudo access (as per requirements)
4. **SSH Keys**: Proper file permissions (0600 for keys, 0700 for .ssh)

## Development

### Project Structure

```
cmd/usermgr/
├── main.go           # Application entry point
├── templates/        # HTML templates
└── README.md

internal/
├── auth/
│   └── admin.go      # Admin authentication
├── user/
│   ├── manager.go    # User CRUD operations
│   └── ssh.go        # SSH key management
├── config/
│   └── config.go     # Configuration management
└── templates/        # Source templates (copied during build)

data/
├── schema.sql        # Database schema
└── users.db          # SQLite database (created at runtime)
```

### Testing

```bash
# Run tests
make test
```

## Docker Integration

The user manager is integrated into the main DevBase Docker container. See the main `Dockerfile` for details.

When the container starts:
1. SSH server runs on port 2222
2. User manager web UI runs on port 8080

## Troubleshooting

### Permission Denied

The application requires sudo to execute system commands. Make sure:
- The binary is executed with sudo privileges
- The user running the application has sudo access

### Database Locked

If you see "database is locked" errors:
- Ensure only one instance of the application is running
- Check file permissions on the database file

### Cannot Create Users

- Check that the `/devbase` directory exists and has proper permissions
- Verify the system has available user IDs
- Check system logs for detailed error messages

## License

Part of the DevBase project.
