#!/bin/bash

echo "=== Enable Google Sheets API ==="
echo ""
echo "You need to enable the Google Sheets API for your project."
echo ""
echo "Opening the Google Cloud Console in your browser..."
echo ""

# Open the API enablement page
URL="https://console.developers.google.com/apis/api/sheets.googleapis.com/overview?project=517658468057"
echo "If the browser doesn't open, go to:"
echo "$URL"
echo ""

# Try to open browser
if [[ "$OSTYPE" == "darwin"* ]]; then
    open "$URL"
elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
    xdg-open "$URL"
fi

echo "Steps to enable the API:"
echo "1. Click the 'ENABLE' button on the page"
echo "2. Wait a few seconds for it to activate"
echo "3. Come back here and press Enter"
echo ""
read -p "Press Enter after you've enabled the API..."

echo ""
echo "Waiting 10 seconds for API to propagate..."
sleep 10

echo ""
echo "Testing authentication..."
make auth