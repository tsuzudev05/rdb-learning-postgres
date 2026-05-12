FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive
ENV LANG=ja_JP.UTF-8
ENV LC_ALL=ja_JP.UTF-8

# ─────────────────────────────────────────
# Step 1: 基本パッケージ + PostgreSQLクライアント
# ─────────────────────────────────────────
RUN apt-get update && apt-get install -y \
    curl wget git vim bash-completion \
    ca-certificates gnupg \
    postgresql-client-16 \
    tar gzip locales \
    && locale-gen ja_JP.UTF-8 \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# ─────────────────────────────────────────
# Step 2: Node.js 20（Claude Code の実行に必要）
# apt-get と分けて実行する必要がある
# ─────────────────────────────────────────
RUN curl -fsSL https://deb.nodesource.com/setup_20.x | bash - \
    && apt-get install -y nodejs \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

# ─────────────────────────────────────────
# Step 3: Claude Code（Node.js インストール後に実行）
# ─────────────────────────────────────────
RUN npm install -g @anthropic-ai/claude-code

# ─────────────────────────────────────────
# Step 4: Go 1.22
# ─────────────────────────────────────────
RUN curl -OL https://go.dev/dl/go1.22.3.linux-amd64.tar.gz \
    && tar -C /usr/local -xzf go1.22.3.linux-amd64.tar.gz \
    && rm go1.22.3.linux-amd64.tar.gz

ENV PATH=$PATH:/usr/local/go/bin
ENV GOPATH=/root/go
ENV PATH=$PATH:/root/go/bin

# ─────────────────────────────────────────
# Step 5: Rust
# ─────────────────────────────────────────
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs | sh -s -- -y
ENV PATH=$PATH:/root/.cargo/bin

# ─────────────────────────────────────────
WORKDIR /workspace
CMD ["bash"]
