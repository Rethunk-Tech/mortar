import { i18n } from '@lingui/core'
import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Checkbox, FormControlLabel, Typography } from '@mui/material'
import { useEffect, useState } from 'react'
import type { ImportPreview } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { ApplyImport } from '../../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { listNames } from '../../i18n/list.ts'
import { ConfirmationDialog } from '../../shell/dialog/ConfirmationDialog.tsx'
import { space } from '../../theme/density.ts'
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
    <ConfirmationDialog
      open={preview !== null}
      title={t`Import settings`}
      confirmLabel={plural(count, { one: 'Import # change', other: 'Import # changes' })}
      confirmDisabled={count === 0}
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
        <Box sx={{ fontSize: 'base.fontSize' }}>{t`Nothing would change.`}</Box>
      ) : (
        sections.map((s) => (
          <Box key={s.section} sx={{ mb: space.gap }}>
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
              <Box key={c.key} sx={{ fontSize: 'base.fontSize', pl: 4 }}>
                {i18n._(
                  msg`${{ name: changeLabel(c.key, c.label) }}: ${{ from: c.from }} to ${{ to: c.to }}`,
                )}
              </Box>
            ))}
          </Box>
        ))
      )}
      {ignored.length > 0 ? (
        <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary', mt: space.gap }}>
          {t`Ignored fields Mortar does not import: ${{ names: listNames(ignored) }}`}
        </Typography>
      ) : null}
    </ConfirmationDialog>
  )
}
