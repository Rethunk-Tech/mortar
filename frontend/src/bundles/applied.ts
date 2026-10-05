import { msg, plural } from '@lingui/core/macro'
import type { ApplyResult } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/bundles/models.ts'
import { i18n } from '../i18n/index.ts'
import { listNames } from '../i18n/list.ts'
import { useMods } from '../mods/store.ts'
import { useProfiles } from '../profiles/store.ts'
import { download } from '../queue/actions.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { bundleWants } from './missingWants.ts'

// Shows a bundle's result on the profile it was added to, and offers the downloads it still needs.
export function bundleApplied(result: ApplyResult, profileId: string): void {
  useProfiles.getState().replace(result.profile)
  const open = useProfiles.getState().openId === profileId
  if (open) {
    useMods.getState().load().catch(reportUnexpected)
  }
  const missing = result.missing ?? []
  const added = plural(result.added, { one: '# mod added', other: '# mods added' })
  const wants = open ? bundleWants(result.missingMods) : []
  useToasts.getState().push({
    kind: missing.length > 0 ? 'warning' : 'success',
    title: i18n._(msg`Bundle added`),
    body:
      missing.length > 0
        ? `${added}\n${i18n._(msg`Not downloaded yet: ${listNames(missing)}`)}`
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
