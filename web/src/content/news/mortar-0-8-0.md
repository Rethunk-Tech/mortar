---
kind: release
version: 0.8.0
title: "Bethesda games, Minecraft and a see-through window"
summary: "Mortar sets up Skyrim, Fallout and Starfield plugin lists and script extenders, runs Minecraft profiles with Fabric, Forge or NeoForge, translates Browse descriptions, says why a plugin won't load, and keeps the files you edit in a game folder."
highlights:
  - area: Browse
    title: Read a mod's description in your language
    text: A description in Browse has a Translate button. Press it to read the description in Mortar's language, and Show original to switch back.
    image: /img/translate-0-8-0.webp
  - area: Load order
    title: Plugins that won't load say why
    text: A BepInEx plugin that will not load stays in the load order with the reason on its row, such as a missing dependency or a plugin it clashes with.
    image: /img/load-order-0-8-0.webp
---

Mortar now sets up Skyrim Special Edition, Skyrim, Fallout 4, Fallout: New Vegas and Starfield. It writes the plugin list in LOOT's default order, keeps archives and plugins whole, and starts the script extender when you have one installed. Minecraft profiles are Java instances with Fabric, Forge or NeoForge.

The window is see-through too: the desktop shows behind Mortar, and on GNOME 51 and later it's blurred.

A file you edit in a game folder now survives its mod's updates and rollbacks. Mortar sets it aside only when another mod takes the file over.

Pairing over the LAN uses a new exchange, so update Mortar on both computers before you pair or send a profile.
