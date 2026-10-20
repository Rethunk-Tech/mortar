---
kind: howto
title: "Play your modded games from Steam Deck Game Mode"
summary: "Set Mortar up once in Desktop Mode, then start each profile from Game Mode like any other game."
date: 2026-10-20
image: /img/deck.webp
---

Mortar runs on the Steam Deck as the x86_64 AppImage. You set it up once in Desktop Mode. After that, each profile you add to Steam plays from Game Mode with its own mods.

## Set it up in Desktop Mode

1. Switch to Desktop Mode and [download the AppImage](/download/#deck).
2. Mark it executable: right-click it, **Properties**, **Permissions**, then **Is executable**. Run it.
3. Pick your game. Mortar installs its mod loader and makes your first profile. Add your mods.

Use the AppImage rather than the Flatpak on the Deck: the Flatpak cannot add itself to Steam. The AppImage updates itself in place.

## Add a profile to Steam

Open the profile's menu and choose **Add this profile to Steam**. Close Steam first. The profile shows up in your library as a non-Steam game.

## Play from Game Mode

Back in Game Mode, start the profile like any other game. Mortar stays out of sight and asks only when something needs an answer. It closes with the game, so Steam counts your playtime.

A gamepad moves focus, confirms, goes back and plays, and the targets are larger, laid out for the Deck's 1280×800 screen.
