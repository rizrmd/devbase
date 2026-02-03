# Use specific Ubuntu version for better caching (instead of :latest)
FROM ubuntu:24.04

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive
ENV APT_CLI_OPTIONS=suppress-warning
ENV TZ=Asia/Jakarta

# Set environment variables for all users (before RUN commands that use them)
ENV PATH="/usr/local/go/bin:${PATH}"
ENV PATH="/root/.bun/bin:${PATH}"
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
    build-essential \
    sudo \
    vim \
    unzip \
    git \
    tzdata \
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

# Install Bun (pinned version for cache stability)
RUN curl -fsSL https://bun.sh/install | bash -s "bun-v1.1.38"

# Install GitHub CLI (separate layer - changes independently)
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list && \
    apt-get update && \
    apt-get install -y --no-install-recommends gh && \
    rm -rf /var/lib/apt/lists/*

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
    echo 'export PATH=$HOME/.bun/bin:$PATH' >> /home/dev/.bashrc && \
    echo 'export GOPATH=$HOME/go' >> /home/dev/.bashrc && \
    echo 'export PATH=$GOPATH/bin:$PATH' >> /home/dev/.bashrc

# Expose SSH port (container port, mapped to different host port by Coolify)
EXPOSE 2222

# Start SSH server in background and keep container alive
CMD /bin/bash -c "/usr/sbin/sshd -D -e & sleep infinity"
