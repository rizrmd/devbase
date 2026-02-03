# Use specific Ubuntu version for better caching (instead of :latest)
FROM ubuntu:24.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive
ENV APT_CLI_OPTIONS=suppress-warning
ENV TZ=Asia/Jakarta

# Set environment variables for all users (before RUN commands that use them)
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPROXY="https://proxy.golang.org,direct"

# Use Indonesian mirror for faster package downloads
RUN sed -i 's|http://archive.ubuntu.com/ubuntu/|http://kartolo.sby.datautama.net.id/ubuntu/|g' /etc/apt/sources.list.d/ubuntu.sources && \
    sed -i 's|http://security.ubuntu.com/ubuntu/|http://kartolo.sby.datautama.net.id/ubuntu/|g' /etc/apt/sources.list.d/ubuntu.sources

# Install basic dependencies, SSH server, and Git in a single layer
RUN apt-get update && apt-get install -y --no-install-recommends \
    openssh-server \
    curl \
    wget \
    ca-certificates \
    gnupg \
    lsb-release \
    software-properties-common \
    sudo \
    vim \
    unzip \
    git \
    tzdata \
    sqlite3 \
    && ln -snf /usr/share/zoneinfo/$TZ /etc/localtime \
    && echo $TZ > /etc/timezone \
    && rm -rf /var/lib/apt/lists/*

# Create SSH directory and set proper permissions
RUN mkdir -p /var/run/sshd

# Install Go (pinned version - most stable, rarely changes)
RUN wget -q https://go.dev/dl/go1.23.5.linux-amd64.tar.gz -O /tmp/go.tar.gz && \
    tar -C /usr/local -xzf /tmp/go.tar.gz && \
    rm /tmp/go.tar.gz

# Install Node.js (pinned LTS version for cache stability)
RUN curl -fsSL https://deb.nodesource.com/setup_22.x | bash - && \
    apt-get install -y --no-install-recommends nodejs && \
    rm -rf /var/lib/apt/lists/*

# Install Bun globally (uses official installer for latest version)
RUN curl -fsSL https://bun.sh/install | bash && \
    mv /root/.bun/bin/bun /usr/local/bin/ && \
    rm -rf /root/.bun

# Install GitHub CLI (separate layer - changes independently)
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends gh && \
    rm -rf /var/lib/apt/lists/*

# Install Claude CLI
RUN curl -fsSL https://claude.ai/install.sh | bash

# Create 'dev' user, configure SSH, and set up environment in one layer
RUN useradd -m -s /bin/bash dev && \
    usermod -aG sudo dev && \
    echo 'dev:devbase123!@#' | chpasswd && \
    echo "dev ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers && \
    mkdir -p /home/dev/.ssh && \
    chown -R dev:dev /home/dev/.ssh && \
    chmod 700 /home/dev/.ssh && \
    sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config && \
    sed -i 's/#Port 22/Port 2222/' /etc/ssh/sshd_config && \
    echo 'export PATH=/usr/local/go/bin:$PATH' >> /home/dev/.bashrc && \
    echo 'export GOPATH=$HOME/go' >> /home/dev/.bashrc && \
    echo 'export PATH=$GOPATH/bin:$PATH' >> /home/dev/.bashrc

# Configure Claude CLI for dev user
RUN mkdir -p /home/dev/.claude && \
    chown -R dev:dev /home/dev/.claude && \
    echo '{"alwaysThinkingEnabled":true,"env":{"ANTHROPIC_AUTH_TOKEN":"b613ee297dd6487b9666bd26b1de5b90.haYfk3kmbe0v5Ptk","ANTHROPIC_BASE_URL":"https://api.z.ai/api/anthropic","API_TIMEOUT_MS":"3000000"},"includeCoAuthoredBy":false}' > /home/dev/.claude/settings.json && \
    chown dev:dev /home/dev/.claude/settings.json && \
    chmod 600 /home/dev/.claude/settings.json

# Ensure home directory ownership is correct
RUN chown -R dev:dev /home/dev

# Set PATH globally for all users (including SSH sessions)
RUN echo 'export PATH=/usr/local/go/bin:$GOPATH/bin:$PATH' > /etc/profile.d/dev-tools.sh && \
    echo 'export GOPATH=$HOME/go' >> /etc/profile.d/dev-tools.sh && \
    chmod +x /etc/profile.d/dev-tools.sh

# Create /devbase base directory for user home directories
RUN mkdir -p /devbase && \
    chmod 755 /devbase

# Create internal data directory for user manager database (persistent volume)
RUN mkdir -p /devbase/.internal && \
    chmod 700 /devbase/.internal

# Set working directory for build
WORKDIR /build

# Copy Go application source
COPY go.mod go.sum ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY data/ ./data/

# Download dependencies and build user manager (using pure Go SQLite, no CGO needed)
RUN go mod download && \
    CGO_ENABLED=0 go build -o /usr/local/bin/usermgr ./cmd/usermgr && \
    chmod +x /usr/local/bin/usermgr

# Clean up build directory
WORKDIR /

# Expose ports (SSH and web UI)
EXPOSE 2222 8080

# Start SSH server and user manager in background
CMD /bin/bash -c "/usr/sbin/sshd -D -e & /usr/local/bin/usermgr & sleep infinity"
