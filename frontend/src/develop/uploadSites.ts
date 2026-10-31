import { msg } from '@lingui/core/macro'
import { useEffect, useState } from 'react'
import {
  ForgetCurseForgeToken,
  ForgetThunderstoreToken,
  UploadCurseForge,
  UploadNexus,
  UploadSites,
  UploadThunderstore,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { i18n } from '../i18n/index.ts'
import { openWeb } from '../shell/openWeb.ts'
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
  return UploadThunderstore(game, dir)
}

// A refused token is forgotten, so the dialog asks for it again next time.
async function forgetRefused(id: string, game: string, dir: string) {
  if (id === 'thunderstore') {
    await ForgetThunderstoreToken(game, dir).catch(() => undefined)
  } else if (id === 'curseforge') {
    await ForgetCurseForgeToken().catch(() => undefined)
  }
}

/** Uploads the newest package to one site and reports the outcome in a toast: the page's link, or the failure. */
export async function uploadTo(
  id: string,
  { game, dir, name, version }: { game: string; dir: string; name: string; version: string },
) {
  const site = siteLabel(id)
  try {
    const url = await uploadCall(id, game, dir, version)
    useToasts.getState().push({
      kind: 'success',
      title: i18n._(msg`Uploaded ${name} to ${site}`),
      ...(url
        ? {
            action: {
              label: i18n._(msg`Open page`),
              run: () => openWeb(url).catch(reportError(i18n._(msg`Could not open the page`))),
            },
          }
        : {}),
    })
  } catch (e) {
    reportError(i18n._(msg`Could not upload to ${site}`))(e)
    if (errorKind(e) === 'permission') {
      await forgetRefused(id, game, dir)
    }
  }
}

/** The ids of the sites the game can be uploaded to, empty until they are read. */
export function useUploadSites(game: string, dir: string): string[] {
  const [ids, setIds] = useState<string[]>([])
  useEffect(() => {
    UploadSites(game, dir)
      .then((list) => setIds((list ?? []).map((s) => s.id)))
      .catch(() => setIds([]))
  }, [game, dir])
  return ids
}
