# Nexus Mods application: registering Mortar

Everything needed to submit Mortar to Nexus Mods for registration. Status: draft, not yet submitted.

## How Nexus handles this (researched 2026-10-04)

- Nexus's API acceptable use policy ([help.nexusmods.com/article/114](https://help.nexusmods.com/article/114-api-acceptable-use-policy)) says to register an app: email support@nexusmods.com with a testing build, then submit the application name, description and logo, and receive a "slug" for SSO. Requests must carry `Application-Name` and `Application-Version` headers that are accurate and not blank or impersonating another app; it prohibits bulk fetching to rehost data and storing users' keys on your own server or using them without the user initiating the action. The article itself does not mention OAuth, redirect URIs or nxm://.
- Original API announcement ([nexusmods.com/news/13921](https://www.nexusmods.com/news/13921)): contact the Community Managers to get an app approved; only approved apps can use SSO; users can still paste a personal API key.
- Legacy SSO is a websocket flow ([Nexus-Mods/sso-integration-demo](https://github.com/Nexus-Mods/sso-integration-demo)): connect to `wss://sso.nexusmods.com`, send `{id, token, protocol: 2}`, open `https://www.nexusmods.com/sso?id=<uuid>&application=<slug>`, receive the user's API key over the socket. It needs a slug only Nexus staff can generate and involves no redirect URI.
- Newer OAuth2 (PKCE) flow: [Nexus-Mods/oauth2-demo-app](https://github.com/Nexus-Mods/oauth2-demo-app) shows a public-client PKCE flow with a custom protocol or localhost redirect. Vortex uses it against `https://users.nexusmods.com/oauth` (userinfo at `/oauth/userinfo`), client id `vortex_loopback`, scope `openid profile email`, redirect `http://127.0.0.1:<port>`; its source keeps `nxm://oauth/callback` only as a commented-out earlier value ([Vortex constants.ts](https://github.com/Nexus-Mods/Vortex/blob/master/src/renderer/src/extensions/nexus_integration/constants.ts)). Third-party clients (for example NexTwist) still use `nxm://oauth/callback` as their redirect.

Not verified: whether Nexus currently issues OAuth client ids to third parties, the scope list for such clients, and whether a custom-scheme redirect such as `nxm://oauth/callback` is accepted for a new client. No public page documents client registration; this has to be asked (see the support message below). Which flow to request (legacy SSO slug versus OAuth client) is therefore a question in the application, not an assumption. Both paths are built: the `nexussso.ClientID` ldflag selects OAuth PKCE, `nexussso.Slug` alone selects legacy SSO, and with neither set sign-in is hidden (see docs/architecture.md § Nexus).

## Application

- Application name: Mortar (consistent across versions; sent as `Application-Name: Mortar`)
- Proposed slug: `mortar`
- One-line description: Mortar is a desktop mod manager for Stardew Valley (Windows and Linux) that installs mods into per-profile sets, launches the game with SMAPI and shares profiles.
- Longer description: Mortar discovers the user's Stardew Valley install, installs SMAPI, keeps a separate mod set per profile and launches the game from the chosen profile. For Nexus it lets the user sign in, see which of their mods have updates, track and endorse mods, and download files they asked for. Mortar never re-hosts mod files; every download comes from Nexus's own servers (or the mod's other source).
- Website / repository: https://github.com/Rethunk-AI/mortar
- Logo: supply the app icon from the repository (confirm file and a 512x512 export before sending).
- Testing build: attach the current release build from the GitHub releases page.
- Contact: the maintainer through security@rethunk.tech (the project address that receives mail).

## Authentication requested

Preferred: OAuth2 authorization code with PKCE as a public client (no client secret on the user's machine), because Mortar is a desktop app.
Fallback: legacy SSO with slug `mortar`.

- Redirect URI: `nxm://oauth/callback` (Mortar already registers as the system `nxm://` handler; the callback would arrive as `nxm://oauth/callback?code=...&state=...`). Alternative if Nexus requires it: loopback `http://127.0.0.1:<random port>`, as Vortex does. Nexus decides which it allows.
- Scopes: the minimum needed to identify the user and call the API as them (Vortex uses `openid profile email`). Mortar needs no scope beyond reading the user's account, mod and file data, tracked mods, and endorse/abstain on request. Ask Nexus for the exact scope names.
- Today Mortar uses the user's personal API key, held in the OS keyring and sent only from the user's own machine, never to a Mortar server (Mortar has none). Moving to OAuth/SSO is what registration is for.

## What Mortar calls

REST v1 (`https://api.nexusmods.com`): `users/validate.json`, `games/{game}.json`, `games/{game}/mods/{id}.json`, `mods/{id}/files.json`, `mods/{id}/changelogs.json`, `mods/{id}/files/{file}/download_link.json`, `mods/{id}/endorse.json`, `mods/{id}/abstain.json`, `user/tracked_mods.json`; and v2 GraphQL (`/v2/graphql`) for batched update checks. Game: `stardewvalley` only.

## How downloads start

Downloads start only from Nexus's own Mod Manager Download button (an nxm:// link carrying Nexus's key and expiry, which Mortar then redeems) or, for Premium accounts, a Premium API download link requested because the user asked for that mod in Mortar. The browser extension only reads and marks Nexus pages and relays clicks on that button to Mortar; it never starts downloads or opens Nexus pages itself. Free users without an nxm:// link are told a Premium account or the button is required (Mortar surfaces the 403 as "requires a premium account or an nxm:// link"). Free downloads run one at a time. Nothing is scraped from the website, nothing is mass-fetched, no data is re-hosted.

## Rate-limit behaviour (internal/nexus/client.go)

- Every response's `X-RL-Daily-*` and `X-RL-Hourly-*` headers (remaining, limit, reset) are recorded and shown to the user.
- Requests are refused locally once either window's remaining count reaches 5 (`LimitFloor`) until the window's reset time, so Mortar stops before Nexus has to answer 429.
- A 429 from Nexus becomes a rate-limit error carrying the reset time (the daily reset if the daily budget is exhausted, else the hourly one); nothing retries blind.
- Update checks use one batched GraphQL query for all flagged mods rather than one call per mod; request timeout 20 s; responses are cached under the app's cache directory.
- Application metadata is real: not blank, not another app.

## Request identity

```
Application-Name: Mortar
Application-Version: <app version, e.g. 0.1.0>
User-Agent: Mortar/<app version>
```

## Data handling

User API key/tokens stay in the OS keyring on the user's machine. Mortar runs no server and sends no Nexus data to any third party. Cached Nexus metadata is stored locally for the user's own use.

## Short message to Nexus support (support@nexusmods.com)

Subject: Application registration request: Mortar (Stardew Valley mod manager)

Hello,

I would like to register Mortar, an open-source desktop mod manager for Stardew Valley (https://github.com/Rethunk-AI/mortar), under the API acceptable use policy. It currently uses users' personal API keys held in their OS keyring and sends accurate `Application-Name: Mortar`, `Application-Version` and `User-Agent: Mortar/<version>` headers; it tracks the X-RL rate-limit headers and stops before the limits are reached. Downloads start only from the Mod Manager Download button or a Premium download the user asked for.

Could you tell me:
1. Whether to register for legacy SSO (slug `mortar`) or an OAuth2 PKCE client, and how to get a client id.
2. Whether a custom-scheme redirect `nxm://oauth/callback` is accepted for a desktop app, or a loopback redirect is required.
3. The scopes available to a third-party client.
4. What logo size and testing build you need.

Thank you,
Damon Blais (@Albinogeek)
