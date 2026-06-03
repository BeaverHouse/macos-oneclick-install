#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

out="${1:-./austinhome}"
GOCACHE="${GOCACHE:-/tmp/austinhome-go-cache}" go build -o "$out" .

cat <<EOF

============================================================
 AUSTINHOME BUILD COMPLETE
============================================================

Output:
  $out

Mac mini update rule:
  1. Put this file at:
       ~/Downloads/austinhome

  2. Run that file once.
     No extra command is required.

Why:
  Just copying the file does not update the boot-time reinstall
  agent. Running ~/Downloads/austinhome once refreshes the launch
  binary and agent from the Downloads SSOT.

============================================================

EOF
