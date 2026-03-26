---
title: Install
description: Installation instructions for brighten
---


This template is currently setup to build and deploy to homebrew and AUR. 

Because that is what I use so that that is what I have capacity to test at the moment. 

This package `brighten` is built and actually deployed to homebrew and aur to demonstrate the usage of the deployment scripts. 

## Homebrew
```bash
brew install imdevan/brighten/brighten
```

## Arch (AUR)
```bash
yay -S brighten
```

## GitHub Release

Download the latest binary for your platform from the [releases page](https://github.com/imdevan/brighten/releases).

```bash
# Linux (amd64)
curl -L https://github.com/imdevan/brighten/releases/latest/download/brighten-linux-amd64.tar.gz | tar -xz
sudo mv brighten-linux-amd64 /usr/local/bin/brighten
```

```bash
# macOS (Apple Silicon)
curl -L https://github.com/imdevan/brighten/releases/latest/download/brighten-darwin-arm64.tar.gz | tar -xz
sudo mv brighten-darwin-arm64 /usr/local/bin/brighten
```

```bash
# macOS (Intel)
curl -L https://github.com/imdevan/brighten/releases/latest/download/brighten-darwin-amd64.tar.gz | tar -xz
sudo mv brighten-darwin-amd64 /usr/local/bin/brighten
```

## Manual
```bash
gh repo clone imdevan/brighten
cd brighten
just build
sudo just install
```
