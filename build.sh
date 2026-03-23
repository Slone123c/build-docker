#!/bin/bash

# 编译 mydocker 项目
set -e

echo "Building mydocker..."
CGO_ENABLED=1 go build -o mydocker ./cmd/mydocker/
echo "Build complete: mydocker"
