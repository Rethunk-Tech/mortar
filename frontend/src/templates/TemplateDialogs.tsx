import { msg, plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  List,
  ListItem,
  ListItemText,
  Tooltip,
} from '@mui/material'
import { LayoutTemplate, Pencil, Trash2 } from 'lucide-react'
import { useRef, useState } from 'react'
import type { Template } from '../../bindings/github.com/Rethunk-AI/mortar/internal/templates/models.ts'
import {
  DeleteTemplate,
  RenameTemplate,
  RestoreTemplate,
  SaveTemplateFromProfile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/templates/service.ts'
import { i18n } from '../i18n/index.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { PromptDialog } from '../shell/PromptDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { useTemplates } from './useTemplates.ts'

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
  const { templates } = useTemplates(game, open)
  const [replacing, setReplacing] = useState<string | null>(null)
  const save = (name: string) =>
    run(
      () =>
        SaveTemplateFromProfile(game, profileId, name).then(() => {
          useToasts.getState().push({ kind: 'success', title: t`Template saved` })
          setReplacing(null)
          onClose()
        }),
      { errorTitle: t`Could not save the template` },
    )
  return (
    <>
      <PromptDialog
        open={open && replacing === null}
        title={t`Save as template`}
        label={t`Template name`}
        initial={profileName}
        maxLength={MAX_NAME}
        confirmLabel={t`Save`}
        busy={busy}
        onCancel={onClose}
        onSubmit={(name) => {
          if (templates.some((x) => x.name.toLowerCase() === name.toLowerCase())) {
            setReplacing(name)
            return
          }
          save(name)
        }}
      />
      <ConfirmDialog
        open={open && replacing !== null}
        title={t`Replace ${replacing ?? ''}?`}
        body={t`A template with this name already exists.`}
        confirmLabel={t`Save as template`}
        busy={busy}
        onCancel={() => setReplacing(null)}
        onConfirm={() => save(replacing ?? '')}
      />
    </>
  )
}

function TemplateRow({
  template,
  onRename,
  onDelete,
}: {
  template: Template
  onRename: () => void
  onDelete: () => void
}) {
  const { t } = useLingui()
  const mods = plural(template.bundle?.length ?? 0, { one: '# mod', other: '# mods' })
  return (
    <ListItem
      disableGutters={true}
      secondaryAction={
        <>
          <Tooltip title={t`Rename`}>
            <IconButton size="small" aria-label={t`Rename ${template.name}`} onClick={onRename}>
              <Pencil size={16} />
            </IconButton>
          </Tooltip>
          <Tooltip title={t`Delete`}>
            <IconButton size="small" aria-label={t`Delete ${template.name}`} onClick={onDelete}>
              <Trash2 size={16} />
            </IconButton>
          </Tooltip>
        </>
      }
    >
      <ListItemText
        primary={template.name}
        secondary={mods}
        slotProps={{ primary: { noWrap: true, title: template.name } }}
      />
    </ListItem>
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
  const [deleting, setDeleting] = useState<Template | null>(null)
  const [renaming, setRenaming] = useState<string | null>(null)
  const [busy, run] = usePending()
  const closeRef = useRef<HTMLButtonElement>(null)
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
            <List dense={true}>
              {templates.map((template) => (
                <TemplateRow
                  key={template.name}
                  template={template}
                  onRename={() => setRenaming(template.name)}
                  onDelete={() => setDeleting(template)}
                />
              ))}
            </List>
          )}
        </DialogContent>
        <DialogActions>
          <Button ref={closeRef} onClick={onClose}>
            {t`Close`}
          </Button>
        </DialogActions>
      </Dialog>
      <PromptDialog
        open={renaming !== null}
        title={t`Rename template`}
        label={t`Template name`}
        initial={renaming ?? ''}
        maxLength={MAX_NAME}
        confirmLabel={t`Rename`}
        busy={busy}
        onCancel={() => setRenaming(null)}
        onSubmit={(name) =>
          run(
            () =>
              RenameTemplate(game, renaming ?? '', name).then(() => {
                setRenaming(null)
                onChanged()
              }),
            { errorTitle: t`Could not rename the template` },
          )
        }
      />
      <ConfirmDialog
        open={deleting !== null}
        title={t`Delete the template ${deleting?.name ?? ''}?`}
        body={t`Profiles already made from it are not changed.`}
        confirmLabel={t`Delete`}
        color="error"
        busy={busy}
        onCancel={() => setDeleting(null)}
        onConfirm={() => {
          const gone = deleting
          if (gone === null) {
            return
          }
          run(
            async () => {
              await DeleteTemplate(game, gone.name)
              setDeleting(null)
              onChanged()
              setTimeout(() => closeRef.current?.focus(), 0)
              useToasts.getState().push({
                kind: 'success',
                title: i18n._(msg`Template deleted`),
                action: {
                  label: i18n._(msg`Undo`),
                  run: () => RestoreTemplate(game, gone).then(onChanged).catch(reportUnexpected),
                },
              })
            },
            { errorTitle: t`Could not delete the template` },
          )
        }}
      />
    </>
  )
}

export { ManageTemplatesDialog, SaveTemplateDialog }
