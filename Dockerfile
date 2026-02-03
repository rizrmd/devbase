FROM ubuntu:latest

# Prevent interactive prompts during package installation
ENV DEBIAN_FRONTEND=noninteractive

# Install basic dependencies and SSH server
RUN apt-get update && apt-get install -y \
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
    && rm -rf /var/lib/apt/lists/*

# Create SSH directory and set proper permissions
RUN mkdir -p /var/run/sshd

# Install Node.js (using NodeSource for latest LTS)
RUN curl -fsSL https://deb.nodesource.com/setup_lts.x | bash - && \
    apt-get install -y nodejs && \
    rm -rf /var/lib/apt/lists/*

# Install Bun
RUN curl -fsSL https://bun.sh/install | bash

# Install Go
RUN wget https://go.dev/dl/go1.23.5.linux-amd64.tar.gz -O /tmp/go.tar.gz && \
    tar -C /usr/local -xzf /tmp/go.tar.gz && \
    rm /tmp/go.tar.gz

# Install Git (usually comes with Ubuntu, but ensure it's there)
RUN apt-get update && apt-get install -y git && \
    rm -rf /var/lib/apt/lists/*

# Install GitHub CLI
RUN type -p curl >/dev/null || (apt-get update && apt-get install -y curl) && \
    curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg | dd of=/usr/share/keyrings/githubcli-archive-keyring.gpg && \
    echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/githubcli-archive-keyring.gpg] https://cli.github.com/packages stable main" | tee /etc/apt/sources.list.d/github-cli.list && \
    apt-get update && \
    apt-get install -y gh && \
    rm -rf /var/lib/apt/lists/*

# Create 'dev' user with home directory
RUN useradd -m -s /bin/bash dev && \
    usermod -aG sudo dev

# Set up sudo for dev user without password
RUN echo "dev ALL=(ALL) NOPASSWD:ALL" >> /etc/sudoers

# Configure SSH
RUN mkdir -p /home/dev/.ssh && \
    chown -R dev:dev /home/dev/.ssh && \
    chmod 700 /home/dev/.ssh

# Allow password authentication for initial setup (can be disabled later)
RUN sed -i 's/#PermitRootLogin prohibit-password/PermitRootLogin yes/' /etc/ssh/sshd_config && \
    sed -i 's/#PasswordAuthentication yes/PasswordAuthentication yes/' /etc/ssh/sshd_config

# Set environment variables for all users
# Add Go to PATH
ENV PATH="/usr/local/go/bin:${PATH}"
# Add Bun to PATH
ENV PATH="/root/.bun/bin:${PATH}"
# Add Go to dev user's bashrc
RUN echo 'export PATH=/usr/local/go/bin:$PATH' >> /home/dev/.bashrc && \
    echo 'export PATH=$HOME/.bun/bin:$PATH' >> /home/dev/.bashrc && \
    echo 'export GOPATH=$HOME/go' >> /home/dev/.bashrc && \
    echo 'export PATH=$GOPATH/bin:$PATH' >> /home/dev/.bashrc

# Expose SSH port
EXPOSE 22

# Start SSH server
CMD ["/usr/sbin/sshd", "-D", "-e"]
