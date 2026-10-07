#!/bin/sh
# Usage: webkit-sandbox.sh APPDIR. Makes the bundled WebKitGTK launchable inside its own bubblewrap sandbox.
#
# `wails3 generate appimage` rewrites the helper directory baked into libwebkitgtk to a path relative to $APPDIR/usr
# ("././lib/x86_64-linux-gnu/webkitgtk-6.0/" on Debian and Ubuntu). WebKit hands that string to bwrap twice: as a bind
# destination, resolved against the sandbox root, and as the exec path, resolved against the working directory.
# Under /lib the destination cannot be created (the sandbox's /lib is a read-only bind of the host's), which kills the
# launch with "bwrap: Can't mkdir parents". So the helpers move to usr/lib/webkitgtk-6.0, the string becomes
# "././webkitgtk-6.0/" (a creatable top-level destination), and AppRun starts the program in $APPDIR/usr/lib, a directory
# WebKit binds into the sandbox at the same absolute path, so the relative exec path resolves there too.
set -eu
appdir=$1
usr=$appdir/usr
web=$(find "$usr" -name WebKitWebProcess -type f)
[ -n "$web" ] || { echo "no WebKitWebProcess under $usr" >&2; exit 1; }
dir=$(dirname "$web")
rel=${dir#"$usr"/}
[ "$dir" = "$usr/lib/webkitgtk-6.0" ] || mv "$dir" "$usr/lib/webkitgtk-6.0"
for lib in "$usr"/lib/libwebkit*.so*; do
  [ -f "$lib" ] && [ ! -L "$lib" ] || continue
  OLD="././$rel/" perl -0pi -e 's{\Q$ENV{OLD}\E\0}{my $n = "././webkitgtk-6.0/\0"; $n . "\0" x (length($&) - length $n)}e' "$lib"
  grep -qF '././webkitgtk-6.0/' "$lib" || { echo "$lib does not name ././$rel/" >&2; exit 1; }
done

# The AppImageKit AppRun binary would chdir to $APPDIR/usr before starting the program, so AppRun sets the same environment itself.
cat >"$appdir/AppRun" <<'APPRUN'
#!/bin/sh
set -e

APPDIR="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
. "$APPDIR/glibc-check.sh"
OWD="${OWD:-$PWD}"
export OWD
usr="$APPDIR/usr"
export PATH="$usr/bin:$PATH"
export LD_LIBRARY_PATH="$usr/lib:$usr/lib/x86_64-linux-gnu:$usr/lib/aarch64-linux-gnu:$usr/lib64${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"
export XDG_DATA_DIRS="$usr/share:${XDG_DATA_DIRS:-/usr/local/share:/usr/share}"
export GSETTINGS_SCHEMA_DIR="$usr/share/glib-2.0/schemas"
cd "$usr/lib"
exec "$usr/bin/mortar" "$@"
APPRUN
