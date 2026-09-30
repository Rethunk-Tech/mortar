import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  KeyboardSensor,
  PointerSensor,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import {
  arrayMove,
  SortableContext,
  sortableKeyboardCoordinates,
  useSortable,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { CSS } from '@dnd-kit/utilities'
import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  alpha,
  Box,
  Button,
  ButtonBase,
  Chip,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  IconButton,
  Menu,
  MenuItem,
  Typography,
} from '@mui/material'
import { ArrowLeft, GripVertical, MoreHorizontal, Plus, RotateCcw } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { compact } from '../game/compact.ts'
import { NameField } from '../game/NameField.tsx'
import { NewProfileDialog } from '../game/NewProfileDialog.tsx'
import { useNav } from '../nav/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { useProfiles } from './store.ts'

const DRAG_TINT_ALPHA = 0.24
const panelSx = { bgcolor: 'background.paper', borderRadius: '6px' }
const dialogPaper = { paper: { sx: { bgcolor: 'rgba(40,40,48,0.92)' } } }

function Row({ profile, onDelete }: { profile: Profile; onDelete: (p: Profile) => void }) {
  const { t } = useLingui()
  const rename = useProfiles((s) => s.rename)
  const duplicate = useProfiles((s) => s.duplicate)
  const setHidden = useProfiles((s) => s.setHidden)
  const [renaming, setRenaming] = useState(false)
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const {
    attributes,
    listeners,
    setNodeRef,
    setActivatorNodeRef,
    transform,
    transition,
    isDragging,
  } = useSortable({ id: profile.id })
  const mods = (profile.entries ?? []).reduce((n, e) => n + (e.mods ?? []).length, 0)
  const modsLabel = plural(mods, { one: '# mod', other: '# mods' })
  const updated = new Date(String(profile.updated)).toLocaleDateString()
  const choose = (run: () => void) => () => {
    setAnchor(null)
    run()
  }
  return (
    <Box
      ref={setNodeRef}
      style={{
        transform: CSS.Transform.toString(transform && { ...transform, x: 0 }),
        transition,
      }}
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        px: 1,
        py: 1,
        mb: 1,
        ...panelSx,
        border: '2px solid',
        borderColor: isDragging ? 'primary.main' : 'transparent',
        bgcolor: (th) =>
          isDragging ? alpha(th.palette.primary.main, DRAG_TINT_ALPHA) : 'background.paper',
        position: 'relative',
        zIndex: isDragging ? 1 : 0,
      }}
    >
      <IconButton
        ref={setActivatorNodeRef}
        aria-label={t`Reorder ${profile.name}`}
        {...attributes}
        {...listeners}
        sx={{ cursor: 'grab', touchAction: 'none' }}
      >
        <GripVertical size={18} />
      </IconButton>
      <Box sx={{ flex: 1, minWidth: 0 }}>
        {renaming ? (
          <NameField
            size="small"
            initial={profile.name}
            label={t`Profile name`}
            onSubmit={(name) => rename(profile.id, name)}
            onCancel={() => setRenaming(false)}
          />
        ) : (
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography noWrap={true} sx={{ fontSize: 16, fontWeight: 600 }}>
              {profile.name}
            </Typography>
            {profile.hidden ? <Chip size="small" label={t`Hidden`} /> : null}
          </Box>
        )}
        <Typography noWrap={true} sx={{ fontSize: 13, color: 'text.secondary' }}>
          {t`${modsLabel} · Updated ${updated}`}
        </Typography>
      </Box>
      <IconButton
        aria-label={t`Actions for ${profile.name}`}
        onClick={(e) => setAnchor(e.currentTarget)}
      >
        <MoreHorizontal size={18} />
      </IconButton>
      <Menu
        anchorEl={anchor}
        open={anchor !== null}
        onClose={() => setAnchor(null)}
        disableRestoreFocus={true}
        slotProps={dialogPaper}
      >
        <MenuItem onClick={choose(() => setRenaming(true))}>{t`Rename`}</MenuItem>
        <MenuItem
          onClick={choose(() => {
            duplicate(profile.id).catch(reportUnexpected)
          })}
        >
          {t`Duplicate`}
        </MenuItem>
        <MenuItem
          onClick={choose(() => {
            setHidden(profile.id, !profile.hidden).catch(reportUnexpected)
          })}
        >
          {profile.hidden ? t`Show in sidebar` : t`Hide from sidebar`}
        </MenuItem>
        <MenuItem onClick={choose(() => onDelete(profile))}>{t`Delete`}</MenuItem>
      </Menu>
    </Box>
  )
}

function Trash() {
  const { t } = useLingui()
  const trash = useProfiles((s) => s.trash)
  const restore = useProfiles((s) => s.restore)
  if (trash.length === 0) {
    return null
  }
  return (
    <Box
      component="section"
      aria-label={t`Recently deleted`}
      sx={{
        ...panelSx,
        width: 300,
        flexShrink: 0,
        alignSelf: 'flex-start',
        p: 2,
        [compact]: { width: 'auto', alignSelf: 'stretch' },
      }}
    >
      <Typography component="h2" sx={{ fontSize: 15, fontWeight: 700, mb: 1 }}>
        {t`Recently deleted`}
      </Typography>
      {trash.map((item) => {
        const days = plural(item.daysLeft, { one: '# day left', other: '# days left' })
        return (
          <Box key={item.id} sx={{ display: 'flex', alignItems: 'center', gap: 1, py: 0.75 }}>
            <Box sx={{ flex: 1, minWidth: 0 }}>
              <Typography noWrap={true} sx={{ fontSize: 14 }}>
                {item.name}
              </Typography>
              <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>{days}</Typography>
            </Box>
            <Button
              size="small"
              startIcon={<RotateCcw size={14} />}
              onClick={() => {
                restore(item.id).catch(reportUnexpected)
              }}
              sx={{ whiteSpace: 'nowrap' }}
            >
              {t`Restore`}
            </Button>
          </Box>
        )
      })}
    </Box>
  )
}

export function ProfilesPage() {
  const { t } = useLingui()
  const closeProfiles = useNav((s) => s.closeProfiles)
  const profiles = useProfiles((s) => s.profiles)
  const reorder = useProfiles((s) => s.reorder)
  const remove = useProfiles((s) => s.remove)
  const loadTrash = useProfiles((s) => s.loadTrash)
  const [creating, setCreating] = useState(false)
  const [deleting, setDeleting] = useState<Profile | null>(null)
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )
  useEffect(() => {
    loadTrash().catch(reportUnexpected)
  }, [loadTrash])
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape' && !e.defaultPrevented) {
        closeProfiles()
      }
    }
    globalThis.addEventListener('keydown', onKey)
    return () => globalThis.removeEventListener('keydown', onKey)
  }, [closeProfiles])
  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (!over || active.id === over.id) {
      return
    }
    const ids = profiles.map((p) => p.id)
    reorder(arrayMove(ids, ids.indexOf(String(active.id)), ids.indexOf(String(over.id)))).catch(
      reportUnexpected,
    )
  }
  return (
    <Box sx={{ height: '100%', display: 'flex', flexDirection: 'column' }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1.5, px: 2, py: 1.5 }}>
        <ButtonBase
          aria-label={t`Back`}
          onClick={closeProfiles}
          sx={{
            width: 36,
            height: 36,
            borderRadius: '6px',
            '&:hover': { bgcolor: 'action.hover' },
          }}
        >
          <ArrowLeft size={20} />
        </ButtonBase>
        <Typography component="h1" sx={{ fontSize: 22, fontWeight: 600, flex: 1 }}>
          {t`Profiles`}
        </Typography>
        <Button
          variant="contained"
          startIcon={<Plus size={16} />}
          onClick={() => setCreating(true)}
          sx={{ whiteSpace: 'nowrap' }}
        >
          {t`New profile`}
        </Button>
      </Box>
      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          overflow: 'auto',
          display: 'flex',
          gap: 2,
          px: 2,
          pb: 2,
          [compact]: { flexDirection: 'column' },
        }}
      >
        <Box sx={{ flex: 1, minWidth: 0 }}>
          {profiles.length === 0 ? (
            <Typography sx={{ color: 'text.secondary' }}>{t`No profiles yet.`}</Typography>
          ) : (
            <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
              <SortableContext
                items={profiles.map((p) => p.id)}
                strategy={verticalListSortingStrategy}
              >
                {profiles.map((p) => (
                  <Row key={p.id} profile={p} onDelete={setDeleting} />
                ))}
              </SortableContext>
            </DndContext>
          )}
        </Box>
        <Trash />
      </Box>
      <NewProfileDialog open={creating} onClose={() => setCreating(false)} />
      <Dialog open={deleting !== null} onClose={() => setDeleting(null)} slotProps={dialogPaper}>
        <DialogTitle>{t`Delete ${deleting?.name ?? ''}?`}</DialogTitle>
        <DialogContent>
          <DialogContentText>
            {t`The profile stays restorable for 30 days from Recently deleted.`}
          </DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setDeleting(null)}>{t`Cancel`}</Button>
          <Button
            variant="contained"
            color="error"
            onClick={() => {
              if (deleting) {
                remove(deleting.id).catch(reportUnexpected)
              }
              setDeleting(null)
            }}
          >
            {t`Delete`}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}
