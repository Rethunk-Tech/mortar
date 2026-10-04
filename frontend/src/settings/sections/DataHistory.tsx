import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Button } from '@mui/material'
import { useEffect, useState } from 'react'
import type { HistoryUsage } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import {
  HistoryUsage as LoadHistoryUsage,
  TrimHistory,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/profile/service.ts'
import { formatBytes } from '../../i18n/bytes.ts'
import { useProfiles } from '../../profiles/store.ts'
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
            freed > 0
              ? t`Trimmed ${usage.profileName}'s history, freed ${formatBytes(freed)}`
              : t`Trimmed ${usage.profileName}'s history`,
        })
      },
      { errorTitle: t`Could not trim history` },
    )
  const events = plural(usage.events, { one: '# change', other: '# changes' })
  return (
    <SettingRow
      label={t`Profile history: ${usage.profileName}`}
      description={`${formatBytes(sizeOf(usage))} · ${events}`}
    >
      <Button variant="outlined" disabled={usage.events <= 1} onClick={() => setOpen(true)}>
        {t`Trim…`}
      </Button>
      <TrimDialog
        title={t`Trim history of ${usage.profileName}`}
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
  const [rows, setRows] = useState<HistoryUsage[]>([])
  useEffect(() => {
    if (gameId) {
      LoadHistoryUsage(gameId)
        .then((r) => setRows(r ?? []))
        .catch(reportUnexpected)
    }
  }, [gameId])
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
