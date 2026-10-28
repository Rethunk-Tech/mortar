import { useLingui } from '@lingui/react/macro'
import { Alert, Box, Link } from '@mui/material'
import type { Problem } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { OpenAt } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import type { DevProject } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/models.ts'
import { reportError } from '../toasts/report.ts'
import { runOf, useDevRuns } from './builds.ts'

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
    <Box
      component="ul"
      sx={{ m: 0, p: 0, listStyle: 'none', display: 'flex', flexDirection: 'column', gap: 0.5 }}
    >
      {problems.map((p) => (
        <Box component="li" key={`${p.file}:${p.line}:${p.column}:${p.code}`} sx={{ minWidth: 0 }}>
          <Box sx={{ overflowWrap: 'anywhere' }}>{p.message}</Box>
          <Link
            component="button"
            color="inherit"
            onClick={() =>
              OpenAt(game, project.dir, p.file, p.line, p.column).catch(reportError(failed))
            }
            sx={{
              textAlign: 'left',
              fontFamily: 'monospace',
              fontSize: 'small.fontSize',
              opacity: 0.8,
              overflowWrap: 'anywhere',
            }}
          >
            {`${relative(project.dir, p.file)}:${p.line}${p.code ? ` · ${p.code}` : ''}`}
          </Link>
        </Box>
      ))}
    </Box>
  )
}

export function Outcome({ game, project }: { game: string; project: DevProject }) {
  const { t } = useLingui()
  const run = useDevRuns((s) => runOf(s.byDir, project.dir))
  const build = run.build
  const pack = run.pack
  const problems = build?.problems ?? []
  return (
    <>
      {build && (!build.ok || problems.length > 0) ? (
        <Alert
          variant="compact"
          severity={build.ok ? 'warning' : 'error'}
          sx={{ '& .MuiAlert-message': { minWidth: 0, flex: 1 } }}
        >
          {build.ok ? null : <Box sx={{ fontWeight: 600 }}>{t`The build failed.`}</Box>}
          {problems.length > 0 ? (
            <Problems game={game} project={project} problems={problems} />
          ) : null}
        </Alert>
      ) : null}
      {pack && !pack.ok ? (
        <Alert
          variant="compact"
          severity="error"
          sx={{ '& .MuiAlert-message': { minWidth: 0, flex: 1 } }}
        >
          <Box sx={{ fontWeight: 600 }}>{t`The build failed.`}</Box>
          {(pack.problems ?? []).length > 0 ? (
            <Problems game={game} project={project} problems={pack.problems ?? []} />
          ) : null}
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
