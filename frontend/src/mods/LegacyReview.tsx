import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Checkbox, FormControlLabel, List, ListItem, Typography } from '@mui/material'
import { useState } from 'react'
import type { Mod } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import type { LegacyWhy } from './legacyReasons.ts'

// The legacy offer's suggestions one by one, each with why it was picked; the player keeps the ticked ones.
export function LegacyReview({
  picks,
  mods,
  busy,
  onCancel,
  onMark,
}: {
  picks: { key: string; why: LegacyWhy }[]
  mods: Mod[]
  busy: boolean
  onCancel: () => void
  onMark: (keys: string[]) => void
}) {
  const { t } = useLingui()
  const [off, setOff] = useState<ReadonlySet<string>>(new Set())
  const why: Record<LegacyWhy, string> = {
    required: t`Other mods require it`,
    category: t`Listed as a library on its site`,
    supporting: t`Supplies tilesheets for other mods`,
  }
  const chosen = picks.filter((p) => !off.has(p.key)).map((p) => p.key)
  return (
    <ConfirmDialog
      open={true}
      title={t`Libraries and supporting mods`}
      confirmLabel={plural(
        { count: chosen.length },
        { one: 'Mark # implicit', other: 'Mark # implicit' },
      )}
      confirmDisabled={chosen.length === 0}
      busy={busy}
      onCancel={onCancel}
      onConfirm={() => onMark(chosen)}
    >
      <List dense={true} disablePadding={true} sx={{ maxHeight: '50vh', overflowY: 'auto' }}>
        {picks.map((p) => (
          <ListItem key={p.key} disableGutters={true}>
            <FormControlLabel
              control={
                <Checkbox
                  size="small"
                  checked={!off.has(p.key)}
                  onChange={(e) => {
                    const next = new Set(off)
                    if (e.target.checked) {
                      next.delete(p.key)
                    } else {
                      next.add(p.key)
                    }
                    setOff(next)
                  }}
                />
              }
              label={
                <>
                  <Typography sx={{ fontSize: 14 }}>
                    {mods.find((m) => m.key === p.key)?.name ?? p.key}
                  </Typography>
                  <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
                    {why[p.why]}
                  </Typography>
                </>
              }
            />
          </ListItem>
        ))}
      </List>
    </ConfirmDialog>
  )
}
