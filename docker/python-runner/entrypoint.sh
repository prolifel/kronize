#!/bin/sh
script=$(mktemp /tmp/script-XXXXXX.py)
cat > "$script"
exec python "$script"
