# Sourced by AppRun before Mortar starts. The bundled WebKitGTK and GTK 4 libraries need the host's glibc 2.38 or
# newer; on an older system they fail with a loader error the user never sees when starting from a file manager.
mortar_need=2.38
mortar_have="$(getconf GNU_LIBC_VERSION 2>/dev/null | awk '{print $2}')"
if [ -n "$mortar_have" ] && [ "$(printf '%s\n%s\n' "$mortar_need" "$mortar_have" | sort -V | head -n1)" != "$mortar_need" ]; then
  mortar_msg="This Mortar AppImage needs a newer Linux (glibc $mortar_need or newer, as in Ubuntu 24.04, Debian 13 or Fedora 39); this system has glibc $mortar_have. Install the Mortar Flatpak instead, which brings everything it needs."
  if command -v zenity >/dev/null 2>&1; then
    zenity --error --title=Mortar --no-wrap --text="$mortar_msg" 2>/dev/null || true
  elif command -v kdialog >/dev/null 2>&1; then
    kdialog --title Mortar --error "$mortar_msg" 2>/dev/null || true
  elif command -v notify-send >/dev/null 2>&1; then
    notify-send Mortar "$mortar_msg" 2>/dev/null || true
  fi
  echo "$mortar_msg" >&2
  exit 1
fi
