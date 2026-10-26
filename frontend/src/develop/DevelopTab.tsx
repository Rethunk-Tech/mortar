import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonGroup, Menu, Typography } from '@mui/material'
import { ChevronDown, FolderPlus, GitBranch, Hammer, Plus, Users } from 'lucide-react'
import { useEffect, useState } from 'react'
import { Add } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { StartInstances } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { PageActions } from '../game/PageActions.tsx'
import { useProfiles } from '../profiles/store.ts'
import { useSettings } from '../settings/store.ts'
import { CardsView } from '../shell/CardsView.tsx'
import { ControlsRow } from '../shell/ControlsRow.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { useStoredState } from '../shell/useStoredState.ts'
import { ViewToggle } from '../shell/ViewToggle.tsx'
import { isViewMode, type ViewMode } from '../shell/viewMode.ts'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { refreshInfo } from './builds.ts'
import { CloneDialog } from './CloneDialog.tsx'
import { NewModDialog } from './NewModDialog.tsx'
import { ProjectCard } from './ProjectCard.tsx'

// A project in another profile: its builds install there, so it is worked on from that profile.
function ElsewhereRow({ project }: { project: DevProject }) {
  const { t } = useLingui()
  const profileName = useProfiles(
    (s) => s.profiles.find((p) => p.id === project.profile)?.name ?? '',
  )
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
      <Typography sx={{ flex: 1, minWidth: 0 }} noWrap={true}>
        {t`${{ name: project.name }} in ${{ profile: profileName }}`}
      </Typography>
      <Button
        onClick={() => useProfiles.getState().open(project.profile)}
      >{t`Open profile`}</Button>
    </Box>
  )
}

const TEST_COPIES = 2

// Starts the profile twice, host and guest, for a multiplayer test of everything in development here.
function TestCopies({ game, profile }: { game: string; profile: string }) {
  const { t } = useLingui()
  return (
    <Button
      variant="outlined"
      startIcon={<Users size={14} />}
      onClick={() =>
        StartInstances(game, profile, TEST_COPIES).catch(reportError(t`Could not start the game`))
      }
    >
      {t`Test with ${{ count: TEST_COPIES }} copies`}
    </Button>
  )
}

/** The open profile's own mods in development: start one from the template, then build, test, pack and release it. */
export function DevelopTab({ game }: { game: string }) {
  const { t } = useLingui()
  const projects = useSettings((s) => s.games?.[game]?.devProjects) ?? []
  const openId = useProfiles((s) => s.openId)
  const [creating, setCreating] = useState(false)
  const [cloning, setCloning] = useState(false)
  const [addMenu, setAddMenu] = useState<HTMLElement | null>(null)
  const [adding, run] = usePending()
  const addFolder = () =>
    run(
      async () => {
        const dir = await PickFolder(t`Choose the mod project's folder`)
        if (!dir) {
          return
        }
        const project = await Add(game, dir, openId)
        useToasts
          .getState()
          .push({ kind: 'success', title: t`Added ${{ name: project.name }}`, body: project.dir })
      },
      { errorTitle: t`Could not add the project` },
    )
  const [query, setQuery] = useState('')
  const [view, setView] = useStoredState<ViewMode>(`mortar.developView.${game}`, 'list', isViewMode)
  const here = projects.filter((p) => p.profile === openId)
  const needle = query.trim().toLowerCase()
  const shown = here.filter((p) => p.name.toLowerCase().includes(needle))
  const elsewhere = projects.filter((p) => p.profile !== openId)
  // Files can change in an editor while Mortar is in the background; coming back rereads them.
  const dirs = here.map((p) => p.dir).join('\n')
  useEffect(() => {
    const reread = () => {
      for (const dir of dirs.split('\n').filter(Boolean)) {
        refreshInfo(game, dir).catch(() => undefined)
      }
    }
    globalThis.addEventListener('focus', reread)
    return () => globalThis.removeEventListener('focus', reread)
  }, [game, dirs])
  return (
    <Box
      sx={{
        flex: 1,
        minHeight: 0,
        overflowY: 'auto',
        display: 'flex',
        flexDirection: 'column',
      }}
    >
      <PageActions>
        {here.length > 0 ? <TestCopies game={game} profile={openId} /> : null}
        <ButtonGroup variant="outlined">
          <Button startIcon={<Plus size={14} />} onClick={() => setCreating(true)}>
            {t`New mod…`}
          </Button>
          <Button
            aria-label={t`More ways to add a mod project`}
            aria-haspopup="menu"
            aria-expanded={addMenu !== null}
            onClick={(e) => setAddMenu(e.currentTarget)}
            sx={{ minWidth: 30, px: 0.5 }}
          >
            <ChevronDown size={14} aria-hidden={true} />
          </Button>
        </ButtonGroup>
        <Menu open={addMenu !== null} anchorEl={addMenu} onClose={() => setAddMenu(null)}>
          <MenuAction
            icon={<FolderPlus size={16} aria-hidden={true} />}
            label={t`Add a project folder…`}
            disabled={adding}
            onClick={() => {
              setAddMenu(null)
              addFolder()
            }}
          />
          <MenuAction
            icon={<GitBranch size={16} aria-hidden={true} />}
            label={t`Clone from ${{ source: sourceLabel('github') }}…`}
            onClick={() => {
              setAddMenu(null)
              setCloning(true)
            }}
          />
        </Menu>
      </PageActions>
      {here.length === 0 ? (
        <EmptyState
          icon={<Hammer />}
          title={t`No mods in development`}
          action={
            <Button
              variant="outlined"
              size="large"
              startIcon={<Plus size={14} />}
              onClick={() => setCreating(true)}
            >
              {t`New mod…`}
            </Button>
          }
        >
          {t`Start a mod from the template: Mortar sets up the project and installs each build into this profile.`}
        </EmptyState>
      ) : (
        <ControlsRow>
          <ViewToggle value={view} onChange={setView} />
          <SearchField
            value={query}
            onChange={setQuery}
            label={t`Filter projects`}
            placeholder={plural(
              { count: here.length },
              { one: 'Filter # project', other: 'Filter # projects' },
            )}
            grow={true}
          />
        </ControlsRow>
      )}
      <CardsView list={view === 'list'}>
        {shown.map((p) => (
          <ProjectCard key={p.dir} game={game} project={p} list={view === 'list'} />
        ))}
      </CardsView>
      {elsewhere.length > 0 ? (
        <Box sx={{ px: space.gutter, pb: space.gutter }}>
          <Typography sx={{ fontWeight: 600, mb: 0.5 }}>{t`In other profiles`}</Typography>
          {elsewhere.map((p) => (
            <ElsewhereRow key={p.dir} project={p} />
          ))}
        </Box>
      ) : null}
      <CloneDialog open={cloning} game={game} onClose={() => setCloning(false)} />
      <NewModDialog open={creating} game={game} onClose={() => setCreating(false)} />
    </Box>
  )
}
