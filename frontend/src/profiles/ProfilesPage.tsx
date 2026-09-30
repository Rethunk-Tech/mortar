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
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  Divider,
  IconButton,
  Menu,
  MenuItem,
  type MenuItemProps,
  Typography,
} from '@mui/material'
import {
  ArrowLeft,
  Copy,
  Eye,
  EyeOff,
  GripVertical,
  MoreHorizontal,
  Pencil,
  Plus,
  RotateCcw,
  Trash2,
} from 'lucide-react'
import { type ReactNode, useEffect, useState } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { compact } from '../game/compact.ts'
import { NameField } from '../game/NameField.tsx'
import { NewProfileDialog } from '../game/NewProfileDialog.tsx'
import { useNav } from '../nav/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { userModCount } from './count.ts'
import { useProfiles } from './store.ts'

const DRAG_TINT_ALPHA = 0.24
const panelSx = { bgcolor: 'rgba(50,50,60,0.78)', borderRadius: '6px' }
const menuPaper = {
  paper: {
    sx: {
      width: 220,
      p: 0.75,
      bgcolor: 'rgba(28,28,34,0.99)',
      border: '1px solid rgba(255,255,255,0.14)',
      borderRadius: '8px',
    },
  },
  list: { sx: { p: 0 } },
}

function Item({ icon, sx, children, ...props }: MenuItemProps & { icon: ReactNode }) {
  return (
    <MenuItem
      {...props}
      sx={{ height: 38, gap: '10px', px: '10px', borderRadius: '5px', fontSize: 14, ...sx }}
    >
      {icon}
      {children}
    </MenuItem>
  )
}
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
  const mods = userModCount(profile)
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
        gap: 1.5,
        height: 64,
        px: 1,
        boxSizing: 'border-box',
        mb: '6px',
        ...panelSx,
        border: '2px solid',
        borderColor: isDragging ? 'primary.main' : 'transparent',
        bgcolor: (th) =>
          isDragging ? alpha(th.palette.primary.main, DRAG_TINT_ALPHA) : panelSx.bgcolor,
        position: 'relative',
        zIndex: isDragging ? 1 : 0,
      }}
    >
      <IconButton
        ref={setActivatorNodeRef}
        aria-label={t`Reorder ${profile.name}`}
        {...attributes}
        {...listeners}
        sx={{
          width: 32,
          height: 44,
          borderRadius: '6px',
          color: 'text.secondary',
          cursor: 'grab',
          touchAction: 'none',
        }}
      >
        <GripVertical size={16} />
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
            <Typography noWrap={true} sx={{ fontSize: 17, fontWeight: 600 }}>
              {profile.name}
            </Typography>
            {profile.hidden ? (
              <Box
                component="span"
                sx={{ px: 1, borderRadius: '10px', bgcolor: 'rgba(255,255,255,0.1)', fontSize: 12 }}
              >
                {t`Hidden`}
              </Box>
            ) : null}
          </Box>
        )}
        <Typography noWrap={true} sx={{ fontSize: 13, color: 'text.secondary' }}>
          {t`${modsLabel} · Updated ${updated}`}
        </Typography>
      </Box>
      <IconButton
        aria-label={t`Actions for ${profile.name}`}
        aria-haspopup="menu"
        onClick={(e) => setAnchor(e.currentTarget)}
        sx={{
          width: 40,
          height: 40,
          borderRadius: '6px',
          bgcolor: anchor ? 'rgba(255,255,255,0.1)' : 'transparent',
        }}
      >
        <MoreHorizontal size={18} />
      </IconButton>
      <Menu
        anchorEl={anchor}
        open={anchor !== null}
        onClose={() => setAnchor(null)}
        disableRestoreFocus={true}
        slotProps={menuPaper}
      >
        <Item icon={<Pencil size={15} />} onClick={choose(() => setRenaming(true))}>
          {t`Rename`}
        </Item>
        <Item
          icon={<Copy size={15} />}
          onClick={choose(() => {
            duplicate(profile.id).catch(reportUnexpected)
          })}
        >
          {t`Duplicate`}
        </Item>
        <Item
          icon={profile.hidden ? <Eye size={15} /> : <EyeOff size={15} />}
          onClick={choose(() => {
            setHidden(profile.id, !profile.hidden).catch(reportUnexpected)
          })}
        >
          {profile.hidden ? t`Show in sidebar` : t`Hide from sidebar`}
        </Item>
        <Divider sx={{ my: 0.5 }} />
        <Item
          icon={<Trash2 size={15} />}
          onClick={choose(() => onDelete(profile))}
          sx={{ color: '#ff9a90' }}
        >
          {t`Delete`}
        </Item>
      </Menu>
    </Box>
  )
}

function Trash() {
  const { t } = useLingui()
  const trash = useProfiles((s) => s.trash)
  const restore = useProfiles((s) => s.restore)
  return (
    <Box
      component="aside"
      aria-label={t`Recently deleted`}
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 1.25,
        p: 2,
        alignSelf: 'start',
        bgcolor: 'rgba(40,40,48,0.78)',
        borderRadius: '8px',
      }}
    >
      <Typography component="h2" sx={{ fontSize: 16, fontWeight: 700 }}>
        {t`Recently deleted`}
      </Typography>
      <Typography sx={{ fontSize: 13, lineHeight: 1.45 }}>
        {t`Deleted profiles stay here for 30 days, mods and settings included.`}
      </Typography>
      {trash.map((item) => {
        const days = plural(item.daysLeft, { one: '# day left', other: '# days left' })
        return (
          <Box
            key={item.id}
            sx={{
              display: 'flex',
              alignItems: 'center',
              gap: 1.25,
              p: 1.5,
              bgcolor: 'rgba(55,55,65,0.9)',
              borderRadius: '6px',
            }}
          >
            <Box sx={{ flex: 1, minWidth: 0 }}>
              <Typography noWrap={true} sx={{ fontSize: 15, fontWeight: 600 }}>
                {item.name}
              </Typography>
              <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>{days}</Typography>
            </Box>
            <Button
              variant="outlined"
              startIcon={<RotateCcw size={14} />}
              onClick={() => {
                restore(item.id).catch(reportUnexpected)
              }}
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
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1.5,
          height: 64,
          flexShrink: 0,
          px: 2.5,
          bgcolor: 'background.paper',
        }}
      >
        <ButtonBase
          aria-label={t`Back`}
          onClick={closeProfiles}
          sx={{
            width: 44,
            height: 44,
            borderRadius: '6px',
            '&:hover': { bgcolor: 'action.hover' },
          }}
        >
          <ArrowLeft size={20} />
        </ButtonBase>
        <Typography component="h1" sx={{ fontSize: 26, fontWeight: 700, flex: 1 }}>
          {t`Profiles`}
        </Typography>
        <Button
          variant="contained"
          startIcon={<Plus size={16} />}
          onClick={() => setCreating(true)}
          sx={{ height: 40, px: 2, fontSize: 14 }}
        >
          {t`New profile`}
        </Button>
      </Box>
      <Box
        sx={{
          flex: 1,
          minHeight: 0,
          overflow: 'auto',
          display: 'grid',
          gridTemplateColumns: 'minmax(0, 1fr) 340px',
          alignContent: 'start',
          gap: 2,
          px: 2.5,
          py: 2,
          [compact]: { gridTemplateColumns: 'minmax(0, 1fr)' },
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
