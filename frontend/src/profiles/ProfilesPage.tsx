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
  Divider,
  IconButton,
  InputAdornment,
  ListItemIcon,
  Menu,
  MenuItem,
  TextField,
  Tooltip,
  Typography,
} from '@mui/material'
import {
  ArrowLeft,
  Download,
  FileUp,
  FolderInput,
  Plus,
  RotateCcw,
  Search,
  Trash2,
} from 'lucide-react'
import { useEffect, useId, useState } from 'react'
import type { SourceInfo } from '../../bindings/github.com/Rethunk-AI/mortar/internal/migrate/models.ts'
import type {
  Profile,
  TrashItem,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { BundlesSection } from '../bundles/BundlesSection.tsx'
import { compact } from '../game/compact.ts'
import { NewProfileDialog } from '../game/NewProfileDialog.tsx'
import { useNav } from '../nav/store.ts'
import { openImport } from '../share/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { ExternalImportMenuItems, ExternalImportProfileDialog } from './ExternalImportMenu.tsx'
import { useExternalImportSources } from './externalImportSources.ts'
import { findModInProfiles, openModInProfile } from './findMod.ts'
import { GameModsDialog } from './GameModsDialog.tsx'
import { shouldShowExternalImportDivider } from './importMenu.ts'
import { ProfileRow } from './ProfileRow.tsx'
import { useProfiles } from './store.ts'

function TrashRow({ item }: { item: TrashItem }) {
  const { t } = useLingui()
  const restore = useProfiles((s) => s.restore)
  const purge = useProfiles((s) => s.purge)
  const [pending, run] = usePending()
  const [confirming, setConfirming] = useState(false)
  const days = plural(item.daysLeft, { one: '# day left', other: '# days left' })
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 0.5,
        py: 1,
        pl: 1.5,
        pr: 0.5,
        bgcolor: 'rgba(55,55,65,0.9)',
        borderRadius: '6px',
      }}
    >
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography noWrap={true} title={item.name} sx={{ fontSize: 15, fontWeight: 600 }}>
          {item.name}
        </Typography>
        <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>{days}</Typography>
      </Box>
      <Tooltip title={t`Restore`}>
        <span>
          <IconButton
            aria-label={t`Restore ${item.name}`}
            disabled={pending}
            onClick={() => run(() => restore(item.id))}
          >
            <RotateCcw size={16} />
          </IconButton>
        </span>
      </Tooltip>
      <Tooltip title={t`Delete permanently`}>
        <span>
          <IconButton
            aria-label={t`Delete ${item.name} permanently`}
            color="error"
            disabled={pending}
            onClick={() => setConfirming(true)}
          >
            <Trash2 size={16} />
          </IconButton>
        </span>
      </Tooltip>
      <Dialog open={confirming} onClose={() => setConfirming(false)}>
        <DialogTitle>{t`Delete ${item.name} permanently?`}</DialogTitle>
        <DialogContent>
          <DialogContentText>{t`This cannot be undone.`}</DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirming(false)}>{t`Cancel`}</Button>
          <Button
            color="error"
            variant="contained"
            onClick={() => {
              setConfirming(false)
              run(() => purge(item.id))
            }}
          >
            {t`Delete permanently`}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
  )
}

function Damaged() {
  const { t } = useLingui()
  const damaged = useProfiles((s) => s.damaged)
  const openFolder = useProfiles((s) => s.openFolder)
  const remove = useProfiles((s) => s.remove)
  const [pending, run] = usePending()
  if (damaged.length === 0) {
    return null
  }
  return (
    <Box
      component="section"
      aria-label={t`Damaged`}
      sx={{ display: 'flex', flexDirection: 'column', gap: 1.25, mb: 2 }}
    >
      <Typography component="h2" sx={{ fontSize: 16, fontWeight: 700 }}>{t`Damaged`}</Typography>
      {damaged.map((item) => (
        <Box
          key={item.id}
          sx={{
            display: 'flex',
            alignItems: 'center',
            gap: 0.5,
            py: 1,
            pl: 1.5,
            pr: 0.5,
            bgcolor: 'rgba(55,55,65,0.9)',
            borderRadius: '6px',
          }}
        >
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Typography noWrap={true} title={item.id} sx={{ fontSize: 15, fontWeight: 600 }}>
              {item.id}
            </Typography>
            <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>{item.error}</Typography>
          </Box>
          <Tooltip title={t`Open folder`}>
            <span>
              <IconButton
                aria-label={t`Open folder`}
                disabled={pending}
                onClick={() => run(() => openFolder(item.id))}
              >
                <FolderOpen size={16} />
              </IconButton>
            </span>
          </Tooltip>
          <Tooltip title={t`Move to trash`}>
            <span>
              <IconButton
                aria-label={t`Move to trash`}
                color="error"
                disabled={pending}
                onClick={() => run(() => remove(item.id))}
              >
                <Trash2 size={16} />
              </IconButton>
            </span>
          </Tooltip>
        </Box>
      ))}
    </Box>
  )
}

function Trash() {
  const { t } = useLingui()
  const trash = useProfiles((s) => s.trash)
  const purgeTrash = useProfiles((s) => s.purgeTrash)
  const [confirming, setConfirming] = useState(false)
  const [pending, run] = usePending()
  return (
    <Box
      component="aside"
      aria-label={t`Recently deleted`}
      sx={{
        display: 'flex',
        flexDirection: 'column',
        gap: 1.25,
        p: 2,
        bgcolor: 'rgba(40,40,48,0.78)',
        borderRadius: '8px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography
          component="h2"
          sx={{ fontSize: 16, fontWeight: 700 }}
        >{t`Recently deleted`}</Typography>
        {trash.length === 0 ? null : (
          <Tooltip title={t`Empty trash`}>
            <IconButton
              aria-label={t`Empty trash`}
              color="error"
              disabled={pending}
              onClick={() => setConfirming(true)}
            >
              <Trash2 size={16} />
            </IconButton>
          </Tooltip>
        )}
      </Box>
      <Typography sx={{ fontSize: 13, lineHeight: 1.45 }}>
        {t`Deleted profiles stay here for 30 days, mods and settings included.`}
      </Typography>
      {trash.map((item) => (
        <TrashRow key={item.id} item={item} />
      ))}
      <Dialog open={confirming} onClose={() => setConfirming(false)}>
        <DialogTitle>{t`Empty trash?`}</DialogTitle>
        <DialogContent>
          <DialogContentText>{t`This cannot be undone.`}</DialogContentText>
        </DialogContent>
        <DialogActions>
          <Button onClick={() => setConfirming(false)}>{t`Cancel`}</Button>
          <Button
            color="error"
            variant="contained"
            onClick={() => {
              setConfirming(false)
              run(purgeTrash)
            }}
          >
            {t`Empty trash`}
          </Button>
        </DialogActions>
      </Dialog>
    </Box>
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
  game,
  onBack,
  onImportGame,
  onImport,
  onRestoreZip,
  onCreate,
}: {
  game: string
  onBack: () => void
  onImportGame: () => void
  onImport: () => void
  onRestoreZip: () => void
  onCreate: () => void
}) {
  const { t } = useLingui()
  const importMenuId = useId()
  const externalSources = useExternalImportSources(game)
  const [importAnchor, setImportAnchor] = useState<HTMLElement | null>(null)
  const [externalSource, setExternalSource] = useState<SourceInfo | null>(null)
  const closeImportMenu = () => setImportAnchor(null)
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
        startIcon={<Download size={16} />}
        aria-haspopup="menu"
        aria-expanded={importAnchor !== null}
        aria-controls={importMenuId}
        onClick={(event) => setImportAnchor(event.currentTarget)}
        sx={{ height: 40, px: 2, fontSize: 14, whiteSpace: 'nowrap' }}
      >
        {t`Import`}
      </Button>
      <Menu
        id={importMenuId}
        anchorEl={importAnchor}
        open={importAnchor !== null}
        onClose={closeImportMenu}
      >
        <MenuItem
          onClick={() => {
            closeImportMenu()
            onImportGame()
          }}
        >
          <ListItemIcon sx={{ color: 'inherit' }}>
            <FolderInput size={16} aria-hidden={true} />
          </ListItemIcon>
          {t`From the Mods folder`}
        </MenuItem>
        <MenuItem
          onClick={() => {
            closeImportMenu()
            onImport()
          }}
        >
          <ListItemIcon sx={{ color: 'inherit' }}>
            <Download size={16} aria-hidden={true} />
          </ListItemIcon>
          {t`From a link or file…`}
        </MenuItem>
        <MenuItem
          onClick={() => {
            closeImportMenu()
            onRestoreZip()
          }}
        >
          <ListItemIcon sx={{ color: 'inherit' }}>
            <FileUp size={16} aria-hidden={true} />
          </ListItemIcon>
          {t`From a backup…`}
        </MenuItem>
        {shouldShowExternalImportDivider(externalSources.length) ? <Divider /> : null}
        <ExternalImportMenuItems
          sources={externalSources}
          onPick={(source) => {
            closeImportMenu()
            setExternalSource(source)
          }}
        />
      </Menu>
      <ExternalImportProfileDialog
        game={game}
        source={externalSource}
        onClose={() => setExternalSource(null)}
      />
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
  const damaged = useProfiles((s) => s.damaged)
  const reorder = useProfiles((s) => s.reorder)
  const restoreZip = useProfiles((s) => s.restoreZip)
  const loadTrash = useProfiles((s) => s.loadTrash)
  const [creating, setCreating] = useState(false)
  const [importingGameMods, setImportingGameMods] = useState(false)
  const openProfile = useProfiles((s) => s.open)
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
        game={game}
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
          pt: 2,
          pb: 1.5,
          [compact]: { gridTemplateColumns: 'minmax(0, 1fr)' },
        }}
      >
        <Box sx={{ flex: 1, minWidth: 0 }}>
          <Damaged />
          {profiles.length === 0 && damaged.length === 0 ? (
            <EmptyState
              compact={true}
              icon={<Plus />}
              title={t`No profiles yet.`}
              action={
                <Button
                  variant="contained"
                  startIcon={<Plus size={16} />}
                  onClick={() => setCreating(true)}
                >{t`New profile`}</Button>
              }
            >
              {t`Create a profile to manage a set of mods.`}
            </EmptyState>
          ) : (
            <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
              <SortableContext
                items={profiles.map((p) => p.id)}
                strategy={verticalListSortingStrategy}
              >
                {profiles.map((p) => (
                  <ProfileRow key={p.id} profile={p} />
                ))}
              </SortableContext>
            </DndContext>
          )}
        </Box>
        <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2, width: 320, flexShrink: 0 }}>
          <BundlesSection game={game} profiles={profiles} />
          <Trash />
        </Box>
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
    </Box>
  )
}
