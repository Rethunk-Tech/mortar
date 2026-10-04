import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  Tooltip,
  Typography,
} from '@mui/material'
import { LayoutTemplate, Trash2 } from 'lucide-react'
import { useState } from 'react'
import type { Template } from '../../bindings/github.com/Rethunk-AI/mortar/internal/templates/models.ts'
import {
  DeleteTemplate,
  SaveTemplateFromProfile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/templates/service.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { PromptDialog } from '../shell/PromptDialog.tsx'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'

const MAX_NAME = 60

function SaveTemplateDialog({
  open,
  game,
  profileId,
  profileName,
  onClose,
}: {
  open: boolean
  game: string
  profileId: string
  profileName: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const [busy, run] = usePending()
  return (
    <PromptDialog
      open={open}
      title={t`Save as template`}
      label={t`Template name`}
      initial={profileName}
      maxLength={MAX_NAME}
      confirmLabel={t`Save`}
      busy={busy}
      onCancel={onClose}
      onSubmit={(name) => {
        run(
          () =>
            SaveTemplateFromProfile(game, profileId, name).then(() => {
              useToasts.getState().push({ kind: 'success', title: t`Template saved` })
              onClose()
            }),
          { errorTitle: t`Could not save the template` },
        )
      }}
    />
  )
}

function TemplateRow({ template, onDelete }: { template: Template; onDelete: () => void }) {
  const { t } = useLingui()
  const mods = plural(template.bundle?.length ?? 0, { one: '# mod', other: '# mods' })
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.5 }}>
      <Typography noWrap={true} sx={{ flex: 1, minWidth: 0 }} title={template.name}>
        {template.name}
      </Typography>
      <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>{mods}</Typography>
      <Tooltip title={t`Delete`}>
        <IconButton size="small" aria-label={t`Delete ${template.name}`} onClick={onDelete}>
          <Trash2 size={16} />
        </IconButton>
      </Tooltip>
    </Box>
  )
}

function ManageTemplatesDialog({
  open,
  game,
  templates,
  onChanged,
  onClose,
}: {
  open: boolean
  game: string
  templates: Template[]
  onChanged: () => void
  onClose: () => void
}) {
  const { t } = useLingui()
  const [deleting, setDeleting] = useState<string | null>(null)
  const [busy, run] = usePending()
  return (
    <>
      <Dialog open={open} onClose={onClose}>
        <DialogTitle>{t`Manage templates`}</DialogTitle>
        <DialogContent sx={{ minWidth: 380, maxWidth: 'calc(100vw - 64px)' }}>
          {templates.length === 0 ? (
            <EmptyState compact={true} icon={<LayoutTemplate size={28} />} title={t`No templates`}>
              {t`Choose Save as template… from a profile's menu.`}
            </EmptyState>
          ) : (
            templates.map((template) => (
              <TemplateRow
                key={template.name}
                template={template}
                onDelete={() => setDeleting(template.name)}
              />
            ))
          )}
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>{t`Close`}</Button>
        </DialogActions>
      </Dialog>
      <ConfirmDialog
        open={deleting !== null}
        title={t`Delete the template ${deleting ?? ''}?`}
        body={t`Profiles already made from it are not changed.`}
        confirmLabel={t`Delete`}
        color="error"
        busy={busy}
        onCancel={() => setDeleting(null)}
        onConfirm={() => {
          const name = deleting ?? ''
          run(
            () =>
              DeleteTemplate(game, name).then(() => {
                setDeleting(null)
                onChanged()
              }),
            { errorTitle: t`Could not delete the template` },
          )
        }}
      />
    </>
  )
}

export { ManageTemplatesDialog, SaveTemplateDialog }
