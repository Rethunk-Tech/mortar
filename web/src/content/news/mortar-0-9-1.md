---
kind: release
version: 0.9.1
title: "Fixes for Cyberpunk 2077 and mods in other games"
summary: "Mods in games other than Stardew Valley keep their own names and requirements, mods split by an earlier release load again, and updates keep what you set on a mod."
highlights:
  - area: Config
    title: Settings say whose they are
    text: The Config page names the mod whose settings are open, even when your filter hides that mod from the list.
    image: /img/config-named-0-9-1.webp
---

This release fixes what testers found in 0.9.0.

In games other than Stardew Valley, shares, packs, collections and saves no longer name Stardew Valley mods as requirements, and a mod already saved under a Stardew Valley name gets its own name back, even offline. Mods an earlier release split into one entry per file are joined back together, so they load again. Updating a mod keeps its note, tags, pin and whether it is switched off.

For Cyberpunk 2077, loose scripts from two mods no longer overwrite each other, a modlist.txt you wrote yourself is set aside and put back instead of deleted, and the game no longer starts with an old archive order when Mortar cannot write the new one.
