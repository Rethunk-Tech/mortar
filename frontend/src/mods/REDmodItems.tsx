import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Repeat2, Undo2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  ConvertToREDmod,
  REDmodChoice,
  RevertREDmod,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { openProfileOf, useProfiles } from '../profiles/store.ts'
import { DisabledReason } from '../shell/DisabledReason.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { lockedHint } from './lockedHint.ts'
import { ICON_SIZE } from './menu.ts'
import { useLocked } from './useLocked.ts'

type Choice = 'convert' | 'revert' | ''

// What the menu offers all of `keys`: a choice every one of them has, else nothing.
function useREDmodChoice(game: string, profile: string, keys: string[]): Choice {
  const [choice, setChoice] = useState<Choice>('')
  const wanted = keys.join('\n')
  useEffect(() => {
    let current = true
    setChoice('')
    if (!(game && profile && wanted)) {
      return
    }
    Promise.all(wanted.split('\n').map((key) => REDmodChoice(game, profile, key)))
      .then((all) => {
        const first = all[0] ?? ''
        if (
          current &&
          all.every((c) => c === first) &&
          (first === 'convert' || first === 'revert')
        ) {
          setChoice(first)
        }
      })
      .catch(reportUnexpected)
    return () => {
      current = false
    }
  }, [game, profile, wanted])
  return choice
}

// Vortex's Convert to REDmod and Revert to archive mod: shown for the Cyberpunk 2077 mods each applies to.
export function REDmodItems({ mods, close }: { mods: Mod[]; close: () => void }) {
  const { t } = useLingui()
  const locked = useLocked()
  const game = useProfiles((s) => s.game)
  const profile = useProfiles(openProfileOf)
  const keys = [...new Set(mods.map((m) => m.key))]
  const choice = useREDmodChoice(
    game?.layout === 'cyberpunk' ? game.id : '',
    profile?.id ?? '',
    keys,
  )
  if (!(choice && game && profile)) {
    return null
  }
  const run = () => {
    close()
    const act = choice === 'convert' ? ConvertToREDmod : RevertREDmod
    act(game.id, profile.id, keys)
      .then((res) => {
        useProfiles.getState().replace(res.profile)
        const done = keys.length - (res.skipped?.length ?? 0)
        if (done > 0) {
          useToasts.getState().push({
            kind: 'success',
            title:
              choice === 'convert'
                ? plural(done, {
                    one: 'Converted # mod to a REDmod',
                    other: 'Converted # mods to REDmods',
                  })
                : plural(done, {
                    one: 'Reverted # mod to an archive mod',
                    other: 'Reverted # mods to archive mods',
                  }),
            body:
              choice === 'convert'
                ? t`It loads through REDmod now, after the archive mods. Check its place in the order.`
                : t`It loads with the archive mods again.`,
            changes: [res.profile.lastChange ?? ''],
          })
        }
        for (const skip of res.skipped ?? []) {
          useToasts.getState().push({ kind: 'warning', title: skip.name, body: skip.reason })
        }
      })
      .catch(reportUnexpected)
  }
  return (
    <DisabledReason title={t(lockedHint)} disabled={locked}>
      <MenuAction
        disabled={locked}
        icon={choice === 'convert' ? <Repeat2 size={ICON_SIZE} /> : <Undo2 size={ICON_SIZE} />}
        label={choice === 'convert' ? t`Convert to REDmod` : t`Revert to archive mod`}
        onClick={run}
      />
    </DisabledReason>
  )
}
