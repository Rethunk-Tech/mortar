#ifndef MORTAR_FROSTED_WAYLAND_H
#define MORTAR_FROSTED_WAYLAND_H

#define GDK_WINDOWING_WAYLAND
#include <gtk/gtk.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>
#include <wayland-client.h>

#ifdef FROSTED_GTK3
#include <gdk/gdkwayland.h>
#else
#include <gdk/wayland/gdkwayland.h>
#endif

#define EXT_BACKGROUND_EFFECT_MANAGER_V1_DESTROY 0
#define EXT_BACKGROUND_EFFECT_MANAGER_V1_GET_BACKGROUND_EFFECT 1
#define EXT_BACKGROUND_EFFECT_SURFACE_V1_DESTROY 0
#define EXT_BACKGROUND_EFFECT_SURFACE_V1_SET_BLUR_REGION 1
#define EXT_BACKGROUND_EFFECT_MANAGER_V1_CAPABILITY_BLUR 1

struct ext_background_effect_manager_v1;
struct ext_background_effect_surface_v1;

static const struct wl_interface ext_background_effect_surface_v1_interface;
static const struct wl_interface ext_background_effect_manager_v1_interface;

static const struct wl_interface *manager_types[] = {
    NULL,
    &ext_background_effect_surface_v1_interface,
    &wl_surface_interface,
};

static const struct wl_message manager_requests[] = {
    {"destroy", "", manager_types + 0},
    {"get_background_effect", "no", manager_types + 1},
};

static const struct wl_message manager_events[] = {
    {"capabilities", "u", manager_types + 0},
};

static const struct wl_interface ext_background_effect_manager_v1_interface = {
    "ext_background_effect_manager_v1",
    1,
    2,
    manager_requests,
    1,
    manager_events,
};

static const struct wl_interface *surface_effect_types[] = {
    NULL,
    &wl_region_interface,
};

static const struct wl_message surface_effect_requests[] = {
    {"destroy", "", surface_effect_types + 0},
    {"set_blur_region", "?o", surface_effect_types + 1},
};

static const struct wl_interface ext_background_effect_surface_v1_interface = {
    "ext_background_effect_surface_v1", 1, 2, surface_effect_requests, 0, NULL,
};

struct frosted_state {
  struct wl_display *display;
  struct wl_surface *surface;
  struct wl_compositor *compositor;
  struct ext_background_effect_manager_v1 *manager;
  struct ext_background_effect_surface_v1 *effect;
  uint32_t caps;
};

static void registry_global(void *data, struct wl_registry *registry,
                            uint32_t name, const char *iface,
                            uint32_t version) {
  struct frosted_state *st = data;
  (void)version;
  if (strcmp(iface, "ext_background_effect_manager_v1") == 0) {
    st->manager = wl_registry_bind(
        registry, name, &ext_background_effect_manager_v1_interface, 1);
  } else if (strcmp(iface, "wl_compositor") == 0) {
    uint32_t bind_ver = version < 4 ? version : 4;
    st->compositor =
        wl_registry_bind(registry, name, &wl_compositor_interface, bind_ver);
  }
}

static void registry_global_remove(void *data, struct wl_registry *registry,
                                   uint32_t name) {
  (void)data;
  (void)registry;
  (void)name;
}

static const struct wl_registry_listener registry_listener = {
    .global = registry_global,
    .global_remove = registry_global_remove,
};

static void
manager_capabilities(void *data,
                     struct ext_background_effect_manager_v1 *manager,
                     uint32_t flags) {
  (void)manager;
  ((struct frosted_state *)data)->caps = flags;
}

struct ext_background_effect_manager_v1_listener {
  void (*capabilities)(void *data,
                       struct ext_background_effect_manager_v1 *manager,
                       uint32_t flags);
};

static const struct ext_background_effect_manager_v1_listener manager_listener =
    {
        .capabilities = manager_capabilities,
};

static struct wl_surface *frosted_wl_surface(void *gtk_window) {
#ifdef FROSTED_GTK3
  GdkWindow *gw = gtk_widget_get_window(GTK_WIDGET(gtk_window));
  if (gw == NULL) {
    return NULL;
  }
  return gdk_wayland_window_get_wl_surface(gw);
#else
  GdkSurface *gs = gtk_native_get_surface(GTK_NATIVE(gtk_window));
  if (gs == NULL) {
    return NULL;
  }
  return gdk_wayland_surface_get_wl_surface(gs);
#endif
}

static struct wl_display *frosted_wl_display(void *gtk_window) {
#ifdef FROSTED_GTK3
  GdkWindow *gw = gtk_widget_get_window(GTK_WIDGET(gtk_window));
  if (gw == NULL) {
    return NULL;
  }
  GdkDisplay *gd = gdk_window_get_display(gw);
  if (gd == NULL || !GDK_IS_WAYLAND_DISPLAY(gd)) {
    return NULL;
  }
  return gdk_wayland_display_get_wl_display(gd);
#else
  GdkSurface *gs = gtk_native_get_surface(GTK_NATIVE(gtk_window));
  if (gs == NULL) {
    return NULL;
  }
  GdkDisplay *gd = gdk_surface_get_display(gs);
  if (gd == NULL || !GDK_IS_WAYLAND_DISPLAY(gd)) {
    return NULL;
  }
  return gdk_wayland_display_get_wl_display(gd);
#endif
}

static void *frosted_bind(void *gtk_window) {
  struct frosted_state *st = calloc(1, sizeof(*st));
  if (st == NULL || gtk_window == NULL) {
    return st;
  }
  st->display = frosted_wl_display(gtk_window);
  st->surface = frosted_wl_surface(gtk_window);
  if (st->display == NULL || st->surface == NULL) {
    return st;
  }
  struct wl_registry *registry = wl_display_get_registry(st->display);
  if (registry == NULL) {
    return st;
  }
  wl_registry_add_listener(registry, &registry_listener, st);
  wl_display_roundtrip(st->display);
  if (st->manager != NULL) {
    wl_proxy_add_listener((struct wl_proxy *)st->manager,
                          (void (**)(void))&manager_listener, st);
    wl_display_roundtrip(st->display);
  }
  wl_registry_destroy(registry);
  return st;
}

static void frosted_ensure_effect(struct frosted_state *st) {
  if (st == NULL || st->effect != NULL || st->manager == NULL ||
      st->surface == NULL) {
    return;
  }
  if ((st->caps & EXT_BACKGROUND_EFFECT_MANAGER_V1_CAPABILITY_BLUR) == 0) {
    return;
  }
  st->effect =
      (struct ext_background_effect_surface_v1 *)wl_proxy_marshal_flags(
          (struct wl_proxy *)st->manager,
          EXT_BACKGROUND_EFFECT_MANAGER_V1_GET_BACKGROUND_EFFECT,
          &ext_background_effect_surface_v1_interface,
          wl_proxy_get_version((struct wl_proxy *)st->manager), 0, NULL,
          st->surface);
}

static void frosted_set_region(void *p, int width, int height) {
  struct frosted_state *st = p;
  if (st == NULL || st->compositor == NULL || width <= 0 || height <= 0) {
    return;
  }
  frosted_ensure_effect(st);
  if (st->effect == NULL) {
    return;
  }
  struct wl_region *region = wl_compositor_create_region(st->compositor);
  if (region == NULL) {
    return;
  }
  wl_region_add(region, 0, 0, width, height);
  wl_proxy_marshal_flags((struct wl_proxy *)st->effect,
                         EXT_BACKGROUND_EFFECT_SURFACE_V1_SET_BLUR_REGION, NULL,
                         0, 0, region);
  wl_region_destroy(region);
  wl_surface_commit(st->surface);
}

static void frosted_clear(void *p) {
  struct frosted_state *st = p;
  if (st == NULL || st->effect == NULL || st->surface == NULL) {
    return;
  }
  // NULL region removes the effect (protocol); a leftover region would blur a
  // solid window.
  wl_proxy_marshal_flags((struct wl_proxy *)st->effect,
                         EXT_BACKGROUND_EFFECT_SURFACE_V1_SET_BLUR_REGION, NULL,
                         0, 0, NULL);
  wl_surface_commit(st->surface);
}

#endif
