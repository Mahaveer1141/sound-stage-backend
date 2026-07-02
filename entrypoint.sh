#!/bin/sh
set -e

echo "Running database migrations..."
goose -dir ./migrations up

exec "$@"
