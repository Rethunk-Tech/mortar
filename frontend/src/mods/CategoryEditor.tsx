import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  IconButton,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'
import { Plus, Trash2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { CustomCategory } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { colorHex, PROFILE_COLORS } from '../profiles/appearance.ts'
import { useProfiles } from '../profiles/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useCustomCategories } from './customCategories.ts'

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
    <Box sx={{ display: 'flex', gap: 1, alignItems: 'flex-start', mb: 1.5 }}>
      <TextField
        size="small"
        label={t`Name`}
        value={row.name}
        onChange={(e) => onChange({ ...row, name: e.target.value })}
        sx={{ flex: 1 }}
      />
      <Box sx={{ display: 'flex', flexWrap: 'wrap', gap: 0.5, pt: 0.5, maxWidth: 180 }}>
        {PROFILE_COLORS.map((token) => (
          <Tooltip key={token} title={token}>
            <IconButton
              aria-label={token}
              aria-pressed={row.color === token}
              size="small"
              onClick={() => onChange({ ...row, color: row.color === token ? '' : token })}
              sx={{
                width: 28,
                height: 28,
                bgcolor: colorHex(token),
                outline: row.color === token ? '2px solid #fff' : '2px solid transparent',
                outlineOffset: 1,
              }}
            />
          </Tooltip>
        ))}
      </Box>
      <Tooltip title={t`Delete category`}>
        <IconButton aria-label={t`Delete category`} onClick={onDelete} size="small">
          <Trash2 size={16} />
        </IconButton>
      </Tooltip>
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
  const [busy, setBusy] = useState(false)

  useEffect(() => {
    if (open && gameId !== '') {
      load(gameId).catch(reportUnexpected)
    }
  }, [open, gameId, load])

  useEffect(() => {
    if (open) {
      setDraft(stored.map((c) => ({ ...c })))
    }
  }, [open, stored])

  const persist = async (next: CustomCategory[]) => {
    if (gameId === '') {
      return
    }
    setBusy(true)
    try {
      const saved = await save(gameId, next)
      setDraft(saved.map((c) => ({ ...c })))
    } catch (e) {
      reportUnexpected(e)
    } finally {
      setBusy(false)
    }
  }

  const confirmDelete = async () => {
    if (!pendingDelete) {
      return
    }
    const next = draft.filter((c) => c.id !== pendingDelete.id)
    setPendingDelete(null)
    await persist(next)
  }

  return (
    <>
      <Dialog open={open} onClose={onClose} transitionDuration={0} maxWidth="sm" fullWidth={true}>
        <DialogTitle>{t`Custom categories`}</DialogTitle>
        <DialogContent>
          <Typography sx={{ fontSize: 13, color: 'text.secondary', mb: 2 }}>
            {t`Group mods by these names, or assign one as a mod's primary category.`}
          </Typography>
          {draft.map((row) => (
            <CategoryRow
              key={row.id || row.name}
              row={row}
              onChange={(next) => {
                const rows = draft.map((c) => (c === row ? next : c))
                setDraft(rows)
              }}
              onDelete={() => setPendingDelete(row)}
            />
          ))}
          <Button
            startIcon={<Plus size={14} />}
            onClick={() => setDraft([...draft, { id: '', name: '', color: '' }])}
            disabled={busy}
          >
            {t`Add category`}
          </Button>
        </DialogContent>
        <DialogActions>
          <Button onClick={onClose}>{t`Close`}</Button>
          <Button
            variant="contained"
            disabled={busy || gameId === ''}
            onClick={() => persist(draft).catch(reportUnexpected)}
          >
            {t`Save`}
          </Button>
        </DialogActions>
      </Dialog>
      <Dialog
        open={pendingDelete !== null}
        onClose={() => setPendingDelete(null)}
        transitionDuration={0}
      >
        <DialogTitle>{t`Delete ${pendingDelete?.name ?? ''}?`}</DialogTitle>
        <DialogContent>
          <Typography>{t`Mods in this category will move to Uncategorized.`}</Typography>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setPendingDelete(null)}>{t`Cancel`}</Button>
          <Button color="error" onClick={() => confirmDelete().catch(reportUnexpected)}>
            {t`Delete`}
          </Button>
        </DialogActions>
      </Dialog>
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
    { label: t`Uncategorized`, value: '' },
    ...categories.map((c) => ({ label: c.name, value: c.id })),
  ]
  if (nexusCategory !== '' && !options.some((o) => o.value === nexusCategory)) {
    options.push({ label: nexusCategory, value: nexusCategory })
  }

  return (
    <Dialog open={open} onClose={onClose} transitionDuration={0}>
      <DialogTitle>{t`Set category`}</DialogTitle>
      <DialogContent sx={{ display: 'flex', flexDirection: 'column', gap: 0.5, minWidth: 280 }}>
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
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Cancel`}</Button>
      </DialogActions>
    </Dialog>
  )
}

export { CategoryEditorDialog, SetCategoryDialog }
