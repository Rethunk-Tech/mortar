#!/bin/sh
# Installed as the linuxdeploy that `wails3 generate appimage` runs. It packs the AppDir once itself, and the Mortar
# task packs it again after adding the glibc check, so that first pack is skipped; the file wails3 then moves is a stub.
case " $* " in
  *" --output "*) : >"$OUTPUT"; exit 0 ;;
esac
exec "$LINUXDEPLOY_REAL" "$@"
