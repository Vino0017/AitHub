#!/bin/bash

# Monitor ClawHub import progress
# Usage: ./monitor_import.sh

source .env

echo "=== ClawHub Import Monitor ==="
echo ""

while true; do
    clear
    echo "=== ClawHub Import Monitor ==="
    echo "Time: $(date '+%Y-%m-%d %H:%M:%S')"
    echo ""

    # Get total count
    TOTAL=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM skills WHERE namespace_id = (SELECT id FROM namespaces WHERE name = 'clawhub') AND framework = 'openclaw';")

    # Get recent imports (last 5 minutes)
    RECENT=$(psql "$DATABASE_URL" -t -c "SELECT COUNT(*) FROM skills WHERE namespace_id = (SELECT id FROM namespaces WHERE name = 'clawhub') AND framework = 'openclaw' AND created_at > NOW() - INTERVAL '5 minutes';")

    # Get last import time
    LAST=$(psql "$DATABASE_URL" -t -c "SELECT MAX(created_at) FROM skills WHERE namespace_id = (SELECT id FROM namespaces WHERE name = 'clawhub') AND framework = 'openclaw';")

    echo "Total OpenClaw Skills: $TOTAL"
    echo "Imported (last 5 min): $RECENT"
    echo "Last Import: $LAST"
    echo ""

    # Show recent imports
    echo "━━━ Recent Imports ━━━"
    psql "$DATABASE_URL" -c "SELECT name, created_at FROM skills WHERE namespace_id = (SELECT id FROM namespaces WHERE name = 'clawhub') AND framework = 'openclaw' ORDER BY created_at DESC LIMIT 10;" 2>/dev/null

    echo ""
    echo "Press Ctrl+C to exit"
    sleep 5
done
