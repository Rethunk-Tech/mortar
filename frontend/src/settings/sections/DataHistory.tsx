import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { useState } from 'react'
import type { HistoryUsage } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import {
  HistoryUsage as LoadHistoryUsage,
  TrimHistory,
} from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/service.ts'
import { formatBytes } from '../../i18n/bytes.ts'
import { useProfiles } from '../../profiles/store.ts'
import { DisabledReason } from '../../shell/DisabledReason.tsx'
import { useLoaded } from '../../shell/useLoaded.ts'
import { reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { usePending } from '../../toasts/usePending.ts'
import { SettingRow } from '../SettingsSection.tsx'
import { TrimDialog } from './DataBackups.tsx'

const sizeOf = (u: HistoryUsage) => u.snapshotBytes + u.fileBytes

function HistoryRow({
  usage,
  gameId,
  onTrimmed,
}: {
  usage: HistoryUsage
  gameId: string
  onTrimmed: (next: HistoryUsage) => void
}) {
  const { t } = useLingui()
  const [open, setOpen] = useState(false)
  const [pending, run] = usePending()
  const trim = (keep: number) =>
    run(
      async () => {
        const next = await TrimHistory(gameId, usage.profileId, keep)
        setOpen(false)
        onTrimmed(next)
        const freed = sizeOf(usage) - sizeOf(next)
        useToasts.getState().push({
          kind: 'success',
          title:
            next.events === usage.events
              ? t`No changes to trim`
              : t`Trimmed ${{ profile: usage.profileName }}'s history, freed ${{ size: formatBytes(Math.max(freed, 0)) }}`,
        })
      },
      { errorTitle: t`Could not trim history` },
    )
  const events = plural({ count: usage.events }, { one: '# change', other: '# changes' })
  return (
    <SettingRow
      label={t`Profile history: ${{ profile: usage.profileName }}`}
      description={`${formatBytes(sizeOf(usage))} · ${events}`}
    >
      <DisabledReason title={t`Nothing to trim yet.`} disabled={usage.events <= 1}>
        <Button variant="outlined" disabled={usage.events <= 1} onClick={() => setOpen(true)}>
          {t`Trim…`}
        </Button>
      </DisabledReason>
      <TrimDialog
        title={t`Trim history of ${{ profile: usage.profileName }}`}
        body={t`Older changes are dropped and can no longer be undone.`}
        fieldLabel={t`Changes to keep`}
        defaultKeep={50}
        open={open}
        busy={pending}
        onClose={() => setOpen(false)}
        onTrim={trim}
      />
    </SettingRow>
  )
}

export function HistoryUsageRows() {
  const gameId = useProfiles((s) => s.game?.id)
  const { data: rows, setData: setRows } = useLoaded<HistoryUsage[]>(
    gameId ? () => LoadHistoryUsage(gameId).then((r) => r ?? []) : null,
    [gameId],
    [],
    reportUnexpected,
  )
  if (!gameId) {
    return null
  }
  return rows.map((u) => (
    <HistoryRow
      key={u.profileId}
      usage={u}
      gameId={gameId}
      onTrimmed={(next) =>
        setRows((all) => all.map((r) => (r.profileId === next.profileId ? next : r)))
      }
    />
  ))
}
