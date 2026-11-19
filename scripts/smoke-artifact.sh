#!/usr/bin/env bash
# Smoke test of the published artifact: what `go get` downloads is the git tree, so take a `git archive`
# of the repository (the working tree incl. uncommitted, non-ignored files; in CI that is the commit), put
# it next to an empty module that requires it via `replace`, and run the README "Testing" example there
# offline (GOPROXY=off: the module must need nothing but the standard library).
set -euo pipefail

root=$(git rev-parse --show-toplevel)
module=$(cd "$root" && go list -m)
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT

# A tree object of the working tree, via a throwaway index: the real index and HEAD stay untouched.
export GIT_INDEX_FILE="$tmp/index"
git -C "$root" add -A
tree=$(git -C "$root" write-tree)
unset GIT_INDEX_FILE

mkdir "$tmp/module" "$tmp/app"
git -C "$root" archive --format=tar "$tree" | tar -x -C "$tmp/module"
echo "archived $(find "$tmp/module" -type f | wc -l | tr -d ' ') files of $module"
(cd "$tmp/module" && go mod verify)

awk '/^## Testing$/ { section = 1; next } section && /^```go$/ { code = 1; next } code && /^```$/ { exit } code { print }' \
  "$tmp/module/README.md" > "$tmp/app/main.go"
if [ ! -s "$tmp/app/main.go" ]; then
  echo "README.md has no '## Testing' section with a go example" >&2
  exit 1
fi

cat > "$tmp/app/go.mod" <<MOD
module smokeapp

go 1.22

require ${module} v3.999.0

replace ${module} => ../module
MOD

(cd "$tmp/app" && GOWORK=off GOFLAGS=-mod=mod GOPROXY=off go run .)
echo 'README "Testing" example: ok'
