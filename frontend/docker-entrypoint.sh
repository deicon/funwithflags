#!/bin/sh
set -e

# Generate runtime config
cat > /usr/share/nginx/html/config.js << EOF
window.APP_CONFIG = {
  apiBase: "${API_BASE:-}"
};
EOF

# Start nginx
exec nginx -g "daemon off;"
