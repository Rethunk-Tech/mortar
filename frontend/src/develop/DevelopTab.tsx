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
import { EllipsisVertical, Hammer, Users } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Problem } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Forget,
  OpenAt,
  OpenProject,
  SetWatch,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { StartInstances } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/launchsvc/service.ts'
import { OpenWeb } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/opener/service.ts'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { PageActions } from '../game/PageActions.tsx'
import { useNowValue } from '../i18n/useNow.ts'
import { relativeWhen } from '../i18n/when.ts'
import { LetterTile } from '../mods/parts.tsx'
import { useProfiles } from '../profiles/store.ts'
import { persist } from '../settings/persist.ts'
import { useSettings } from '../settings/store.ts'
import { MenuRule } from '../shell/TitleMenu.tsx'
import { space } from '../theme/density.ts'
import { reportError } from '../toasts/report.ts'
import {
  buildProject,
  packProject,
  refreshInfo,
  runOf,
  useDevRuns,
  useProjectInfo,
} from './builds.ts'
import { EditProjectDialog } from './EditProjectDialog.tsx'
import { NewModDialog } from './NewModDialog.tsx'
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

// The project's details from its own files, read when the card shows and kept current by refreshInfo.
function useInfo(game: string, dir: string) {
  const info = useProjectInfo((s) => s.byDir[dir])
  useEffect(() => {
    refreshInfo(game, dir).catch(() => undefined)
  }, [game, dir])
  return info
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
}: {
  game: string
  project: DevProject
  onRelease: () => void
  onEdit: () => void
}) {
  const { t } = useLingui()
  const [anchor, setAnchor] = useState<HTMLElement | null>(null)
  const running = useDevRuns((s) => runOf(s.byDir, project.dir).running)
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
        <MenuItem
          onClick={act(() =>
            OpenProject(game, project.dir).catch(reportError(t`Could not open the folder`)),
          )}
        >
          {t`Open folder`}
        </MenuItem>
        <MenuRule />
        <MenuItem
          onClick={act(() =>
            persist(() => Forget(game, project.dir), t`Could not save that setting`),
          )}
        >
          {t`Remove from list`}
        </MenuItem>
      </Menu>
    </>
  )
}

function ProjectCard({ game, project }: { game: string; project: DevProject }) {
  const { t } = useLingui()
  const running = useDevRuns((s) => runOf(s.byDir, project.dir).running)
  const info = useInfo(game, project.dir)
  const [releasing, setReleasing] = useState(false)
  const [editing, setEditing] = useState(false)
  const test = () =>
    StartInstances(game, project.profile, TEST_COPIES).catch(
      reportError(t`Could not start the game`),
    )
  if (info?.missing) {
    return <MissingCard game={game} project={project} />
  }
  const name = info?.name ?? project.name
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
        <LetterTile mod={{ id: name, name }} size={40} />
        <Box sx={{ minWidth: 0, flex: 1 }}>
          <Box sx={{ display: 'flex', alignItems: 'center', gap: 1 }}>
            <Typography sx={{ fontSize: 'large.fontSize', fontWeight: 600 }}>{name}</Typography>
            <Chip
              size="small"
              variant="outlined"
              label={info?.released ? `v${info.released}` : t`Not released`}
            />
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
        <MoreMenu
          game={game}
          project={project}
          onRelease={() => setReleasing(true)}
          onEdit={() => setEditing(true)}
        />
      </Box>
      <Outcome game={game} project={project} />
      {editing && info ? (
        <EditProjectDialog
          game={game}
          dir={project.dir}
          info={info}
          onClose={() => setEditing(false)}
        />
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

/** The open profile's own mods in development: start one from the template, then build, test, pack and release it. */
export function DevelopTab({ game }: { game: string }) {
  const { t } = useLingui()
  const projects = useSettings((s) => s.games?.[game]?.devProjects) ?? []
  const openId = useProfiles((s) => s.openId)
  const [creating, setCreating] = useState(false)
  const here = projects.filter((p) => p.profile === openId)
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
        p: space.gutter,
        display: 'flex',
        flexDirection: 'column',
        gap: space.gap,
      }}
    >
      <PageActions>
        <Button variant="contained" onClick={() => setCreating(true)}>
          {t`New mod…`}
        </Button>
      </PageActions>
      {here.length === 0 ? (
        <Typography sx={{ color: 'text.secondary' }}>
          {t`Start a mod from the template: Mortar sets up the project and installs each build into this profile.`}
        </Typography>
      ) : null}
      {here.map((p) => (
        <ProjectCard key={p.dir} game={game} project={p} />
      ))}
      {elsewhere.length > 0 ? (
        <Box sx={{ mt: 2 }}>
          <Typography sx={{ fontWeight: 600, mb: 0.5 }}>{t`In other profiles`}</Typography>
          {elsewhere.map((p) => (
            <ElsewhereRow key={p.dir} project={p} />
          ))}
        </Box>
      ) : null}
      <NewModDialog open={creating} game={game} onClose={() => setCreating(false)} />
    </Box>
  )
}
