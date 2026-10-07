# Nexus Mods application: registering Mortar

What Mortar sent Nexus Mods to register (emailed to support@nexusmods.com) and the facts behind it.

## How Nexus handles this (researched 2026-10-04)

- Nexus's API acceptable use policy ([help.nexusmods.com/article/114](https://help.nexusmods.com/article/114-api-acceptable-use-policy)) says to register an app: email support@nexusmods.com with a testing build, then submit the application name, description and logo, and receive a "slug" for SSO. Requests must carry `Application-Name` and `Application-Version` headers that are accurate and not blank or impersonating another app; it prohibits bulk fetching to rehost data and storing users' keys on your own server or using them without the user initiating the action. The article itself does not mention OAuth, redirect URIs or nxm://.
- Original API announcement ([nexusmods.com/news/13921](https://www.nexusmods.com/news/13921)): contact the Community Managers to get an app approved; only approved apps can use SSO; users can still paste a personal API key.
- Legacy SSO is a websocket flow ([Nexus-Mods/sso-integration-demo](https://github.com/Nexus-Mods/sso-integration-demo)): connect to `wss://sso.nexusmods.com`, send `{id, token, protocol: 2}`, open `https://www.nexusmods.com/sso?id=<uuid>&application=<slug>`, receive the user's API key over the socket. It needs a slug only Nexus staff can generate and involves no redirect URI.
- Newer OAuth2 (PKCE) flow: [Nexus-Mods/oauth2-demo-app](https://github.com/Nexus-Mods/oauth2-demo-app) shows a public-client PKCE flow with a custom protocol or localhost redirect. Vortex uses it against `https://users.nexusmods.com/oauth` (userinfo at `/oauth/userinfo`), client id `vortex_loopback`, scope `openid profile email`, redirect `http://127.0.0.1:<port>`; its source keeps `nxm://oauth/callback` only as a commented-out earlier value ([Vortex constants.ts](https://github.com/Nexus-Mods/Vortex/blob/master/src/renderer/src/extensions/nexus_integration/constants.ts)). Third-party clients (for example NexTwist) still use `nxm://oauth/callback` as their redirect.

Nexus's guide [Using OAuth 2.0 and PKCE in your application](https://modding.wiki/en/api/oauth2-guide) is the reference: third parties register a client by emailing support@nexusmods.com (name, short description, logo for a dark background, source link, callback URI), public apps use PKCE with a loopback or custom-scheme redirect, and V1 and V2 both accept the access token as `Authorization: Bearer`. Support (ticket 269342) asked only for the name, a callback that is not dynamic, and the scopes. Both flows stay built: the `nexussso.ClientID` ldflag selects OAuth PKCE, `nexussso.Slug` alone selects legacy SSO, and with neither set sign-in is hidden (see docs/architecture.md § Nexus).

## Application

- Application name: Mortar (consistent across versions; sent as `Application-Name: Mortar`)
- Proposed slug: `mortar`
- One-line description: Mortar is a desktop mod manager for Stardew Valley (Windows and Linux) that installs mods into per-profile sets, launches the game with SMAPI and shares profiles.
- Longer description: Mortar discovers the user's Stardew Valley install, installs SMAPI, keeps a separate mod set per profile and launches the game from the chosen profile. For Nexus it lets the user sign in, see which of their mods have updates, track and endorse mods, and download files they asked for. Mortar never re-hosts mod files; every download comes from Nexus's own servers (or the mod's other source).
- Website / repository: https://github.com/Rethunk-Tech/mortar
- Logo: supply the app icon from the repository (confirm file and a 512x512 export before sending).
- Testing build: attach the current release build from the GitHub releases page.
- Contact: the maintainers through opensource@rethunk.tech.

## Nexus's review (ticket 269342) and our answers

Nexus's reply: the primary failing point is personal API keys, which must be removed; and a concern with an auto update system that appears to run every four hours, since systems that activate without the user's input are generally not allowed. OAuth registration needs an application name, a callback that is not dynamic, and scopes.

| Nexus asked | Our answer |
| --- | --- |
| Application name | `Mortar` |
| Callback (fixed, not dynamic) | `http://127.0.0.1:51762/oauth/callback`, constant `nexussso.RedirectURI`. 51762 is in the IANA dynamic range (49152 to 65535), so no registered service owns it. If it is busy the sign-in fails with "Port 51762 is in use; close the app using it and try again"; there is no fallback port. |
| Scopes | `public openid profile` (constant `nexussso.Scope`). `public` is what authorises the API calls; `openid` and `profile` identify the account. Not independently confirmed on a Nexus help page (none was reachable); to be checked against the registration form. |
| Access token on API calls | `Authorization: Bearer <token>` on REST v1 and GraphQL v2, with the usual `Application-Name` and `Application-Version`. Per Nexus's [OAuth guide](https://modding.wiki/en/api/oauth2-guide#token-usage-and-validation). |
| Personal API keys | Compiled in but inert whenever a client id is built in, and deleted in the release that turns OAuth on. |
| Automatic updates | Update checks stay, including the four-hourly background check, which only notifies. Mortar never downloads or installs a Nexus file without the user's confirmation; nothing unattended queues one, and a queue left from a previous session stays paused at startup when it holds a Nexus file. Update before Play is an opt-in setting that runs only when the user presses Play. |

## Authentication requested

OAuth2 authorization code with PKCE as a public client (no client secret on the user's machine), because Mortar is a desktop app. Redirect URI, scopes and token use are in the table above.

- Today, until Nexus issues the client id, Mortar uses the user's personal API key, held in the OS keyring and sent only from the user's own machine, never to a Mortar server (Mortar has none).

## What Mortar calls

REST v1 (`https://api.nexusmods.com`): `users/validate.json`, `games/{game}.json`, `games/{game}/mods/{id}.json`, `mods/{id}/files.json`, `mods/{id}/changelogs.json`, `mods/{id}/files/{file}/download_link.json`, `mods/{id}/endorse.json`, `mods/{id}/abstain.json`, `user/tracked_mods.json`; and v2 GraphQL (`/v2/graphql`) for batched update checks. Game: `stardewvalley` only.

## How downloads start

Downloads start only from Nexus's own Mod Manager Download button (an nxm:// link carrying Nexus's key and expiry, which Mortar then redeems) or, for Premium accounts, a Premium API download link requested because the user asked for that mod in Mortar. The browser extension only reads and marks Nexus pages and relays clicks on that button to Mortar; it never starts downloads or opens Nexus pages itself. Free users without an nxm:// link are told a Premium account or the button is required (Mortar surfaces the 403 as "requires a premium account or an nxm:// link"). Free downloads run one at a time. Nothing is scraped from the website, nothing is mass-fetched, no data is re-hosted.

## Rate-limit behaviour (internal/nexus/client.go)

- Every response's `X-RL-Daily-*` and `X-RL-Hourly-*` headers (remaining, limit, reset) are recorded and shown to the user.
- Requests are refused locally once either window's remaining count reaches 5 (`LimitFloor`) until the window's reset time, so Mortar stops before Nexus has to answer 429.
- A 429 from Nexus becomes a rate-limit error carrying the reset time (the daily reset if the daily budget is exhausted, else the hourly one); nothing retries blind.
- Update checks use one batched GraphQL query for all flagged mods rather than one call per mod; request timeout 20 s. Only a mod's page data (`internal/nexus/mod.go`, kept on disk) and the tracked-mods list (`internal/nexus/account.go`, 30 s) are cached; update-check results are not.
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

I would like to register Mortar, an open-source desktop mod manager for Stardew Valley (https://github.com/Rethunk-Tech/mortar), under the API acceptable use policy. It currently uses users' personal API keys held in their OS keyring and sends accurate `Application-Name: Mortar`, `Application-Version` and `User-Agent: Mortar/<version>` headers; it tracks the X-RL rate-limit headers and stops before the limits are reached. Downloads start only from the Mod Manager Download button or a Premium download the user asked for.

Could you tell me:
1. Whether to register for legacy SSO (slug `mortar`) or an OAuth2 PKCE client, and how to get a client id.
2. Whether a custom-scheme redirect `nxm://oauth/callback` is accepted for a desktop app, or a loopback redirect is required.
3. The scopes available to a third-party client.
4. What logo size and testing build you need.

Thank you,
Damon Blais (@Albinogeek)
