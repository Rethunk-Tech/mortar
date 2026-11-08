import {
  closestCenter,
  DndContext,
  type DragEndEvent,
  KeyboardSensor,
  PointerSensor,
  type UniqueIdentifier,
  useSensor,
  useSensors,
} from '@dnd-kit/core'
import {
  rectSortingStrategy,
  SortableContext,
  sortableKeyboardCoordinates,
  verticalListSortingStrategy,
} from '@dnd-kit/sortable'
import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, ButtonGroup, Menu, Typography } from '@mui/material'
import { useStoredState } from '@rethunk/react-load-hooks'
import {
  ArrowDownAZ,
  ArrowUpDown,
  ChevronDown,
  Code,
  FolderPlus,
  GitBranch,
  GripVertical,
  Hammer,
  Plus,
  SearchX,
  Users,
} from 'lucide-react'
import { memo, useCallback, useMemo, useState } from 'react'
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
import { profileById, profileByIdIncludingDamaged, useProfiles } from '../profiles/store.ts'
import { persist } from '../settings/persist.ts'
import { useSettings } from '../settings/store.ts'
import { CardsView } from '../shell/CardsView.tsx'
import { ControlsRow } from '../shell/ControlsRow.tsx'
import { useDragAnnouncements } from '../shell/dragAnnouncements.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { MenuAction, MenuCheck } from '../shell/MenuAction.tsx'
import { MenuButton } from '../shell/MenuButton.tsx'
import { SearchField } from '../shell/SearchField.tsx'
import { useOnWindowFocus } from '../shell/useOnWindowFocus.ts'
import { usePopoverAnchor } from '../shell/usePopoverAnchor.ts'
import { ViewToggle } from '../shell/ViewToggle.tsx'
import { isViewMode, type ViewMode } from '../shell/viewMode.ts'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { AdoptHost } from './AdoptDialog.tsx'
import { useAdopting } from './adopting.ts'
import { buildAll, refreshInfo, runOf, useDevRuns, useProjectInfo } from './builds.ts'
import { CloneDialog } from './CloneDialog.tsx'
import { ElsewhereHost } from './ElsewhereDialog.tsx'
import { useElsewhere } from './elsewhere.ts'
import { GameCodeDialog } from './GameCodeDialog.tsx'
import { useGameCode } from './kinds.ts'
import { NewModDialog } from './NewModDialog.tsx'
import { ProjectCard } from './ProjectCard.tsx'
import { isProjectSort, movedDirs, type ProjectSort, sortProjects } from './projectOrder.ts'
import { SortableCard } from './SortableCard.tsx'
import { saveOrder } from './saveOrder.ts'
import { TrustDialog } from './TrustDialog.tsx'
import { installTestMods } from './testMods.ts'

// Builds the profile's trusted projects, all at once save for waiting on their dependencies: a content pack's build
// puts its folder in the profile, and an untrusted project asks first, from its own card. Shown only when there is more
// than one to build.
function BuildAll({ game, projects }: { game: string; projects: DevProject[] }) {
  const { t } = useLingui()
  const buildable = projects.filter((p) => !p.untrusted).map((p) => p.dir)
  const building = useDevRuns((s) => projects.some((p) => runOf(s.byDir, p.dir).running !== null))
  return buildable.length > 1 ? (
    <Button
      variant="outlined"
      startIcon={<Hammer size={14} />}
      disabled={building}
      onClick={() => buildAll(game, buildable)}
    >
      {t`Build all`}
    </Button>
  ) : null
}

// A project in another profile: its builds install there, so it is worked on from that profile. One whose profile
// was deleted can be moved into the open profile or dropped from the list.
function ElsewhereRow({ game, project }: { game: string; project: DevProject }) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const profileName = useProfiles(profileByIdIncludingDamaged(project.profile))?.name ?? ''
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
  const { anchor: menu, show: showMenu, hide: hideMenu } = usePopoverAnchor()
  const saved = useProfiles(profileById(profile))?.testCopies ?? 0
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
          onClick={showMenu}
          sx={{ minWidth: 30, px: 0.5 }}
        >
          <ChevronDown size={14} aria-hidden={true} />
        </Button>
      </ButtonGroup>
      <Menu open={menu !== null} anchorEl={menu} onClose={() => hideMenu()}>
        {Array.from({ length: MAX_TEST_COPIES }, (_, i) => i + 1).map((n) => (
          <MenuCheck
            key={n}
            checked={n === count}
            icon={<Users size={16} aria-hidden={true} />}
            label={plural(n, { one: 'Test with # copy', other: 'Test with # copies' })}
            onClick={() => {
              hideMenu()
              useProfiles
                .getState()
                .setTestCopies(profile, n)
                .catch(reportError(t`Could not save the number of copies`))
              return start(n)
            }}
          />
        ))}
      </Menu>
    </>
  )
}

const NO_PROJECTS: DevProject[] = []

function SortControl({
  value,
  onChange,
}: {
  value: ProjectSort
  onChange: (v: ProjectSort) => void
}) {
  const { t } = useLingui()
  const menu = usePopoverAnchor()
  const options = [
    { id: 'manual' as const, label: t`Manual`, Icon: GripVertical },
    { id: 'name' as const, label: t`Name`, Icon: ArrowDownAZ },
  ]
  return (
    <MenuButton
      menu={menu}
      icon={<ArrowUpDown size={14} />}
      label={t`Sort`}
      ariaLabel={t`Sort projects`}
    >
      {options.map((o) => (
        <MenuCheck
          key={o.id}
          radio={true}
          checked={value === o.id}
          icon={<o.Icon size={16} aria-hidden={true} />}
          label={o.label}
          onClick={() => {
            onChange(o.id)
            menu.hide()
          }}
        />
      ))}
    </MenuButton>
  )
}

function ProjectGridView({
  game,
  projects,
  filtering,
  list,
  draggable,
  onClear,
}: {
  game: string
  projects: DevProject[]
  filtering: boolean
  list: boolean
  draggable: boolean
  onClear: () => void
}) {
  const { t } = useLingui()
  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  )
  const nameOf = useCallback(
    (id: UniqueIdentifier) => projects.find((p) => p.dir === id)?.name ?? String(id),
    [projects],
  )
  const announcements = useDragAnnouncements(nameOf)
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
  if (!draggable) {
    return (
      <CardsView list={list}>
        {projects.map((p) => (
          <ProjectCard key={p.dir} game={game} project={p} list={list} />
        ))}
      </CardsView>
    )
  }
  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (over) {
      saveOrder(
        game,
        movedDirs(
          projects.map((p) => p.dir),
          String(active.id),
          String(over.id),
        ),
      )
    }
  }
  return (
    <DndContext
      sensors={sensors}
      collisionDetection={closestCenter}
      accessibility={{ announcements }}
      onDragEnd={onDragEnd}
    >
      <SortableContext
        items={projects.map((p) => p.dir)}
        strategy={list ? verticalListSortingStrategy : rectSortingStrategy}
      >
        <CardsView list={list}>
          {projects.map((p) => (
            <SortableCard key={p.dir} game={game} project={p} list={list} />
          ))}
        </CardsView>
      </SortableContext>
    </DndContext>
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

// Files can change in an editor while Mortar is in the background; coming back rereads them.
function useRereadOnFocus(game: string, projects: DevProject[]) {
  const dirs = projects.map((p) => p.dir).join('\n')
  const reread = useCallback(() => {
    for (const dir of dirs.split('\n').filter(Boolean)) {
      refreshInfo(game, dir, true).catch(() => undefined)
    }
  }, [game, dirs])
  useOnWindowFocus(reread)
}

function ProjectControls({
  count,
  view,
  onView,
  sort,
  onSort,
  query,
  onQuery,
}: {
  count: number
  view: ViewMode
  onView: (v: ViewMode) => void
  sort: ProjectSort
  onSort: (v: ProjectSort) => void
  query: string
  onQuery: (v: string) => void
}) {
  const { t } = useLingui()
  return (
    <ControlsRow>
      <ViewToggle value={view} onChange={onView} />
      <SortControl value={sort} onChange={onSort} />
      <SearchField
        value={query}
        onChange={onQuery}
        label={t`Filter projects`}
        placeholder={plural({ count }, { one: 'Filter # project', other: 'Filter # projects' })}
        grow={true}
      />
    </ControlsRow>
  )
}

// The view switch and the projects it lays out: its own component, so changing the view redraws the toolbar and the
// cards and leaves the tab's actions and dialogs alone.
function ProjectsView({
  game,
  count,
  sort,
  onSort,
  query,
  onQuery,
  projects,
  filtering,
  draggable,
  onClear,
}: {
  game: string
  count: number
  sort: ProjectSort
  onSort: (v: ProjectSort) => void
  query: string
  onQuery: (v: string) => void
  projects: DevProject[]
  filtering: boolean
  draggable: boolean
  onClear: () => void
}) {
  const [view, setView] = useStoredState<ViewMode>(`mortar.developView.${game}`, 'list', isViewMode)
  return (
    <>
      <ProjectControls
        count={count}
        view={view}
        onView={setView}
        sort={sort}
        onSort={onSort}
        query={query}
        onQuery={onQuery}
      />
      <ProjectGrid
        game={game}
        projects={projects}
        filtering={filtering}
        list={view === 'list'}
        draggable={draggable}
        onClear={onClear}
      />
    </>
  )
}

/** The open profile's own mods in development: start one from the template, then build, test, pack and release it. */
export function DevelopTab({ game }: { game: string }) {
  const { t } = useLingui()
  const projects = useSettings((s) => s.games?.[game]?.devProjects) ?? NO_PROJECTS
  const openId = useProfiles((s) => s.openId)
  const [creating, setCreating] = useState(false)
  const [cloning, setCloning] = useState(false)
  const [reading, setReading] = useState(false)
  const { anchor: addMenu, show: showAddMenu, hide: hideAddMenu } = usePopoverAnchor()
  const addFolder = useAddFolder(game, openId)
  const [query, setQuery] = useState('')
  const [sort, setSort] = useStoredState<ProjectSort>(
    `mortar.developSort.${game}`,
    'manual',
    isProjectSort,
  )
  const here = useMemo(() => projects.filter((p) => p.profile === openId), [projects, openId])
  const code = useGameCode(game)
  const needle = query.trim().toLowerCase()
  const shown = useMemo(
    () =>
      sortProjects(
        here.filter((p) => p.name.toLowerCase().includes(needle)),
        sort,
      ),
    [here, needle, sort],
  )
  const clearQuery = useCallback(() => setQuery(''), [])
  const elsewhere = projects.filter((p) => p.profile !== openId)
  useRereadOnFocus(game, here)
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
        {code.available ? (
          <Button
            variant="outlined"
            startIcon={<Code size={14} />}
            onClick={() => setReading(true)}
          >
            {t`Game code…`}
          </Button>
        ) : null}
        <BuildAll game={game} projects={here} />
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
            onClick={showAddMenu}
            sx={{ minWidth: 30, px: 0.5 }}
          >
            <ChevronDown size={14} aria-hidden={true} />
          </Button>
        </ButtonGroup>
        <Menu open={addMenu !== null} anchorEl={addMenu} onClose={() => hideAddMenu()}>
          <MenuAction
            icon={<FolderPlus size={16} aria-hidden={true} />}
            label={t`Add a project folder…`}
            onClick={() => {
              hideAddMenu()
              addFolder()
            }}
          />
          <MenuAction
            icon={<GitBranch size={16} aria-hidden={true} />}
            label={t`Clone from ${{ source: sourceLabel('github') }}…`}
            onClick={() => {
              hideAddMenu()
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
        <ProjectsView
          game={game}
          count={here.length}
          sort={sort}
          onSort={setSort}
          query={query}
          onQuery={setQuery}
          projects={shown}
          filtering={needle !== ''}
          draggable={sort === 'manual' && needle === '' && here.length > 1}
          onClear={clearQuery}
        />
      )}
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
      {reading && code.available ? (
        <GameCodeDialog game={game} dir={code.dir} onClose={() => setReading(false)} />
      ) : null}
    </Box>
  )
}
