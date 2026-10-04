import { i18n } from '@lingui/core'
import { msg } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box } from '@mui/material'
import type { ImportPreview } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import { ApplyImportedSettings } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { ConfirmDialog } from '../../shell/ConfirmDialog.tsx'
import { reportUnexpected } from '../../toasts/report.ts'

function changeLine(field: string, from: string, to: string) {
  switch (field) {
    case 'accent':
      return i18n._(msg`Accent: ${from} → ${to}`)
    case 'background':
      return i18n._(msg`Background: ${from} → ${to}`)
    case 'lastGame':
      return i18n._(msg`Last game: ${from} → ${to}`)
    case 'backupsKept':
      return i18n._(msg`Backups kept: ${from} → ${to}`)
    case 'listColumns':
      return i18n._(msg`List columns: ${from} → ${to}`)
    case 'listSortColumn':
      return i18n._(msg`List sort: ${from} → ${to}`)
    case 'listSortDir':
      return i18n._(msg`List sort direction: ${from} → ${to}`)
    case 'listGroupBy':
      return i18n._(msg`List grouping: ${from} → ${to}`)
    case 'checkModUpdatesOnStart':
      return i18n._(msg`Check mod updates on start: ${from} → ${to}`)
    case 'includePrereleaseModVersions':
      return i18n._(msg`Include pre-release mod versions: ${from} → ${to}`)
    case 'checkOnlyEnabledMods':
      return i18n._(msg`Check only enabled mods: ${from} → ${to}`)
    case 'enableModsWhenInstalled':
      return i18n._(msg`Enable mods when installed: ${from} → ${to}`)
    case 'tellWhenSmapiOut':
      return i18n._(msg`Tell when SMAPI is out: ${from} → ${to}`)
    case 'tipsSeen':
      return i18n._(msg`Seen tips: ${from} → ${to}`)
    default:
      return i18n._(msg`${field}: ${from} → ${to}`)
  }
}

export function ImportSettingsDialog({
  preview,
  onClose,
}: {
  preview: ImportPreview | null
  onClose: () => void
}) {
  const { t } = useLingui()
  const changes = preview?.changes ?? []
  return (
    <ConfirmDialog
      open={preview !== null}
      title={t`Import settings`}
      confirmLabel={t`Import`}
      confirmDisabled={changes.length === 0}
      maxWidth={360}
      onCancel={onClose}
      onConfirm={() => {
        if (!preview?.raw) {
          return
        }
        ApplyImportedSettings(preview.raw).then(onClose).catch(reportUnexpected)
      }}
    >
      {changes.length === 0 ? (
        <Box sx={{ fontSize: 13 }}>{t`Nothing would change.`}</Box>
      ) : (
        changes.map((c) => (
          <Box key={c.field} sx={{ fontSize: 13 }}>
            {changeLine(c.field, c.from, c.to)}
          </Box>
        ))
      )}
    </ConfirmDialog>
  )
}
