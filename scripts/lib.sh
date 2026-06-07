#!/usr/bin/env bash

# Shared setup for repository-local scripts. This file is sourced, not executed.
repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$repo_root"
