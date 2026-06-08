#!/bin/bash

# Card Onboarding Workers Deployment Script

set -e

echo "Deploying Card Onboarding Workers..."

# Get version
VERSION=$(cat VERSION)
echo "Deploying version: $VERSION"

# Build services
echo "Building services..."
make build

# Deploy services
echo "Deploying services..."
# Add deployment commands here

echo "Deployment complete!"
