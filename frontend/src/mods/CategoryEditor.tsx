import { useLingui } from '@lingui/react/macro'
import { Box, Button, TextField } from '@mui/material'
import { Plus, Tags, Trash2 } from 'lucide-react'
import { useEffect, useLayoutEffect, useState } from 'react'
import type { CustomCategory } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { ColorSwatches } from '../profiles/Swatches.tsx'
import { useProfiles } from '../profiles/store.ts'
import { AppDialog, DialogFooter, ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { FormDialog } from '../shell/dialog/FormDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { IconAction } from '../shell/IconAction.tsx'
import { space } from '../theme/density.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { categoriesDiffer, useCustomCategories } from './customCategories.ts'

// A row's React key survives its edits (each one makes a new object) and unsaved rows, which have no id yet.
const rowKeys = new WeakMap<CustomCategory, number>()
let lastRowKey = 0
function rowKey(row: CustomCategory): number {
  let key = rowKeys.get(row)
  if (key === undefined) {
    lastRowKey += 1
    key = lastRowKey
    rowKeys.set(row, key)
  }
  return key
}

function CategoryRow({
  row,
  onChange,
  onDelete,
}: {
  row: CustomCategory
  onChange: (next: CustomCategory) => void
  onDelete: () => void
}) {
  const { t } = useLingui()
  return (
    <Box sx={{ display: 'flex', gap: space.gap, alignItems: 'flex-start', mb: 1.5 }}>
      <TextField
        size="small"
        placeholder={t`Name`}
        slotProps={{ htmlInput: { 'aria-label': t`Name` } }}
        value={row.name}
        onChange={(e) => onChange({ ...row, name: e.target.value })}
        sx={{ flex: 1 }}
      />
      <Box sx={{ pt: 0.5, maxWidth: 180 }}>
        <ColorSwatches
          label={t`Colour`}
          value={row.color ?? ''}
          onChange={(color) => onChange({ ...row, color })}
        />
      </Box>
      <IconAction label={t`Delete category`} icon={<Trash2 size={16} />} onClick={onDelete} />
    </Box>
  )
}

function CategoryEditorDialog({ open, onClose }: { open: boolean; onClose: () => void }) {
  const { t } = useLingui()
  const gameId = useProfiles((s) => s.game?.id) ?? ''
  const stored = useCustomCategories((s) => s.categories)
  const save = useCustomCategories((s) => s.save)
  const load = useCustomCategories((s) => s.load)
  const [draft, setDraft] = useState<CustomCategory[]>([])
  const [pendingDelete, setPendingDelete] = useState<CustomCategory | null>(null)
  const [pending, run] = usePending()

  useEffect(() => {
    if (open && gameId !== '') {
      load(gameId).catch(reportUnexpected)
    }
  }, [open, gameId, load])

  // Copied before paint, so opening never flashes the empty state over saved categories.
  useLayoutEffect(() => {
    if (open) {
      setDraft(stored.map((c) => ({ ...c })))
    }
  }, [open, stored])

  const persist = (next: CustomCategory[]) => {
    if (gameId === '') {
      return
    }
    run(async () => {
      const saved = await save(gameId, next)
      setDraft(saved.map((c) => ({ ...c })))
    })
  }

  const dirty = categoriesDiffer(draft, stored)

  return (
    <>
      <FormDialog
        open={open}
        onClose={onClose}
        size="sm"
        title={t`Custom categories`}
        subtitle={t`Group mods by these names, or assign one as a mod's primary category.`}
        icon={
          <ToneIcon tone="neutral">
            <Tags size={22} />
          </ToneIcon>
        }
        submitLabel={t`Save`}
        issue={gameId === '' ? t`Pick a game first.` : ''}
        dirty={dirty}
        busy={pending}
        onSubmit={() => persist(draft)}
      >
        <Box>
          {draft.length === 0 ? (
            <EmptyState
              compact={true}
              icon={<Tags size={28} />}
              title={t`No custom categories yet`}
            >
              {t`Add one to group your mods your own way.`}
            </EmptyState>
          ) : null}
          {draft.map((row) => (
            <CategoryRow
              key={rowKey(row)}
              row={row}
              onChange={(next) => {
                rowKeys.set(next, rowKey(row))
                const rows = draft.map((c) => (c === row ? next : c))
                setDraft(rows)
              }}
              onDelete={() => setPendingDelete(row)}
            />
          ))}
          <Button
            startIcon={<Plus size={14} />}
            onClick={() => setDraft([...draft, { id: '', name: '', color: '' }])}
            disabled={pending}
          >
            {t`Add category`}
          </Button>
        </Box>
      </FormDialog>
      <ConfirmationDialog
        open={pendingDelete !== null}
        title={t`Delete ${{ name: pendingDelete?.name ?? '' }}?`}
        body={t`Mods in this category will move to Uncategorised.`}
        confirmLabel={t`Delete`}
        tone="danger"
        busy={pending}
        onCancel={() => setPendingDelete(null)}
        onConfirm={() => {
          if (!pendingDelete) {
            return
          }
          const next = draft.filter((c) => c !== pendingDelete)
          setPendingDelete(null)
          persist(next)
        }}
      />
    </>
  )
}

interface SetCategoryDialogProps {
  open: boolean
  onClose: () => void
  modKey: string
  nexusCategory: string
  currentOverride: string
}

function SetCategoryDialog({
  open,
  onClose,
  modKey,
  nexusCategory,
  currentOverride,
}: SetCategoryDialogProps) {
  const { t } = useLingui()
  const categories = useCustomCategories((s) => s.categories)
  const setEntryCategory = useCustomCategories((s) => s.setEntryCategory)

  const pick = (override: string) => {
    setEntryCategory(modKey, override).then(onClose).catch(reportUnexpected)
  }

  const options: { label: string; value: string }[] = [
    { label: t`Uncategorised`, value: '' },
    ...categories.map((c) => ({ label: c.name, value: c.id })),
  ]
  if (nexusCategory !== '' && !options.some((o) => o.value === nexusCategory)) {
    options.push({ label: nexusCategory, value: nexusCategory })
  }

  return (
    <AppDialog
      open={open}
      onClose={onClose}
      size="sm"
      title={t`Set category`}
      icon={
        <ToneIcon tone="neutral">
          <Tags size={22} />
        </ToneIcon>
      }
      footer={
        <DialogFooter>
          <Button onClick={onClose}>{t`Cancel`}</Button>
        </DialogFooter>
      }
    >
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
        {options.map((opt) => (
          <Button
            key={opt.value || 'none'}
            variant={currentOverride === opt.value ? 'contained' : 'text'}
            onClick={() => pick(opt.value)}
            sx={{ justifyContent: 'flex-start' }}
          >
            {opt.label}
          </Button>
        ))}
      </Box>
    </AppDialog>
  )
}

export { CategoryEditorDialog, SetCategoryDialog }
