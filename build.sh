#!/bin/bash

# 编译 mydocker 项目
set -e

echo "Building mydocker..."
go build -o mydocker ./cmd/mydocker/
echo "Build complete: mydocker"
