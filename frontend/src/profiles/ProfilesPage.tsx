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
import { Box, Button, ButtonBase, Menu, Typography } from '@mui/material'
import {
  ArrowLeft,
  Download,
  FileUp,
  FolderInput,
  FolderOpen,
  Plus,
  RotateCcw,
  SearchX,
  Trash2,
  Wrench,
} from 'lucide-react'
import { useEffect, useId, useRef, useState } from 'react'
import type {
  Profile,
  TrashItem,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { BundlesSection } from '../bundles/BundlesSection.tsx'
import { compact } from '../game/compact.ts'
import { NewProfileDialog } from '../game/NewProfileDialog.tsx'
import { useProfilePageBadges } from '../game/useSidebarProfiles.ts'
import { localId } from '../mods/dependents.ts'
import { useCurrentGame } from '../nav/currentGame.ts'
import { useNav } from '../nav/store.ts'
import { dialogOpen, isTypingTarget } from '../settings/shortcuts.ts'
import { shouldLeavePageOnEscape } from '../settings/shouldLeavePageOnEscape.ts'
import { openImport } from '../share/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { TipIconButton } from '../shell/TipIconButton.tsx'
import { errorDetails } from '../toasts/errorKind.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { usePending } from '../toasts/usePending.ts'
import { findModInProfiles, onFindAllFocus, openModInProfile } from './findMod.ts'
import { GameModsDialog } from './GameModsDialog.tsx'
import { ImportWizard } from './ImportWizard.tsx'
import { PackImportDialog } from './PackImportDialog.tsx'
import { ProfileRow } from './ProfileRow.tsx'
import { hasThunderstore } from './packImport.ts'
import { useProfiles } from './store.ts'
import { useBudgets } from './useBudgets.ts'

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
        bgcolor: 'var(--mortar-raised)',
        borderRadius: '6px',
      }}
    >
      <Box sx={{ flex: 1, minWidth: 0 }}>
        <Typography noWrap={true} title={item.name} sx={{ fontSize: 15, fontWeight: 600 }}>
          {item.name}
        </Typography>
        <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>{days}</Typography>
      </Box>
      <TipIconButton
        label={t`Restore ${{ label: item.name }}`}
        disabled={pending}
        onClick={() => run(() => restore(item.id))}
      >
        <RotateCcw size={16} />
      </TipIconButton>
      <TipIconButton
        label={t`Delete ${item.name} permanently`}
        color="error"
        disabled={pending}
        onClick={() => setConfirming(true)}
      >
        <Trash2 size={16} />
      </TipIconButton>
      <ConfirmDialog
        open={confirming}
        title={t`Delete ${item.name} permanently?`}
        body={t`This cannot be undone.`}
        confirmLabel={t`Delete permanently`}
        color="error"
        onCancel={() => setConfirming(false)}
        onConfirm={() => {
          setConfirming(false)
          run(() => purge(item.id))
        }}
      />
    </Box>
  )
}

function Damaged() {
  const { t } = useLingui()
  const damaged = useProfiles((s) => s.damaged)
  const openFolder = useProfiles((s) => s.openFolder)
  const repair = useProfiles((s) => s.repair)
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
            bgcolor: 'var(--mortar-raised)',
            borderRadius: '6px',
          }}
        >
          <Box sx={{ flex: 1, minWidth: 0 }}>
            <Typography noWrap={true} title={item.id} sx={{ fontSize: 15, fontWeight: 600 }}>
              {item.id.slice(Math.max(item.id.lastIndexOf('/'), item.id.lastIndexOf('\\')) + 1)}
            </Typography>
            <Typography
              sx={{ fontSize: 13, color: 'text.secondary' }}
              title={errorDetails(item.error)}
            >
              {t`Could not read this profile`}
            </Typography>
          </Box>
          <TipIconButton
            label={t`Open folder`}
            disabled={pending}
            onClick={() => run(() => openFolder(item.id))}
          >
            <FolderOpen size={16} />
          </TipIconButton>
          <TipIconButton
            label={item.repairError || t`Repair`}
            disabled={pending || Boolean(item.repairError)}
            onClick={() => run(() => repair(item.id))}
          >
            <Wrench size={16} />
          </TipIconButton>
          <TipIconButton
            label={t`Delete`}
            color="error"
            disabled={pending}
            onClick={() => run(() => remove(item.id))}
          >
            <Trash2 size={16} />
          </TipIconButton>
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
        bgcolor: 'var(--mortar-panel)',
        borderRadius: '8px',
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <Typography
          component="h2"
          sx={{ fontSize: 16, fontWeight: 700 }}
        >{t`Recently deleted`}</Typography>
        {trash.length === 0 ? null : (
          <TipIconButton
            label={t`Empty trash`}
            color="error"
            disabled={pending}
            onClick={() => setConfirming(true)}
          >
            <Trash2 size={16} />
          </TipIconButton>
        )}
      </Box>
      <Typography sx={{ fontSize: 13, lineHeight: 1.45 }}>
        {t`Deleted profiles stay here, mods and settings included.`}
      </Typography>
      {trash.map((item) => (
        <TrashRow key={item.id} item={item} />
      ))}
      <ConfirmDialog
        open={confirming}
        title={t`Empty trash?`}
        body={t`This cannot be undone.`}
        confirmLabel={t`Empty trash`}
        color="error"
        onCancel={() => setConfirming(false)}
        onConfirm={() => {
          setConfirming(false)
          run(purgeTrash)
        }}
      />
    </Box>
  )
}

function FindModSearch({ profiles }: { profiles: Profile[] }) {
  const { t } = useLingui()
  const [query, setQuery] = useState('')
  const inputRef = useRef<HTMLInputElement>(null)
  const hits = findModInProfiles(profiles, query)
  useEffect(() => onFindAllFocus(() => inputRef.current?.focus()), [])
  return (
    <Box sx={{ px: 2.5, pt: 1.5, flexShrink: 0 }}>
      <SearchField
        fullWidth={true}
        value={query}
        onChange={setQuery}
        label={t`Find a mod in all profiles`}
        inputRef={inputRef}
      />
      {hits.length === 0 && query.trim() !== '' ? (
        <EmptyState icon={<SearchX />} title={t`No mod matches`} compact={true}>
          {t`No profile holds a mod by that name or id.`}
        </EmptyState>
      ) : null}
      {hits.length > 0 ? (
        <Box sx={{ mt: 1, display: 'flex', flexDirection: 'column', gap: 0.25 }}>
          {hits.map((h) => (
            <Button
              key={`${h.profileId}/${h.key}/${h.id}`}
              onClick={() => openModInProfile(h)}
              title={`${h.name} · ${localId(h.id)} · ${h.profileName} · ${h.version} · ${h.enabled ? t`Enabled` : t`Disabled`}`}
              sx={{
                minWidth: 0,
                overflow: 'hidden',
                textOverflow: 'ellipsis',
                justifyContent: 'flex-start',
                fontSize: 13,
              }}
            >
              {`${h.name} · ${h.profileName} · ${h.version} · ${h.enabled ? t`Enabled` : t`Disabled`}`}
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
  const [importAnchor, setImportAnchor] = useState<HTMLElement | null>(null)
  const [wizard, setWizard] = useState(false)
  const [packImport, setPackImport] = useState(false)
  const [packPath, setPackPath] = useState('')
  const thunderstore = hasThunderstore(useProfiles((s) => s.game))
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
        sx={{ height: 40, px: 2, fontSize: 14 }}
      >
        {t`Import`}
      </Button>
      <Menu
        id={importMenuId}
        anchorEl={importAnchor}
        open={importAnchor !== null}
        onClose={closeImportMenu}
      >
        <MenuAction
          icon={<FolderInput size={16} aria-hidden={true} />}
          label={t`From the game's Mods folder…`}
          onClick={() => {
            closeImportMenu()
            onImportGame()
          }}
        />
        <MenuAction
          icon={<Download size={16} aria-hidden={true} />}
          label={t`From a link or file…`}
          onClick={() => {
            closeImportMenu()
            onImport()
          }}
        />
        <MenuAction
          icon={<FileUp size={16} aria-hidden={true} />}
          label={t`From a backup…`}
          onClick={() => {
            closeImportMenu()
            onRestoreZip()
          }}
        />
        <MenuAction
          icon={<Download size={16} aria-hidden={true} />}
          label={t`From another mod manager…`}
          onClick={() => {
            closeImportMenu()
            setWizard(true)
          }}
        />
      </Menu>
      <ImportWizard
        open={wizard}
        game={game}
        onClose={() => setWizard(false)}
        onPickPack={(path) => {
          setPackPath(path)
          setPackImport(true)
        }}
        onOwnCode={
          thunderstore
            ? () => {
                setPackPath('')
                setPackImport(true)
              }
            : null
        }
      />
      <PackImportDialog
        open={packImport}
        game={game}
        initialPath={packPath}
        onClose={() => setPackImport(false)}
      />
      <Button
        variant="contained"
        startIcon={<Plus size={16} />}
        onClick={onCreate}
        sx={{ height: 40, px: 2, fontSize: 14 }}
      >
        {t`New profile…`}
      </Button>
    </Box>
  )
}

export function ProfilesPage() {
  const { t } = useLingui()
  const closeProfiles = useNav((s) => s.closeProfiles)
  const game = useCurrentGame()
  const profiles = useProfiles((s) => s.profiles)
  useProfilePageBadges(game)
  const budgets = useBudgets(
    game,
    profiles.map((p) => `${p.id}:${p.entries?.length ?? 0}`).join(','),
  )
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
      const typing = e.target instanceof HTMLElement ? e.target : null
      if (isTypingTarget(typing)) {
        return
      }
      if (shouldLeavePageOnEscape(e, dialogOpen())) {
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
              title={t`No profiles yet`}
              action={
                <Button
                  variant="contained"
                  startIcon={<Plus size={16} />}
                  onClick={() => setCreating(true)}
                >{t`New profile…`}</Button>
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
                  <ProfileRow key={p.id} profile={p} budget={budgets.get(p.id)} />
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
