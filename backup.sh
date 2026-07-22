#!/bin/bash

set -e

BACKUP_DIR="$HOME/dock/restore/$(date +%Y%m%d-%H%M%S)"

mkdir -p "$BACKUP_DIR"

echo "A criar backup em:"
echo "  $BACKUP_DIR"
echo

cp -a "$HOME/dock/cmd"              "$BACKUP_DIR/" 2>/dev/null || true
cp -a "$HOME/dock/internal"         "$BACKUP_DIR/" 2>/dev/null || true
cp -a "$HOME/dock/web"              "$BACKUP_DIR/" 2>/dev/null || true
cp -a "$HOME/dock/config"           "$BACKUP_DIR/" 2>/dev/null || true

cp "$HOME/dock/go.mod"              "$BACKUP_DIR/" 2>/dev/null || true
cp "$HOME/dock/go.sum"              "$BACKUP_DIR/" 2>/dev/null || true
cp "$HOME/dock/docker-compose.yml"  "$BACKUP_DIR/" 2>/dev/null || true

echo "Backup concluído."
echo
echo "Restauro:"
echo "  cp -a $BACKUP_DIR/* $HOME/dock/"
