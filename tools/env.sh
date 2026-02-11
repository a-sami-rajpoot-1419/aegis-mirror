#!/usr/bin/env bash

# Mirror Vault local (WSL2) toolchain paths (no sudo installs)
export GOROOT="$HOME/.local/opt/go"
export GOPATH="$HOME/.local/share/go"
export GOBIN="$HOME/.local/bin"

export PATH="$HOME/.local/bin:$HOME/.local/opt/go/bin:$HOME/.local/opt/node/bin:$PATH"

# Keep CLI tools non-interactive during scripted runs
export CI="1"
export IGNITE_TELEMETRY_DISABLED="1"

# NOTE:
# This file is intended to be sourced (not executed) in an interactive shell.
# Do not enable `set -e` / `set -u` here; it can unexpectedly terminate the
# user's terminal session when a later command fails.
