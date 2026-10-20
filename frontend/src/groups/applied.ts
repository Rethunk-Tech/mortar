import { msg, plural } from '@lingui/core/macro'
import type { ApplyResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savedgroups/models.ts'
import { i18n } from '../i18n/index.ts'
import { listNames } from '../i18n/list.ts'
import { useMods } from '../mods/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { savedGroupWants } from './missingWants.ts'

// Shows a saved group's result on the profile it was added to, and offers the downloads it still needs.
export function savedGroupApplied(result: ApplyResult, profileId: string): void {
  useProfiles.getState().replace(result.profile)
  const open = useProfiles.getState().openId === profileId
  if (open) {
    useMods.getState().load().catch(reportUnexpected)
  }
  const missing = result.missing ?? []
  const added = plural(result.added, { one: '# mod added', other: '# mods added' })
  const wants = open ? savedGroupWants(result.missingMods) : []
  const { group } = result
  useToasts.getState().push({
    kind: missing.length > 0 ? 'warning' : 'success',
    title: group ? i18n._(msg`Added the group ${group}`) : i18n._(msg`Saved group added`),
    changes: [result.profile.lastChange ?? ''],
    body:
      missing.length > 0
        ? `${added}\n${i18n._(msg`Not downloaded yet: ${listNames(missing, missing.length)}`)}`
        : added,
    ...(wants.length > 0
      ? {
          action: {
            label: i18n._(msg`Download them`),
            run: () => download(wants, true).catch(reportUnexpected),
          },
        }
      : {}),
  })
}
