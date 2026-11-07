#!/usr/bin/env bun
// The maintainer's Nexus OAuth session for sandboxes and live tests, so a route that needs Nexus runs without a browser.
//
//   capture   moves the grant a real sign-in left in the app's keyring item to the test item
//   token     prints a fresh access token, into a pipe only
//   grant     prints the app's keyring item with the access token alone, into a pipe only
//   check     says whether the session works, and whose it is
//
// Nexus can rotate a refresh token, so one grant can have one holder. The test item is its own grant, and only this
// script refreshes it, under a lock; sandboxes get the access token without the refresh token, so a sandbox can
// never sign the session out. Its access token lasts hours: run `selftest.sh prep-shots` again when one runs out.
import { mkdirSync, rmSync, statSync } from 'node:fs'
import { tmpdir } from 'node:os'
import { join } from 'node:path'

const CLIENT_ID = 'mortar'
const TOKEN_URL = 'https://users.nexusmods.com/oauth/token'
const USERINFO_URL = 'https://users.nexusmods.com/oauth/userinfo'
const SECRET_TOOL = '/usr/bin/secret-tool'
const APP_ITEM = ['service', 'mortar', 'username', 'nexus-oauth']
const TEST_ITEM = ['service', 'mortar-test', 'username', 'nexus-oauth']
const MS_PER_MINUTE = 60_000
const MS_PER_SECOND = 1000
const REFRESH_MARGIN_MINUTES = 30
const REFRESH_MARGIN_MS = REFRESH_MARGIN_MINUTES * MS_PER_MINUTE
const STALE_LOCK_MS = MS_PER_MINUTE
const LOCK_WAIT_MS = 200

interface Grant {
  access: string
  refresh: string
  expires: string
}

function read(item: string[]): Grant | null {
  const run = Bun.spawnSync([SECRET_TOOL, 'lookup', ...item])
  const raw = run.stdout.toString()
  return run.exitCode === 0 && raw !== '' ? (JSON.parse(raw) as Grant) : null
}

function write(item: string[], grant: Grant) {
  const run = Bun.spawnSync([SECRET_TOOL, 'store', '--label=Mortar Nexus test session', ...item], {
    stdin: new TextEncoder().encode(JSON.stringify(grant)),
  })
  if (run.exitCode !== 0) {
    throw new Error('could not write the keyring item')
  }
}

function fail(message: string): never {
  console.error(message)
  process.exit(1)
}

async function withLock<T>(fn: () => Promise<T>): Promise<T> {
  const dir = join(tmpdir(), 'mortar-nexus-test-session.lock')
  for (;;) {
    try {
      mkdirSync(dir)
      break
    } catch {
      try {
        if (Date.now() - statSync(dir).mtimeMs > STALE_LOCK_MS) {
          rmSync(dir, { recursive: true, force: true })
          continue
        }
      } catch {
        continue
      }
      await Bun.sleep(LOCK_WAIT_MS)
    }
  }
  try {
    return await fn()
  } finally {
    rmSync(dir, { recursive: true, force: true })
  }
}

async function fresh(): Promise<Grant> {
  return await withLock(async () => {
    const grant = read(TEST_ITEM)
    if (!grant) {
      fail('no test session: sign in through a build of Mortar, then run capture')
    }
    if (new Date(grant.expires).getTime() - Date.now() > REFRESH_MARGIN_MS) {
      return grant
    }
    const res = await fetch(TOKEN_URL, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: new URLSearchParams({
        grant_type: 'refresh_token',
        client_id: CLIENT_ID,
        refresh_token: grant.refresh,
      }),
    })
    if (!res.ok) {
      fail(`Nexus refused the refresh (${res.status}): sign in again, then run capture`)
    }
    const body = (await res.json()) as {
      access_token: string
      refresh_token?: string
      expires_in: number
    }
    const next = {
      access: body.access_token,
      refresh: body.refresh_token || grant.refresh,
      expires: new Date(Date.now() + body.expires_in * MS_PER_SECOND).toISOString(),
    }
    write(TEST_ITEM, next)
    return next
  })
}

function refuseTerminal() {
  if (process.stdout.isTTY) {
    fail('this prints a token: send it to a pipe')
  }
}

const command = process.argv[2]
if (command === 'capture') {
  const grant = read(APP_ITEM)
  if (!grant) {
    fail('the app holds no Nexus grant: sign in through a build of Mortar first')
  }
  write(TEST_ITEM, grant)
  if (Bun.spawnSync([SECRET_TOOL, 'clear', ...APP_ITEM]).exitCode !== 0) {
    fail('the grant was copied, but the app still holds it: sign out in that build')
  }
  console.log('captured; the build that signed in is now signed out')
} else if (command === 'token') {
  refuseTerminal()
  process.stdout.write((await fresh()).access)
} else if (command === 'grant') {
  refuseTerminal()
  const grant = await fresh()
  process.stdout.write(JSON.stringify({ ...grant, refresh: '' }))
} else if (command === 'check') {
  const grant = await fresh()
  const res = await fetch(USERINFO_URL, { headers: { Authorization: `Bearer ${grant.access}` } })
  if (!res.ok) {
    fail(`Nexus refused the session (${res.status})`)
  }
  const user = (await res.json()) as { name: string; membership_roles?: string[] }
  console.log(`${user.name}, access token until ${grant.expires}`)
} else {
  fail('usage: nexus-test-session.ts capture|token|grant|check')
}
