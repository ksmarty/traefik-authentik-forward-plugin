#!/bin/sh
#
# Print the plugin hash Traefik will compare against, and warn when pinning it
# is a trap.
#
# With `hash` configured, Traefik downloads the plugin archive from the registry
# and compares it locally (pkg/plugins/manager.go, InstallPlugin). The value it
# compares against is the sha256 of the bytes the registry serves.
#
# Those bytes are NOT stable: the registry repacks archives for the same module
# version. Measured on v1.0.2, ten minutes apart, same content (65 zip entries,
# byte-identical files) but two different zips: 130297 and 131610 bytes. The Go
# module zip on proxy.golang.org is stable and matches the larger one.
#
# So a pinned hash keeps working only while the registry serves the packaging it
# was computed from. This script prints the current value so a pin can at least be
# refreshed deliberately, and flags the disagreement so the cause is visible.
#
# usage: scripts/plugin-hash.sh [<version>]

set -eu

MODULE="${MODULE:-github.com/ksmarty/traefik-authentik-forward-plugin}"
VERSION="${1:-${VERSION:-}}"

if [ -z "$VERSION" ]; then
	VERSION="$(git describe --tags --abbrev=0 2>/dev/null || true)"
fi

if [ -z "$VERSION" ]; then
	echo "usage: $0 <version> (for example: $0 v1.0.4)" >&2
	exit 2
fi

registry="https://plugins.traefik.io/public/download/$MODULE/$VERSION"
proxy="https://proxy.golang.org/$MODULE/@v/$VERSION.zip"

tmp="$(mktemp)"
trap 'rm -f "$tmp"' EXIT INT TERM

sha256_of() {
	if command -v sha256sum >/dev/null 2>&1; then
		sha256sum "$1" | cut -d' ' -f1
	else
		shasum -a 256 "$1" | cut -d' ' -f1
	fi
}

# hash_of prints the sha256 of the body, and nothing when the fetch fails.
hash_of() {
	if curl -fsSL --max-time 60 -o "$tmp" "$1" 2>/dev/null; then
		sha256_of "$tmp"
	fi
}

registry_hash="$(hash_of "$registry" || true)"
proxy_hash="$(hash_of "$proxy" || true)"

hash="${registry_hash:-$proxy_hash}"

if [ -z "$hash" ]; then
	echo "ERROR: $VERSION could not be downloaded from the registry or the module proxy." >&2
	echo "Has the tag been pushed, and has the module reached proxy.golang.org?" >&2
	exit 1
fi

printf '%s\n' "$hash"

cat >&2 <<EOF

$VERSION — Traefik will compare against the value above.

  # CLI
  --experimental.plugins.authentik-forward.modulename=$MODULE
  --experimental.plugins.authentik-forward.version=$VERSION
  --experimental.plugins.authentik-forward.hash=$hash

  # file
  experimental:
    plugins:
      authentik-forward:
        moduleName: "$MODULE"
        version: "$VERSION"
        hash: "$hash"
EOF

if [ -n "$registry_hash" ] && [ -n "$proxy_hash" ] && [ "$registry_hash" != "$proxy_hash" ]; then
	cat >&2 <<EOF

WARNING: the registry archive differs from the Go module zip for this version.
  registry:             $registry_hash
  proxy.golang.org zip: $proxy_hash

The registry repacks archives, so a pinned hash matches only the packaging it was
computed from. If Traefik logs

  invalid hash for plugin $MODULE, expected <pinned>, got <downloaded>

the registry has since repacked. Re-running this script gives the value that
matches right now; the stable fix is a local plugin (experimental.localPlugins),
which has no archive, no hash and no catalogue lookup.
EOF
fi
