FROM ubuntu:24.04

ENV DEBIAN_FRONTEND=noninteractive
ENV TERM=xterm-256color

# Layer 1: System packages
RUN apt-get update && apt-get install -y --no-install-recommends \
        ca-certificates \
        cmake \
        curl \
        device-tree-compiler \
        dfu-util \
        file \
        git \
        gperf \
        jq \
        less \
        ninja-build \
        openssh-client \
        build-essential \
        libicu-dev \
        libsdl2-dev \
        python3 \
        python3-dev \
        python3-pip \
        python3-setuptools \
        python3-venv \
        python3-wheel \
        unzip \
        wget \
        zsh \
        zsh-syntax-highlighting \
        zsh-autosuggestions \
        librsvg2-bin \
        silversearcher-ag \
    && rm -rf /var/lib/apt/lists/*

# Headless Chromium runtime deps (Puppeteer canonical list from pptr.dev/troubleshooting).
# No X11 server needed — Chrome runs with --headless. librsvg2-bin above already
# pulls in libcairo2, libglib2.0-0, libpango-1.0-0, libgdk-pixbuf2.0-0.
RUN apt-get update && apt-get install -y --no-install-recommends \
        fonts-liberation \
        libasound2t64 \
        libatk-bridge2.0-0 \
        libatk1.0-0 \
        libcups2 \
        libdbus-1-3 \
        libgbm1 \
        libnspr4 \
        libnss3 \
        libpangocairo-1.0-0 \
        libxcomposite1 \
        libxdamage1 \
        libxext6 \
        libxfixes3 \
        libxkbcommon0 \
        libxrandr2 \
    && rm -rf /var/lib/apt/lists/*

# Docker CLI
RUN install -m 0755 -d /etc/apt/keyrings \
    && curl -fsSL https://download.docker.com/linux/ubuntu/gpg -o /etc/apt/keyrings/docker.asc \
    && chmod a+r /etc/apt/keyrings/docker.asc \
    && echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/docker.asc] https://download.docker.com/linux/ubuntu noble stable" \
        > /etc/apt/sources.list.d/docker.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends docker-ce-cli \
    && rm -rf /var/lib/apt/lists/*

# GitHub CLI
RUN curl -fsSL https://cli.github.com/packages/githubcli-archive-keyring.gpg \
        -o /etc/apt/keyrings/gh.gpg \
    && chmod a+r /etc/apt/keyrings/gh.gpg \
    && echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/gh.gpg] https://cli.github.com/packages stable main" \
        > /etc/apt/sources.list.d/github-cli.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends gh \
    && rm -rf /var/lib/apt/lists/*

# WezTerm nightly
ARG WEZTERM_VERSION=20260623-212301-09e52dd8
RUN echo 'deb [trusted=yes] https://apt.fury.io/wez/ * *' \
        > /etc/apt/sources.list.d/wezterm.list \
    && apt-get update \
    && apt-get install -y --no-install-recommends wezterm-nightly=${WEZTERM_VERSION} \
    && rm -rf /var/lib/apt/lists/*

# Layer 2: Pinned static binaries

# ripgrep
ARG RG_VERSION=15.1.0
ARG RG_CHECKSUM=1c9297be4a084eea7ecaedf93eb03d058d6faae29bbc57ecdaf5063921491599
RUN curl -fsSL "https://github.com/BurntSushi/ripgrep/releases/download/${RG_VERSION}/ripgrep-${RG_VERSION}-x86_64-unknown-linux-musl.tar.gz" \
        -o /tmp/rg.tar.gz \
    && echo "${RG_CHECKSUM}  /tmp/rg.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/rg.tar.gz -C /tmp \
    && cp "/tmp/ripgrep-${RG_VERSION}-x86_64-unknown-linux-musl/rg" /usr/local/bin/rg \
    && rm -rf /tmp/rg.tar.gz /tmp/ripgrep-*

# fd
ARG FD_VERSION=10.4.2
ARG FD_CHECKSUM=e3257d48e29a6be965187dbd24ce9af564e0fe67b3e73c9bdcd180f4ec11bdde
RUN curl -fsSL "https://github.com/sharkdp/fd/releases/download/v${FD_VERSION}/fd-v${FD_VERSION}-x86_64-unknown-linux-musl.tar.gz" \
        -o /tmp/fd.tar.gz \
    && echo "${FD_CHECKSUM}  /tmp/fd.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/fd.tar.gz -C /tmp \
    && cp "/tmp/fd-v${FD_VERSION}-x86_64-unknown-linux-musl/fd" /usr/local/bin/fd \
    && rm -rf /tmp/fd.tar.gz /tmp/fd-*

# clangd
ARG CLANGD_VERSION=22.1.0
ARG CLANGD_CHECKSUM=c54e57dbff3ccc9e8352367ddb7030ad3f624073ec58c7477424e7919f578572
RUN curl -fsSL "https://github.com/clangd/clangd/releases/download/${CLANGD_VERSION}/clangd-linux-${CLANGD_VERSION}.zip" \
        -o /tmp/clangd.zip \
    && echo "${CLANGD_CHECKSUM}  /tmp/clangd.zip" | sha256sum -c - \
    && unzip -q /tmp/clangd.zip -d /tmp \
    && cp /tmp/clangd_*/bin/clangd /usr/local/bin/clangd \
    && rm -rf /tmp/clangd.zip /tmp/clangd_*

# bat
ARG BAT_VERSION=0.26.1
ARG BAT_CHECKSUM=0dcd8ac79732c0d5b136f11f4ee00e581440e16a44eab5b3105b611bbf2cf191
RUN curl -fsSL "https://github.com/sharkdp/bat/releases/download/v${BAT_VERSION}/bat-v${BAT_VERSION}-x86_64-unknown-linux-musl.tar.gz" \
        -o /tmp/bat.tar.gz \
    && echo "${BAT_CHECKSUM}  /tmp/bat.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/bat.tar.gz -C /tmp \
    && cp "/tmp/bat-v${BAT_VERSION}-x86_64-unknown-linux-musl/bat" /usr/local/bin/bat \
    && rm -rf /tmp/bat.tar.gz /tmp/bat-*

# trdsql
ARG TRDSQL_VERSION=1.2.3
ARG TRDSQL_CHECKSUM=812d1e81f98b6a8b6e7c5f1d8ef8e451d6ea75a14bd8a72c1c2334d7c78d9305
RUN curl -fsSL "https://github.com/noborus/trdsql/releases/download/v${TRDSQL_VERSION}/trdsql_v${TRDSQL_VERSION}_linux_amd64.zip" \
        -o /tmp/trdsql.zip \
    && echo "${TRDSQL_CHECKSUM}  /tmp/trdsql.zip" | sha256sum -c - \
    && unzip -q /tmp/trdsql.zip -d /tmp \
    && cp "/tmp/trdsql_v${TRDSQL_VERSION}_linux_amd64/trdsql" /usr/local/bin/trdsql \
    && rm -rf /tmp/trdsql.zip /tmp/trdsql_*

# delta
ARG DELTA_VERSION=0.19.2
ARG DELTA_CHECKSUM=f1ea01ca7728ce3462debc359f39dfc7cbbc1a63224b71fefabf92042864aa1b
RUN curl -fsSL "https://github.com/dandavison/delta/releases/download/${DELTA_VERSION}/delta-${DELTA_VERSION}-x86_64-unknown-linux-musl.tar.gz" \
        -o /tmp/delta.tar.gz \
    && echo "${DELTA_CHECKSUM}  /tmp/delta.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/delta.tar.gz -C /tmp \
    && cp "/tmp/delta-${DELTA_VERSION}-x86_64-unknown-linux-musl/delta" /usr/local/bin/delta \
    && rm -rf /tmp/delta.tar.gz /tmp/delta-*

# lazygit
ARG LAZYGIT_VERSION=0.61.1
ARG LAZYGIT_CHECKSUM=1b91e660700f2332696726b635202576b543e2bc49b639830dccd26bc5160d5d
RUN curl -fsSL "https://github.com/jesseduffield/lazygit/releases/download/v${LAZYGIT_VERSION}/lazygit_${LAZYGIT_VERSION}_Linux_x86_64.tar.gz" \
        -o /tmp/lazygit.tar.gz \
    && echo "${LAZYGIT_CHECKSUM}  /tmp/lazygit.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/lazygit.tar.gz -C /tmp lazygit \
    && cp /tmp/lazygit /usr/local/bin/lazygit \
    && rm -rf /tmp/lazygit.tar.gz /tmp/lazygit

# markless
ARG MARKLESS_VERSION=0.9.29
ARG MARKLESS_CHECKSUM=7c7911ee34707ff85dbbee2b7f588f4366067adbad2a3a4d5730d0165249599d
RUN curl -fsSL "https://github.com/jvanderberg/markless/releases/download/v${MARKLESS_VERSION}/markless-x86_64-unknown-linux-gnu.tar.gz" \
        -o /tmp/markless.tar.gz \
    && echo "${MARKLESS_CHECKSUM}  /tmp/markless.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/markless.tar.gz -C /tmp \
    && cp /tmp/markless /usr/local/bin/markless \
    && rm -rf /tmp/markless.tar.gz /tmp/markless

# markdown-reader (TUI markdown viewer)
ARG MTE_VERSION=1.34.73
ARG MTE_CHECKSUM=16fc61a29e02b99519fd0f44d7cbc6f90073f23fbe8e7199ac2d6c55748ed028
RUN curl -fsSL "https://github.com/leboiko/markdown-reader/releases/download/v${MTE_VERSION}/markdown-reader-${MTE_VERSION}-x86_64-unknown-linux-musl.tar.gz" \
        -o /tmp/mte.tar.gz \
    && echo "${MTE_CHECKSUM}  /tmp/mte.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/mte.tar.gz -C /tmp \
    && cp "/tmp/markdown-reader-${MTE_VERSION}-x86_64-unknown-linux-musl/markdown-reader" /usr/local/bin/markdown-reader \
    && rm -rf /tmp/mte.tar.gz /tmp/markdown-reader-*

# broot
ARG BROOT_VERSION=1.56.4
ARG BROOT_CHECKSUM=234ce9346dc3a38cb602f62bd872fa20f1b6227a6eca6c3c195fb36be5062df5
RUN curl -fsSL "https://github.com/Canop/broot/releases/download/v${BROOT_VERSION}/broot_${BROOT_VERSION}.zip" \
        -o /tmp/broot.zip \
    && unzip -o -j /tmp/broot.zip "x86_64-unknown-linux-musl/broot" -d /tmp/broot-extract \
    && echo "${BROOT_CHECKSUM}  /tmp/broot-extract/broot" | sha256sum -c - \
    && cp /tmp/broot-extract/broot /usr/local/bin/broot \
    && rm -rf /tmp/broot.zip /tmp/broot-extract

# bd (beads)
ARG BD_VERSION=1.0.3
ARG BD_CHECKSUM=1ef5dca818d7e81574df9e9f9fc2a16ab711da09b0fa7b822ae162d9a81c8912
RUN curl -fsSL "https://github.com/gastownhall/beads/releases/download/v${BD_VERSION}/beads_${BD_VERSION}_linux_amd64.tar.gz" \
        -o /tmp/bd.tar.gz \
    && echo "${BD_CHECKSUM}  /tmp/bd.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/bd.tar.gz -C /tmp \
    && cp /tmp/bd /usr/local/bin/bd \
    && rm -rf /tmp/bd.tar.gz /tmp/bd

# helix
ARG HELIX_VERSION=25.07.1
ARG HELIX_CHECKSUM=3f08e63ecd388fff657ad39722f88bb03dcf326f1f2da2700d99e1dc40ab2e8b
RUN curl -fsSL "https://github.com/helix-editor/helix/releases/download/${HELIX_VERSION}/helix-${HELIX_VERSION}-x86_64-linux.tar.xz" \
        -o /tmp/helix.tar.xz \
    && echo "${HELIX_CHECKSUM}  /tmp/helix.tar.xz" | sha256sum -c - \
    && tar -xJf /tmp/helix.tar.xz -C /tmp \
    && cp /tmp/helix-${HELIX_VERSION}-x86_64-linux/hx /usr/local/bin/hx \
    && mkdir -p /usr/local/share/helix \
    && cp -r /tmp/helix-${HELIX_VERSION}-x86_64-linux/runtime /usr/local/share/helix/runtime \
    && rm -rf /tmp/helix.tar.xz /tmp/helix-${HELIX_VERSION}-x86_64-linux

# Go toolchain (for carrel build)
ARG GO_VERSION=1.26.2
ARG GO_CHECKSUM=990e6b4bbba816dc3ee129eaeaf4b42f17c2800b88a2166c265ac1a200262282
RUN curl -fsSL "https://dl.google.com/go/go${GO_VERSION}.linux-amd64.tar.gz" \
        -o /tmp/go.tar.gz \
    && echo "${GO_CHECKSUM}  /tmp/go.tar.gz" | sha256sum -c - \
    && tar -xzf /tmp/go.tar.gz -C /usr/local \
    && rm -f /tmp/go.tar.gz

# Layer 3: pip-installed tools

ARG PYRIGHT_VERSION=1.1.409
RUN pip3 install --break-system-packages pyright==${PYRIGHT_VERSION} \
        requests \
        markdown \
        markdownify \
        google-api-python-client \
        google-auth \
        cairosvg \
        numpy

# Rust toolchain
ENV RUSTUP_HOME=/usr/local/rustup \
    CARGO_HOME=/usr/local/cargo \
    PATH=/usr/local/cargo/bin:/usr/local/go/bin:/home/dev/go/bin:${PATH}
ENV HELIX_RUNTIME=/usr/local/share/helix/runtime
RUN curl --proto '=https' --tlsv1.2 -sSf https://sh.rustup.rs \
    | sh -s -- -y --no-modify-path --default-toolchain stable --profile minimal \
    && chmod -R a+w /usr/local/rustup /usr/local/cargo

# wezterm terminfo — needed by hx, etc. when TERM=wezterm
COPY omp/config/wezterm/wezterm.terminfo /tmp/wezterm.terminfo
RUN tic -x -o /usr/share/terminfo /tmp/wezterm.terminfo \
    && rm /tmp/wezterm.terminfo

# Layer 4: Source builds (bun + OMP)

ARG BUN_VERSION=1.3.14
RUN curl -fsSL https://bun.sh/install | BUN_INSTALL=/usr/local bash -s "bun-v${BUN_VERSION}"

ARG OMP_VERSION=15.7.3
RUN BUN_INSTALL=/usr/local bun install -g @oh-my-pi/pi-coding-agent@${OMP_VERSION}

# Vega-Lite → SVG rendering for documentation charts
ARG VEGA_CLI_VERSION=6.2.0
ARG VEGA_LITE_VERSION=6.4.3
RUN BUN_INSTALL=/usr/local bun install -g vega-cli@${VEGA_CLI_VERSION} vega-lite@${VEGA_LITE_VERSION}

# Carrel build environment (binary built at container startup via entrypoint)
ENV CGO_ENABLED=1 \
    GOPATH=/home/dev/go

# User setup
RUN userdel -r ubuntu 2>/dev/null || true \
    && groupadd -f -g 1000 dev \
    && useradd -m -u 1000 -g dev -s /usr/bin/zsh dev \
    && mkdir -p /home/dev \
    && chown 1000:1000 /home/dev

# OS tool configuration (image-baked defaults — carrel os-setup overwrites at runtime)
COPY --chown=dev:dev omp/config/zsh/p10k.zsh /home/dev/.p10k.zsh
COPY --chown=dev:dev omp/config/zsh/zshrc.zsh /home/dev/.zshrc
COPY --chown=dev:dev omp/config/zsh/zprofile.zsh /home/dev/.zprofile
COPY --chown=dev:dev omp/config/wezterm/wezterm.lua /home/dev/.config/wezterm/wezterm.lua
COPY --chown=dev:dev omp/config/wezterm/mux-server.lua /home/dev/.config/wezterm/wezterm.lua

COPY --chown=dev:dev omp/config/helix/config.toml /home/dev/.config/helix/config.toml

# Install powerlevel10k
RUN git clone --depth=1 https://github.com/romkatv/powerlevel10k.git /opt/powerlevel10k \
    && chown -R 1000:1000 /opt/powerlevel10k

# Fix ownership of build-time artifacts in /home/dev
RUN chown -R dev:dev /home/dev

# Entrypoint
COPY bin/entrypoint.sh /usr/local/bin/entrypoint.sh

ENV HOME=/home/dev
WORKDIR /workspace
ENTRYPOINT ["entrypoint.sh"]
