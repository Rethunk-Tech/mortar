import { msg } from '@lingui/core/macro'
import {
  ForgetCurseForgeToken,
  ForgetThunderstoreToken,
  UploadCurseForge,
  UploadGitHub,
  UploadNexus,
  UploadThunderstore,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { i18n } from '../i18n/index.ts'
import { openPageReported } from '../shell/openPageReported.ts'
import { errorKind } from '../toasts/errorKind.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

export const siteLabel = (id: string) => (id === 'nexus' ? 'Nexus Mods' : sourceLabel(id))

function uploadCall(id: string, game: string, dir: string, version: string): Promise<string> {
  if (id === 'nexus') {
    return UploadNexus(game, dir, version)
  }
  if (id === 'curseforge') {
    return UploadCurseForge(game, dir, version)
  }
  if (id === 'github') {
    return UploadGitHub(game, dir, version)
  }
  return UploadThunderstore(game, dir)
}

/** The site whose kept token a refusal invalidates; nexus and github keep none to forget. */
export function tokenToForget(id: string): 'thunderstore' | 'curseforge' | undefined {
  return id === 'thunderstore' || id === 'curseforge' ? id : undefined
}

// A refused token is forgotten, so the dialog asks for it again next time.
async function forgetRefused(id: string, game: string, dir: string) {
  const site = tokenToForget(id)
  if (site === 'thunderstore') {
    await ForgetThunderstoreToken(game, dir).catch(() => undefined)
  } else if (site === 'curseforge') {
    await ForgetCurseForgeToken().catch(() => undefined)
  }
}

/** Uploads the newest package to one site and returns the page's link (empty when the site has none). A refused token
 * is forgotten, so the next upload asks for it again. */
export async function uploadResult(
  id: string,
  { game, dir, version }: { game: string; dir: string; version: string },
): Promise<string> {
  try {
    return await uploadCall(id, game, dir, version)
  } catch (e) {
    if (errorKind(e) === 'permission') {
      await forgetRefused(id, game, dir)
    }
    throw e
  }
}

/** Uploads the newest package to one site and reports the outcome in a toast: the page's link, or the failure. */
export async function uploadTo(
  id: string,
  { game, dir, name, version }: { game: string; dir: string; name: string; version: string },
) {
  const site = siteLabel(id)
  try {
    const url = await uploadResult(id, { game, dir, version })
    useToasts.getState().push({
      kind: 'success',
      title: i18n._(msg`Uploaded ${name} to ${site}`),
      ...(url
        ? {
            action: {
              label: i18n._(msg`Open page`),
              run: () => openPageReported(url),
            },
          }
        : {}),
    })
  } catch (e) {
    reportError(i18n._(msg`Could not upload to ${site}`))(e)
  }
}

export const TEAMS_PAGE = 'https://thunderstore.io/settings/teams/'
