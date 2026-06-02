# Build a Turing-compatible CUDA v12 backend for ollama.
#
# Problem: ollama v0.30.0 ships a CUDA 12.8-compiled libggml-cuda.so whose
# SASS binary for sm_75 (Turing / GTX 1660 Ti) contains an invalid kernel
# image in ggml_cuda_kernel_can_use_pdl.  The CUDA runtime falls back to CPU.
#
# Fix: recompile only the v12 backend with CUDA 12.2 (matching driver 535)
# and emit PTX instead of SASS for sm_75 ("75-virtual").  The driver's JIT
# compiler handles final compilation correctly — a one-time ~2-3 s cost at
# first model load.
#
# Usage:
#   docker build -t ollama/ollama:turing -f tools/ollama-turing.Dockerfile .
#
# Then switch the ollama service in docker-compose.yml to image: ollama/ollama:turing
#
# Remove this file once upstream ollama ships a fixed CUDA backend.

ARG CMAKEVERSION=3.31.2
ARG NINJAVERSION=1.12.1

FROM nvidia/cuda:12.2.2-devel-rockylinux8 AS builder

RUN dnf install -y epel-release \
    && dnf install -y ccache gcc-toolset-11-gcc gcc-toolset-11-gcc-c++ \
                      gcc-toolset-11-binutils unzip git \
    && dnf clean all
ENV PATH=/opt/rh/gcc-toolset-11/root/usr/bin:$PATH

ARG CMAKEVERSION
RUN curl -fsSL https://github.com/Kitware/CMake/releases/download/v${CMAKEVERSION}/cmake-${CMAKEVERSION}-linux-x86_64.tar.gz | \
    tar xz -C /usr/local --strip-components 1

ARG NINJAVERSION
RUN curl -fsSL -o /tmp/ninja.zip https://github.com/ninja-build/ninja/releases/download/v${NINJAVERSION}/ninja-linux.zip \
    && unzip /tmp/ninja.zip -d /usr/local/bin && rm /tmp/ninja.zip
ENV CMAKE_GENERATOR=Ninja

# CUDA 12.2 toolkit (driver 535 provides CUDA 12.2 runtime)
RUN dnf config-manager --add-repo https://developer.download.nvidia.com/compute/cuda/repos/rhel8/x86_64/cuda-rhel8.repo \
    && dnf install -y cuda-toolkit-12-2 \
    && dnf clean all

# Copy ollama's build glue (FetchContent pulls the pinned llama.cpp commit)
COPY LLAMA_CPP_VERSION /src/
COPY llama/ /src/llama/
WORKDIR /src

# Architecture list:
#   50-61: PTX-only (virtual) — these ancient GPUs rely on driver JIT anyway
#   70:    SASS + PTX    — Volta, no known issues
#   75:    PTX-only      — Turing fix: let driver JIT handle sm_75
#   80-90a:SASS + PTX    — Ampere/Ada, no known issues
#   120 omitted — unsupported by CUDA 12.2 toolkit (needs >= 12.8)
RUN cmake -S llama/server -B build \
    -DGGML_CUDA=ON \
    -DOLLAMA_RUNNER_DIR=cuda_v12 \
    -DCMAKE_CUDA_ARCHITECTURES="75" \
    -DCMAKE_CUDA_FLAGS="-Wno-deprecated-gpu-targets" \
    -DBUILD_SHARED_LIBS=ON \
    && cmake --build build -- -j $(nproc) \
    && cmake --install build --component llama-server --prefix /dist

FROM ollama/ollama:0.30.0
# Replace only the CUDA v12 backend shared library — keep everything else
# (ollama binary, CPU backends, CUDA runtime libs, vulkan, etc.) untouched.
COPY --from=builder /dist/lib/ollama/cuda_v12/libggml-cuda.so* /usr/lib/ollama/cuda_v12/