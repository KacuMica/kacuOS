# Stage 1: build. Has the full Rust toolchain, gcc (mlua vendors and
# compiles its own Lua interpreter), and everything needed to produce the
# binary. This stage is thrown away after build — its size doesn't matter.
FROM rust:1-slim-bookworm AS builder

RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /build
COPY kacuctl/ .
RUN cargo build --release

# Stage 2: runtime. Just the compiled binary and presets — no rustc, no
# gcc, no source code. This is what actually ships.
#
# Note: this base image is Debian, not Arch, so `pacman` isn't available
# here — `kacuctl install <preset>` will fail inside this container by
# design (kacuctl is an Arch-specific tool). What you CAN run here without
# installing Rust locally: `kacuctl list`, `kacuctl status`, `kacuctl plugin
# list/add/run-hook`, `kacuctl --help`. This still proves the build works
# and gives you a portable way to poke at kacuctl's non-pacman commands.
FROM debian:bookworm-slim

COPY --from=builder /build/target/release/kacuctl /usr/local/bin/kacuctl
COPY kacuctl/presets /etc/kacu/presets

ENTRYPOINT ["kacuctl"]
CMD ["--help"]
