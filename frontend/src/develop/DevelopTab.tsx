import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonGroup, Menu, Typography } from '@mui/material'
import { useStoredState } from '@rethunk/react-load-hooks'
import {
  Check,
  ChevronDown,
  Code,
  FolderPlus,
  GitBranch,
  Hammer,
  Plus,
  SearchX,
  Users,
} from 'lucide-react'
import { memo, useCallback, useEffect, useMemo, useState } from 'react'
import {
  Add,
  Detect,
  Forget,
  Move,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { StartInstances } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { installExtraFolder } from '../downloads/autoInstall.ts'
import { PageActions } from '../game/PageActions.tsx'
import { useProfiles } from '../profiles/store.ts'
import { persist } from '../settings/persist.ts'
import { useSettings } from '../settings/store.ts'
import { CardsView } from '../shell/CardsView.tsx'
import { ControlsRow } from '../shell/ControlsRow.tsx'
import { EmptyState } from '../shell/EmptyState.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { ViewToggle } from '../shell/ViewToggle.tsx'
import { isViewMode, type ViewMode } from '../shell/viewMode.ts'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { AdoptHost } from './AdoptDialog.tsx'
import { useAdopting } from './adopting.ts'
import { refreshInfo, useProjectInfo } from './builds.ts'
import { CloneDialog } from './CloneDialog.tsx'
import { ElsewhereHost } from './ElsewhereDialog.tsx'
import { useElsewhere } from './elsewhere.ts'
import { GameCodeDialog } from './GameCodeDialog.tsx'
import { useDevKinds } from './kinds.ts'
import { NewModDialog } from './NewModDialog.tsx'
import { ProjectCard } from './ProjectCard.tsx'
import { TrustDialog } from './TrustDialog.tsx'
import { installTestMods } from './testMods.ts'

const profileNameOf = (list: { id: string; name: string }[], id: string) =>
  list.find((p) => p.id === id)?.name ?? ''

// A project in another profile: its builds install there, so it is worked on from that profile. One whose profile
// was deleted can be moved into the open profile or dropped from the list.
function ElsewhereRow({ game, project }: { game: string; project: DevProject }) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const profileName = useProfiles(
    (s) => profileNameOf(s.profiles, project.profile) || profileNameOf(s.damaged, project.profile),
  )
  const gone = useProfiles((s) => s.profiles.length > 0 && profileName === '')
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
      <Typography sx={{ flex: 1, minWidth: 0 }} noWrap={true}>
        {gone ? project.name : t`${{ name: project.name }} in ${{ profile: profileName }}`}
      </Typography>
      {gone ? (
        <>
          <Button
            onClick={() =>
              persist(() => Forget(game, project.dir), t`Could not remove the project`)
            }
          >{t`Remove from list`}</Button>
          <Button
            disabled={openId === ''}
            onClick={() =>
              persist(async () => {
                await Move(game, project.dir, openId)
                installExtraFolder(game, project.output)
              }, t`Could not move the project`)
            }
          >{t`Move here`}</Button>
        </>
      ) : (
        <Button
          onClick={() => useProfiles.getState().open(project.profile)}
        >{t`Open profile`}</Button>
      )}
    </Box>
  )
}

// Matches profile.MaxTestCopies: a host and a few clients on one computer.
const MAX_TEST_COPIES = 4

// Starts copies of the profile, one hosting and the rest joining, to test everything in development here. Mods only
// the player who installs them needs test in one copy, others in two, until a count picked from the menu is
// remembered for the profile.
function TestCopies({ game, profile, dirs }: { game: string; profile: string; dirs: string[] }) {
  const { t } = useLingui()
  const [menu, setMenu] = useState<HTMLElement | null>(null)
  const saved = useProfiles((s) => s.profiles.find((p) => p.id === profile)?.testCopies ?? 0)
  const clientOnly = useProjectInfo((s) => dirs.every((d) => s.byDir[d]?.clientOnly === true))
  const count = saved || (clientOnly ? 1 : 2)
  const start = (n: number) =>
    installTestMods(game, profile)
      .then(() => StartInstances(game, profile, n))
      .catch(reportError(t`Could not start the game`))
  return (
    <>
      <ButtonGroup variant="outlined">
        <Button
          startIcon={<Users size={14} />}
          title={plural(count, { one: 'Starts # copy', other: 'Starts # copies, one hosting' })}
          onClick={() => start(count)}
        >
          {t`Test`}
        </Button>
        <Button
          aria-label={t`How many copies to test with`}
          aria-haspopup="menu"
          aria-expanded={menu !== null}
          onClick={(e) => setMenu(e.currentTarget)}
          sx={{ minWidth: 30, px: 0.5 }}
        >
          <ChevronDown size={14} aria-hidden={true} />
        </Button>
      </ButtonGroup>
      <Menu open={menu !== null} anchorEl={menu} onClose={() => setMenu(null)}>
        {Array.from({ length: MAX_TEST_COPIES }, (_, i) => i + 1).map((n) => (
          <MenuAction
            key={n}
            icon={
              n === count ? (
                <Check size={16} aria-hidden={true} />
              ) : (
                <Users size={16} aria-hidden={true} />
              )
            }
            label={plural(n, { one: 'Test with # copy', other: 'Test with # copies' })}
            onClick={() => {
              setMenu(null)
              useProfiles
                .getState()
                .setTestCopies(profile, n)
                .catch(reportError(t`Could not save the number of copies`))
              start(n)
            }}
          />
        ))}
      </Menu>
    </>
  )
}

const NO_PROJECTS: DevProject[] = []

function ProjectGridView({
  game,
  projects,
  filtering,
  list,
  onClear,
}: {
  game: string
  projects: DevProject[]
  filtering: boolean
  list: boolean
  onClear: () => void
}) {
  const { t } = useLingui()
  if (projects.length === 0 && filtering) {
    return (
      <EmptyState
        icon={<SearchX />}
        title={t`No projects match`}
        action={<Button onClick={onClear}>{t`Show all projects`}</Button>}
      >
        {t`Try a different search.`}
      </EmptyState>
    )
  }
  return (
    <CardsView list={list}>
      {projects.map((p) => (
        <ProjectCard key={p.dir} game={game} project={p} list={list} />
      ))}
    </CardsView>
  )
}

// Memoised: opening a dialog or the add menu changes the tab's state, which would otherwise redraw every card.
const ProjectGrid = memo(ProjectGridView)

// Picking is modal and the dialog that follows holds its own state, so the tab keeps none for the pick: one more
// state here would redraw every project card twice per add.
function useAddFolder(game: string, openId: string) {
  const { t } = useLingui()
  return () => {
    ;(async () => {
      const dir = await PickFolder(t`Choose the mod project's folder`)
      if (!dir) {
        return
      }
      const found = await Detect(game, dir)
      if (!found.known) {
        useAdopting.getState().ask({ game, dir, found })
        return
      }
      if (found.game && found.game !== game) {
        useElsewhere.getState().ask({ game, dir, found })
        return
      }
      const project = await Add(game, dir, openId)
      useToasts
        .getState()
        .push({ kind: 'success', title: t`Added ${{ name: project.name }}`, body: project.dir })
    })().catch(reportError(t`Could not add the project`))
  }
}

/** The open profile's own mods in development: start one from the template, then build, test, pack and release it. */
export function DevelopTab({ game }: { game: string }) {
  const { t } = useLingui()
  const projects = useSettings((s) => s.games?.[game]?.devProjects) ?? NO_PROJECTS
  const openId = useProfiles((s) => s.openId)
  const [creating, setCreating] = useState(false)
  const [cloning, setCloning] = useState(false)
  const [reading, setReading] = useState(false)
  const [addMenu, setAddMenu] = useState<HTMLElement | null>(null)
  const addFolder = useAddFolder(game, openId)
  const [query, setQuery] = useState('')
  const [view, setView] = useStoredState<ViewMode>(`mortar.developView.${game}`, 'list', isViewMode)
  const here = useMemo(() => projects.filter((p) => p.profile === openId), [projects, openId])
  const kinds = useDevKinds((s) => s.byDir)
  const codeHere = here.find((p) => !kinds[p.dir])
  const needle = query.trim().toLowerCase()
  const shown = useMemo(
    () => here.filter((p) => p.name.toLowerCase().includes(needle)),
    [here, needle],
  )
  const clearQuery = useCallback(() => setQuery(''), [])
  const elsewhere = projects.filter((p) => p.profile !== openId)
  // Files can change in an editor while Mortar is in the background; coming back rereads them.
  const dirs = here.map((p) => p.dir).join('\n')
  useEffect(() => {
    const reread = () => {
      for (const dir of dirs.split('\n').filter(Boolean)) {
        refreshInfo(game, dir, true).catch(() => undefined)
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
        {codeHere ? (
          <Button
            variant="outlined"
            startIcon={<Code size={14} />}
            onClick={() => setReading(true)}
          >
            {t`Game code…`}
          </Button>
        ) : null}
        {here.length > 0 ? (
          <TestCopies game={game} profile={openId} dirs={here.map((p) => p.dir)} />
        ) : null}
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
      <ProjectGrid
        game={game}
        projects={shown}
        filtering={needle !== ''}
        list={view === 'list'}
        onClear={clearQuery}
      />
      {elsewhere.length > 0 ? (
        <Box sx={{ px: space.gutter, pb: space.gutter }}>
          <Typography sx={{ fontWeight: 600, mb: 0.5 }}>{t`In other profiles`}</Typography>
          {elsewhere.map((p) => (
            <ElsewhereRow key={p.dir} game={game} project={p} />
          ))}
        </Box>
      ) : null}
      <CloneDialog open={cloning} game={game} onClose={() => setCloning(false)} />
      <NewModDialog open={creating} game={game} onClose={() => setCreating(false)} />
      <TrustDialog />
      <AdoptHost />
      <ElsewhereHost />
      {reading && codeHere ? (
        <GameCodeDialog game={game} dir={codeHere.dir} onClose={() => setReading(false)} />
      ) : null}
    </Box>
  )
}
