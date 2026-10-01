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
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  ButtonBase,
  Dialog,
  DialogActions,
  DialogContent,
  DialogContentText,
  DialogTitle,
  InputAdornment,
  TextField,
  Typography,
} from '@mui/material'
import { ArrowLeft, Download, FileUp, FolderInput, Plus, RotateCcw, Search } from 'lucide-react'
import { useEffect, useRef, useState } from 'react'
import type {
  Profile,
  TrashItem,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { compact } from '../game/compact.ts'
import { NewProfileDialog } from '../game/NewProfileDialog.tsx'
import { useNav } from '../nav/store.ts'
import { openImport } from '../share/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { CompareDialog, PickCompareDialog } from './CompareDialog.tsx'
import { findModInProfiles, openModInProfile } from './findMod.ts'
import { GameModsDialog } from './GameModsDialog.tsx'
import { ProfileRow } from './ProfileRow.tsx'
import { useProfiles } from './store.ts'

function TrashRow({ item }: { item: TrashItem }) {
  const { t } = useLingui()
  const restore = useProfiles((s) => s.restore)
  const [pending, run] = usePending()
  const days = plural(item.daysLeft, { one: '# day left', other: '# days left' })
  return (
    <Box
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
        disabled={pending}
        onClick={() => run(() => restore(item.id))}
        sx={{ whiteSpace: 'nowrap' }}
      >
        {t`Restore`}
      </Button>
    </Box>
  )
}

function Trash() {
  const { t } = useLingui()
  const trash = useProfiles((s) => s.trash)
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
      {trash.map((item) => (
        <TrashRow key={item.id} item={item} />
      ))}
    </Box>
  )
}

function DeleteDialog({ deleting, onDone }: { deleting: Profile | null; onDone: () => void }) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
  const remove = useProfiles((s) => s.remove)
  // The row whose ⋯ button takes focus once the dialog has closed: this one, or a neighbour when it is deleted.
  const focusAfter = useRef('')
  const close = (focusId: string) => {
    focusAfter.current = focusId
    onDone()
  }
  const cancel = () => close(deleting?.id ?? '')
  return (
    <Dialog
      open={deleting !== null}
      onClose={cancel}
      slotProps={{
        transition: {
          onExited: () => {
            document
              .querySelector<HTMLElement>(
                `[data-actions="${globalThis.CSS.escape(focusAfter.current)}"]`,
              )
              ?.focus()
          },
        },
      }}
    >
      <DialogTitle>{t`Delete ${deleting?.name ?? ''}?`}</DialogTitle>
      <DialogContent>
        <DialogContentText>
          {t`The profile stays restorable for 30 days from Recently deleted.`}
        </DialogContentText>
      </DialogContent>
      <DialogActions>
        <Button onClick={cancel}>{t`Cancel`}</Button>
        <Button
          variant="contained"
          color="error"
          onClick={() => {
            if (deleting) {
              const at = profiles.findIndex((p) => p.id === deleting.id)
              close((profiles[at + 1] ?? profiles[at - 1])?.id ?? '')
              remove(deleting.id).catch(reportUnexpected)
            }
          }}
        >
          {t`Delete`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

function FindModSearch({ profiles }: { profiles: Profile[] }) {
  const { t } = useLingui()
  const [query, setQuery] = useState('')
  const hits = findModInProfiles(profiles, query)
  return (
    <Box sx={{ px: 2.5, pt: 1.5, flexShrink: 0 }}>
      <TextField
        size="small"
        fullWidth={true}
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        placeholder={t`Find a mod in all profiles`}
        slotProps={{
          input: {
            startAdornment: (
              <InputAdornment position="start">
                <Search size={16} aria-hidden={true} />
              </InputAdornment>
            ),
          },
        }}
      />
      {hits.length > 0 ? (
        <Box sx={{ mt: 1, display: 'flex', flexDirection: 'column', gap: 0.25 }}>
          {hits.map((h) => (
            <Button
              key={`${h.profileId}/${h.key}/${h.uniqueId}`}
              onClick={() => openModInProfile(h)}
              sx={{
                whiteSpace: 'nowrap',
                justifyContent: 'flex-start',
                textTransform: 'none',
                fontSize: 13,
              }}
            >
              {`${h.name} · ${h.uniqueId} · ${h.profileName} · ${h.version} · ${h.enabled ? t`Enabled` : t`Switched off`}`}
            </Button>
          ))}
        </Box>
      ) : null}
    </Box>
  )
}

function ProfilesHeader({
  onBack,
  onImportGame,
  onImport,
  onRestoreZip,
  onCreate,
}: {
  onBack: () => void
  onImportGame: () => void
  onImport: () => void
  onRestoreZip: () => void
  onCreate: () => void
}) {
  const { t } = useLingui()
  return (
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
        onClick={onBack}
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
        variant="outlined"
        color="inherit"
        startIcon={<FolderInput size={16} />}
        onClick={onImportGame}
        sx={{ height: 40, px: 2, fontSize: 14, whiteSpace: 'nowrap' }}
      >
        {t`Import from the game's Mods folder`}
      </Button>
      <Button
        variant="outlined"
        color="inherit"
        startIcon={<Download size={16} />}
        onClick={onImport}
        sx={{ height: 40, px: 2, fontSize: 14, whiteSpace: 'nowrap' }}
      >
        {t`Import`}
      </Button>
      <Button
        variant="outlined"
        color="inherit"
        startIcon={<FileUp size={16} />}
        onClick={onRestoreZip}
        sx={{ height: 40, px: 2, fontSize: 14, whiteSpace: 'nowrap' }}
      >
        {t`Restore from zip…`}
      </Button>
      <Button
        variant="contained"
        startIcon={<Plus size={16} />}
        onClick={onCreate}
        sx={{ height: 40, px: 2, fontSize: 14 }}
      >
        {t`New profile`}
      </Button>
    </Box>
  )
}

export function ProfilesPage() {
  const { t } = useLingui()
  const closeProfiles = useNav((s) => s.closeProfiles)
  const game = useNav((s) => (s.route.name === 'profiles' ? s.route.game : 'stardew'))
  const profiles = useProfiles((s) => s.profiles)
  const reorder = useProfiles((s) => s.reorder)
  const restoreZip = useProfiles((s) => s.restoreZip)
  const loadTrash = useProfiles((s) => s.loadTrash)
  const [creating, setCreating] = useState(false)
  const [importingGameMods, setImportingGameMods] = useState(false)
  const openProfile = useProfiles((s) => s.open)
  const [deleting, setDeleting] = useState<Profile | null>(null)
  const [pickFor, setPickFor] = useState<Profile | null>(null)
  const [compare, setCompare] = useState<{ a: Profile; b: Profile } | null>(null)
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
      <ProfilesHeader
        onBack={closeProfiles}
        onImportGame={() => setImportingGameMods(true)}
        onImport={() => openImport()}
        onRestoreZip={() => restoreZip().catch(reportUnexpected)}
        onCreate={() => setCreating(true)}
      />
      <FindModSearch profiles={profiles} />
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
                  <ProfileRow
                    key={p.id}
                    profile={p}
                    onDelete={setDeleting}
                    onCompare={setPickFor}
                    canCompare={profiles.length > 1}
                  />
                ))}
              </SortableContext>
            </DndContext>
          )}
        </Box>
        <Trash />
      </Box>
      <NewProfileDialog open={creating} onClose={() => setCreating(false)} />
      <GameModsDialog
        open={importingGameMods}
        game={game}
        onClose={() => setImportingGameMods(false)}
        onImported={(id) => {
          openProfile(id)
          closeProfiles()
        }}
      />
      <DeleteDialog deleting={deleting} onDone={() => setDeleting(null)} />
      <PickCompareDialog
        from={pickFor}
        onPicked={(other) => {
          if (pickFor) {
            setCompare({ a: pickFor, b: other })
          }
          setPickFor(null)
        }}
        onClose={() => setPickFor(null)}
      />
      <CompareDialog
        a={compare?.a ?? null}
        b={compare?.b ?? null}
        onClose={() => setCompare(null)}
      />
    </Box>
  )
}
