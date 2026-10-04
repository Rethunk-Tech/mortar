import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Checkbox, FormControlLabel, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import type { ImportPreview } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { ApplyImport } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../../toasts/report.ts'
import { useToasts } from '../../toasts/store.ts'
import { prefCopy } from '../prefCopy.ts'

function sectionTitle(section: string): string {
  switch (section) {
    case 'appearance':
      return i18n._(msg`Appearance`)
    case 'notifications':
      return i18n._(msg`Notifications`)
    case 'downloads':
      return i18n._(msg`Downloads and updates`)
    case 'storage':
      return i18n._(msg`Storage`)
    case 'sharing':
      return i18n._(msg`Sharing`)
    case 'games':
      return i18n._(msg`Games`)
    default:
      return i18n._(msg`General`)
  }
}

// The registry label where the settings pages have one, else the label the file preview carried.
function changeLabel(key: string, fallback: string): string {
  const { label } = prefCopy(i18n, key)
  return label === key ? fallback : label
}

export function ImportSettingsDialog({
  path,
  preview,
  onClose,
}: {
  path: string
  preview: ImportPreview | null
  onClose: () => void
}) {
  const { t } = useLingui()
  const sections = preview?.sections ?? []
  const [off, setOff] = useState<Set<string>>(new Set())
  useEffect(() => {
    if (preview !== null) {
      setOff(new Set())
    }
  }, [preview])
  const chosen = sections.filter((s) => !off.has(s.section))
  const count = chosen.reduce((n, s) => n + (s.changes?.length ?? 0), 0)
  const ignored = preview?.ignored ?? []
  const toggle = (section: string) =>
    setOff((prev) => {
      const next = new Set(prev)
      if (!next.delete(section)) {
        next.add(section)
      }
      return next
    })
  return (
    <ConfirmDialog
      open={preview !== null}
      title={t`Import settings`}
      confirmLabel={t`Import ${count} changes`}
      confirmDisabled={count === 0}
      maxWidth={460}
      onCancel={onClose}
      onConfirm={() => {
        ApplyImport(
          path,
          chosen.map((s) => s.section),
        )
          .then(() => {
            useToasts.getState().push({ kind: 'success', title: t`Settings imported` })
            onClose()
          })
          .catch(reportUnexpected)
      }}
    >
      {sections.length === 0 ? (
        <Box sx={{ fontSize: 13 }}>{t`Nothing would change.`}</Box>
      ) : (
        sections.map((s) => (
          <Box key={s.section} sx={{ mb: 1 }}>
            <FormControlLabel
              control={
                <Checkbox
                  checked={!off.has(s.section)}
                  onChange={() => toggle(s.section)}
                  size="small"
                />
              }
              label={sectionTitle(s.section)}
            />
            {(s.changes ?? []).map((c) => (
              <Box key={c.key} sx={{ fontSize: 13, pl: 4 }}>
                {i18n._(msg`${changeLabel(c.key, c.label)}: ${c.from} → ${c.to}`)}
              </Box>
            ))}
          </Box>
        ))
      )}
      {ignored.length > 0 ? (
        <Typography sx={{ fontSize: 12, color: 'text.secondary', mt: 1 }}>
          {t`Ignored fields Mortar does not import: ${ignored.join(', ')}`}
        </Typography>
      ) : null}
    </ConfirmDialog>
  )
}
