import { msg, plural } from '@lingui/core/macro'
import type { Request } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/models.ts'
import { Add } from '../../bindings/github.com/Rethunk-AI/mortar/internal/queue/service.ts'
import { i18n } from '../i18n/index.ts'
import { openSettings } from '../nav/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { useQueue } from './store.ts'

// The game and profile open now, or null when none is.
function target() {
  const { game, openId } = useProfiles.getState()
  return game && openId ? { game: game.id, profileId: openId } : null
}

// What a caller says about a file; the rest is filled in.
export type Want = Pick<Request, 'kind'> &
  Partial<
    Pick<
      Request,
      | 'modId'
      | 'fileId'
      | 'name'
      | 'fileName'
      | 'version'
      | 'currentKey'
      | 'repo'
      | 'tag'
      | 'asset'
      | 'latest'
      | 'batchId'
    >
  >

// Queues files for the open profile. Nexus files cannot download while signed out: say so and point at the
// sign-in instead. GitHub needs no account.
export async function download(reqs: Want[], showQueue = false): Promise<boolean> {
  const toasts = useToasts.getState()
  if (!useNexus.getState().signedIn && reqs.some((r) => !r.repo)) {
    toasts.push({
      kind: 'warning',
      title: i18n._(msg`Sign in to Nexus Mods to download`),
      body: i18n._(
        msg`Downloads come from Nexus with your account, so Mortar needs your API key first.`,
      ),
      action: { label: i18n._(msg`Open settings`), run: () => openSettings('nexus') },
    })
    return false
  }
  const at = target()
  if (!at) {
    return false
  }
  const batchId = reqs.length > 1 ? crypto.randomUUID() : ''
  try {
    await Add(
      reqs.map((r) => ({
        modId: 0,
        repo: '',
        tag: '',
        asset: '',
        fileId: 0,
        name: '',
        fileName: '',
        version: '',
        currentKey: '',
        batchId,
        latest: false,
        ...r,
        ...at,
      })),
    )
  } catch (e) {
    toastError(i18n._(msg`Could not add the download`), e)
    return false
  }
  if (showQueue) {
    useQueue.getState().setOpen(true)
  } else {
    toasts.push({
      kind: 'info',
      title: i18n._(
        msg`${plural(reqs.length, { one: '# download added', other: '# downloads added' })}`,
      ),
      action: {
        label: i18n._(msg`Show`),
        run: () => useQueue.getState().setOpen(true),
        live: () =>
          useQueue.getState().state.items.length === 0
            ? { disabled: true, reason: i18n._(msg`The download queue is empty.`) }
            : { disabled: false },
      },
    })
  }
  return true
}
