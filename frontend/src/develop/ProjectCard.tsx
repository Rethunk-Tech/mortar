import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Alert,
  Box,
  Button,
  Chip,
  CircularProgress,
  FormControlLabel,
  IconButton,
  Link,
  Menu,
  MenuItem,
  Switch,
  Typography,
} from '@mui/material'
import { EllipsisVertical, GitBranch, Hammer, Users } from 'lucide-react'
import { useEffect, useState } from 'react'
import type {
  GitState,
  Problem,
  ProjectInfo,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Forget,
  OpenAt,
  OpenEditor,
  OpenProject,
  SetWatch,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import {
  StartDebugging,
  StartInstances,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { OpenWeb } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/opener/service.ts'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { useNowValue } from '../i18n/useNow.ts'
import { relativeWhen } from '../i18n/when.ts'
import { LetterTile } from '../mods/LetterTile.tsx'
import { persist } from '../settings/persist.ts'
import { ConfirmationDialog } from '../shell/dialog/ConfirmationDialog.tsx'
import { MenuRule } from '../shell/TitleMenu.tsx'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import {
  buildProject,
  packProject,
  refreshInfo,
  runOf,
  useDevRuns,
  useProjectInfo,
} from './builds.ts'
import { EditProjectDialog } from './EditProjectDialog.tsx'
import { GameCodeDialog } from './GameCodeDialog.tsx'
import { showModLog, useModCosts } from './modCosts.ts'
import { ReleaseDialog } from './ReleaseDialog.tsx'

const THUNDERSTORE_UPLOAD = 'https://thunderstore.io/package/create/'
const TEST_COPIES = 2
// The status chip's "built X ago" moves on once a minute.
const MINUTE_MS = 60_000

// A problem's file below the project, as the editor would show it.
const relative = (dir: string, file: string) =>
  file.startsWith(`${dir}/`) || file.startsWith(`${dir}\\`) ? file.slice(dir.length + 1) : file

function Problems({
  game,
  project,
  problems,
}: {
  game: string
  project: DevProject
  problems: Problem[]
}) {
  const { t } = useLingui()
  const failed = t`Could not open the file`
  return (
    <Box component="ul" sx={{ m: 0, pl: 2.5, fontFamily: 'monospace', fontSize: 'small.fontSize' }}>
      {problems.map((p) => (
        <li key={`${p.file}:${p.line}:${p.column}:${p.code}`}>
          <Link
            component="button"
            color={p.severity === 'error' ? 'error' : 'warning'}
            onClick={() =>
              OpenAt(game, project.dir, p.file, p.line, p.column).catch(reportError(failed))
            }
            sx={{ textAlign: 'left' }}
          >
            {`${relative(project.dir, p.file)}(${p.line},${p.column}) ${p.code}: ${p.message}`}
          </Link>
        </li>
      ))}
    </Box>
  )
}

function Outcome({ game, project }: { game: string; project: DevProject }) {
  const { t } = useLingui()
  const run = useDevRuns((s) => runOf(s.byDir, project.dir))
  const build = run.build
  const pack = run.pack
  return (
    <>
      {build && !build.ok ? (
        <Alert variant="compact" severity="error">
          {t`The build failed.`}
        </Alert>
      ) : null}
      {build && (build.problems ?? []).length > 0 ? (
        <Problems game={game} project={project} problems={build.problems ?? []} />
      ) : null}
      {pack?.zip && (pack.issues ?? []).length === 0 ? (
        <Alert
          variant="compact"
          severity="success"
          action={
            <Button
              size="small"
              onClick={() =>
                OpenWeb(THUNDERSTORE_UPLOAD).catch(reportError(t`Could not open the page`))
              }
            >
              {t`Upload…`}
            </Button>
          }
        >
          {t`Ready to upload: ${{ path: pack.zip }}`}
        </Alert>
      ) : null}
      {pack && (pack.issues ?? []).length > 0 ? (
        <Alert variant="compact" severity="warning">
          {(pack.issues ?? []).join(' ')}
        </Alert>
      ) : null}
    </>
  )
}

function Status({ dir }: { dir: string }) {
  const { t, i18n } = useLingui()
  const run = useDevRuns((s) => runOf(s.byDir, dir))
  const now = useNowValue((n) => Math.floor(n / MINUTE_MS))
  if (run.running) {
    return (
      <Chip
        size="small"
        icon={<CircularProgress size={12} />}
        label={run.running === 'build' ? t`Building` : t`Packing`}
      />
    )
  }
  if (!run.build) {
    return null
  }
  if (run.build.ok) {
    const when = relativeWhen(run.builtAt, { now: now * MINUTE_MS, locale: i18n.locale })
    return <Chip size="small" color="success" variant="outlined" label={t`Built ${{ when }}`} />
  }
  const count = (run.build.problems ?? []).filter((p) => p.severity === 'error').length
  return (
    <Chip size="small" color="error" label={plural(count, { one: '# error', other: '# errors' })} />
  )
}

// The repository's branch, and what is not committed or not pushed yet.
function GitChips({ git }: { git: GitState }) {
  return (
    <>
      <Chip size="small" variant="outlined" icon={<GitBranch size={12} />} label={git.branch} />
      {git.changed > 0 ? (
        <Chip
          size="small"
          color="warning"
          variant="outlined"
          label={plural(
            { count: git.changed },
            { one: '# changed file', other: '# changed files' },
          )}
        />
      ) : null}
      {git.unpushed > 0 ? (
        <Chip
          size="small"
          variant="outlined"
          label={plural(
            { count: git.unpushed },
            { one: '# unpushed commit', other: '# unpushed commits' },
          )}
        />
      ) : null}
    </>
  )
}

// The project's details from its own files, read when the card shows and kept current by refreshInfo.
function useInfo(game: string, dir: string) {
  const info = useProjectInfo((s) => s.byDir[dir])
  useEffect(() => {
    refreshInfo(game, dir).catch(() => undefined)
  }, [game, dir])
  return info
}

// The mod's own cost in the development profile's latest measured run.
function Costs({
  game,
  profile,
  info,
}: {
  game: string
  profile: string
  info: ProjectInfo | undefined
}) {
  const { t, i18n } = useLingui()
  const { startupMs, frameMs } = useModCosts(game, profile, info)
  const start = startupMs === undefined ? '' : Math.round(startupMs).toLocaleString(i18n.locale)
  const frame =
    frameMs === undefined ? '' : frameMs.toLocaleString(i18n.locale, { maximumFractionDigits: 2 })
  let line = ''
  if (start && frame) {
    line = t`Last run: ${{ start }} ms to start, ${{ frame }} ms a frame`
  } else if (start) {
    line = t`Last run: ${{ start }} ms to start`
  } else if (frame) {
    line = t`Last run: ${{ frame }} ms a frame`
  }
  return line ? (
    <Typography sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>{line}</Typography>
  ) : null
}

// A project whose folder or project file is gone: it can only leave the list.
function MissingCard({ game, project }: { game: string; project: DevProject }) {
  const { t } = useLingui()
  return (
    <Alert
      variant="compact"
      severity="warning"
      action={
        <Button
          size="small"
          onClick={() => persist(() => Forget(game, project.dir), t`Could not save that setting`)}
        >
          {t`Remove from list`}
        </Button>
      }
    >
      {t`${{ name: project.name }} is no longer in ${{ path: project.dir }}.`}
    </Alert>
  )
}

function MoreMenu({
  game,
  project,
  onRelease,
  onEdit,
  onCode,
}: {
  game: string
  project: DevProject
  onRelease: () => void
  onEdit: () => void
  onCode: () => void
}) {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const [forgetting, setForgetting] = useState(false)
  const running = useDevRuns((s) => runOf(s.byDir, project.dir).running)
  const info = useProjectInfo((s) => s.byDir[project.dir])
  // One start with the game's Mono debugger open; the toast says where to attach.
  const debug = () =>
    StartDebugging(game, project.profile)
      .then((address) =>
        useToasts.getState().push({
          kind: 'info',
          title: t`Starting for debugging`,
          body: t`Attach your editor's debugger to ${{ address }}.`,
        }),
      )
      .catch(reportError(t`Could not start the game`))
  const act = (fn: () => unknown) => () => {
    setAnchor(null)
    fn()
  }
  return (
    <>
      <IconButton aria-label={t`More actions`} onClick={(e) => setAnchor(e.currentTarget)}>
        <EllipsisVertical size={18} />
      </IconButton>
      <Menu anchorEl={anchor} open={anchor !== null} onClose={() => setAnchor(null)}>
        <MenuItem onClick={act(onEdit)}>{t`Edit details…`}</MenuItem>
        <MenuItem disabled={running !== null} onClick={act(() => packProject(game, project.dir))}>
          {t`Pack for ${{ source: sourceLabel('thunderstore') }}`}
        </MenuItem>
        <MenuItem onClick={act(onRelease)}>{t`Release…`}</MenuItem>
        <MenuItem onClick={act(onCode)}>{t`Game code…`}</MenuItem>
        <MenuItem disabled={!info} onClick={act(() => info && showModLog(info))}>
          {t`Show log`}
        </MenuItem>
        <MenuItem onClick={act(debug)}>{t`Start for debugging`}</MenuItem>
        <MenuItem
          onClick={act(() =>
            OpenEditor(game, project.dir).catch(reportError(t`Could not open the editor`)),
          )}
        >
          {t`Open in editor`}
        </MenuItem>
        <MenuItem
          onClick={act(() =>
            OpenProject(game, project.dir).catch(reportError(t`Could not open the folder`)),
          )}
        >
          {t`Open folder`}
        </MenuItem>
        <MenuRule />
        <MenuItem onClick={act(() => setForgetting(true))}>{t`Remove from list…`}</MenuItem>
      </Menu>
      <ConfirmationDialog
        open={forgetting}
        tone="warning"
        title={t`Remove ${{ name: project.name }} from the list?`}
        body={t`Its folder stays where it is; Mortar stops building it and installing it into the profile.`}
        confirmLabel={t`Remove from list`}
        onCancel={() => setForgetting(false)}
        onConfirm={() => {
          setForgetting(false)
          persist(() => Forget(game, project.dir), t`Could not save that setting`)
        }}
      />
    </>
  )
}

function ProjectActions({ game, project }: { game: string; project: DevProject }) {
  const { t } = useLingui()
  const running = useDevRuns((s) => runOf(s.byDir, project.dir).running)
  const test = () =>
    StartInstances(game, project.profile, TEST_COPIES).catch(
      reportError(t`Could not start the game`),
    )
  return (
    <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap, flexWrap: 'wrap' }}>
      <Button
        variant="outlined"
        startIcon={<Hammer size={16} />}
        disabled={running !== null}
        onClick={() => buildProject(game, project.dir).catch(() => undefined)}
      >
        {t`Build`}
      </Button>
      <Button variant="outlined" startIcon={<Users size={16} />} onClick={test}>
        {t`Test with ${{ count: TEST_COPIES }} copies`}
      </Button>
      <FormControlLabel
        control={
          <Switch
            size="small"
            checked={Boolean(project.watch)}
            onChange={(_, on) =>
              persist(() => SetWatch(game, project.dir, on), t`Could not save that setting`)
            }
          />
        }
        label={t`Build on save`}
      />
    </Box>
  )
}

// One project: its name and state, its actions beside them in list view or below them on a grid card.
export function ProjectCard({
  game,
  project,
  list,
}: {
  game: string
  project: DevProject
  list: boolean
}) {
  const { t } = useLingui()
  const info = useInfo(game, project.dir)
  const [releasing, setReleasing] = useState(false)
  const [editing, setEditing] = useState(false)
  const [reading, setReading] = useState(false)
  if (info?.missing) {
    return <MissingCard game={game} project={project} />
  }
  const name = info?.name ?? project.name
  const actions = <ProjectActions game={game} project={project} />
  return (
    <Box
      sx={{
        p: space.pad,
        bgcolor: 'var(--mortar-overlay-45)',
        borderRadius: '6px',
        display: 'flex',
        flexDirection: 'column',
        gap: space.gap,
      }}
    >
      <Box sx={{ display: 'flex', alignItems: 'center', gap: space.gap }}>
        <LetterTile
          mod={{ id: name, name, ...(info?.icon ? { picture: info.icon } : {}) }}
          size={40}
        />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
            <Typography sx={{ fontSize: 'large.fontSize', fontWeight: 600 }}>{name}</Typography>
            <Chip
              size="small"
              variant="outlined"
              label={info?.released ? `v${info.released}` : t`Not released`}
            />
            {info?.git ? <GitChips git={info.git} /> : null}
            <Status dir={project.dir} />
          </Box>
          <Typography
            noWrap={true}
            title={project.dir}
            sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}
          >
            {info ? [info.guid, info.author].filter(Boolean).join(' · ') : project.dir}
          </Typography>
        </Box>
        {list ? actions : null}
        <MoreMenu
          game={game}
          project={project}
          onRelease={() => setReleasing(true)}
          onEdit={() => setEditing(true)}
          onCode={() => setReading(true)}
        />
      </Box>
      {list ? null : <Box>{actions}</Box>}
      <Costs game={game} profile={project.profile} info={info} />
      <Outcome game={game} project={project} />
      {editing && info ? (
        <EditProjectDialog
          game={game}
          dir={project.dir}
          info={info}
          onClose={() => setEditing(false)}
        />
      ) : null}
      {reading ? (
        <GameCodeDialog game={game} dir={project.dir} onClose={() => setReading(false)} />
      ) : null}
      {releasing ? (
        <ReleaseDialog
          game={game}
          dir={project.dir}
          name={name}
          onClose={() => setReleasing(false)}
        />
      ) : null}
    </Box>
  )
}
