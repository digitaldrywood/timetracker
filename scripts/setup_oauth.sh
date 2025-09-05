#!/bin/bash

echo "=== Google OAuth2 Setup Instructions ==="
echo ""
echo "To use this time tracker, you need to configure your Google Cloud OAuth2 client:"
echo ""
echo "1. Go to: https://console.cloud.google.com/apis/credentials"
echo ""
echo "2. Find your OAuth 2.0 Client ID (the one used in .local/credentials.json)"
echo ""
echo "3. Click on it to edit"
echo ""
echo "4. Under 'Authorized redirect URIs', add:"
echo "   http://localhost:8080/callback"
echo ""
echo "5. Click 'Save'"
echo ""
echo "6. Now you can run: make auth"
echo ""
echo "Press Enter when you've completed these steps..."
read

echo "Testing authentication..."
./bin/timetracker -summary