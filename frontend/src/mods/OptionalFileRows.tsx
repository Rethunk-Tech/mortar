import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Box, Button, Chip, Radio, Typography } from '@mui/material'
import type { File } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/nexus/models.ts'
import type { OverlayFileSet } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { formatKb } from '../i18n/bytes.ts'
import { download } from '../queue/actions.ts'
import { useNexus } from '../settings/nexus.ts'
import { Fold } from '../shell/Fold.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { LockedReason } from './LockedReason.tsx'
import { openPage } from './menu.ts'
import { nexusDomain } from './nexusDomain.ts'
import { nexusModUrl } from './nexusUrl.ts'
import { plainDescription } from './optionalFiles.ts'
import { useLocked } from './useLocked.ts'
import type { OverlayRow } from './virtualRows.ts'

const text = { fontSize: 13, overflowWrap: 'anywhere' } as const
const muted = { fontSize: 12, color: 'text.secondary' } as const
const chipSx = { height: 20, fontSize: 11, '& .MuiChip-label': { px: 0.75 } } as const

/** Size, version and the first lines of a Nexus file's description. */
function FileFacts({ file }: { file: File | undefined }) {
  if (!file) {
    return null
  }
  const description = plainDescription(file.description)
  return (
    <>
      <Typography sx={muted}>
        {[formatKb(file.sizeKb), file.version].filter(Boolean).join(' · ')}
      </Typography>
      {description ? (
        <Typography
          title={description}
          sx={{
            ...muted,
            overflow: 'hidden',
            display: '-webkit-box',
            WebkitBoxOrient: 'vertical',
            WebkitLineClamp: 2,
          }}
        >
          {description}
        </Typography>
      ) : null}
    </>
  )
}

/** The main mod's files an optional file replaces, and the files it adds. */
function FilePaths({ set }: { set: OverlayFileSet | undefined }) {
  const { i18n, t } = useLingui()
  const replaces = set?.replaces ?? []
  const adds = set?.adds ?? []
  const added = adds.length
  if (replaces.length + added === 0) {
    return null
  }
  const paths = (list: string[]) =>
    list.map((p) => (
      <Typography key={p} sx={{ ...muted, overflowWrap: 'anywhere' }}>
        {p}
      </Typography>
    ))
  return (
    <Fold
      title={i18n._(
        plural(replaces.length, {
          one: `Replaces # file · adds ${added}`,
          other: `Replaces # files · adds ${added}`,
        }),
      )}
    >
      {replaces.length > 0 ? (
        <Typography sx={{ ...muted, fontWeight: 700 }}>{t`Replaces`}</Typography>
      ) : null}
      {paths(replaces)}
      {adds.length > 0 ? (
        <Typography sx={{ ...muted, fontWeight: 700 }}>{t`Adds`}</Typography>
      ) : null}
      {paths(adds)}
    </Fold>
  )
}

/** An optional file in the profile: a radio when it is one of a set of alternatives. */
export function InstalledOptional({
  row,
  file,
  set,
  radio,
  onChoose,
}: {
  row: OverlayRow
  file: File | undefined
  set: OverlayFileSet | undefined
  radio: boolean
  onChoose: () => void
}) {
  const { t } = useLingui()
  const locked = useLocked()
  return (
    <Box sx={{ display: 'flex', gap: 0.5, alignItems: 'flex-start' }}>
      {radio ? (
        <LockedReason locked={locked}>
          <Radio
            size="small"
            checked={row.enabled}
            disabled={locked}
            onChange={onChoose}
            slotProps={{ input: { 'aria-label': t`Use ${row.label}` } }}
            sx={{ p: 0.25 }}
          />
        </LockedReason>
      ) : null}
      <Box sx={{ minWidth: 0, flex: 1, display: 'flex', flexDirection: 'column', gap: 0.25 }}>
        <Box sx={{ display: 'flex', gap: 1, alignItems: 'center', flexWrap: 'wrap' }}>
          <Typography sx={text}>{file?.name || row.label}</Typography>
          <Chip size="small" color="primary" variant="outlined" label={t`Installed`} sx={chipSx} />
          {row.enabled || radio ? null : <Chip size="small" label={t`Off`} sx={chipSx} />}
        </Box>
        <FileFacts file={file} />
        <FilePaths set={set} />
      </Box>
    </Box>
  )
}

/** A Nexus optional file the profile does not have: Premium queues it to go on top, others open the files page. */
export function NexusOptional({
  file,
  modId,
  modName,
}: {
  file: File
  modId: number
  modName: string
}) {
  const { t } = useLingui()
  const locked = useLocked()
  const premium = useNexus((s) => s.signedIn && s.premium)
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.25, alignItems: 'flex-start' }}>
      <Typography sx={text}>{file.name || file.fileName}</Typography>
      <FileFacts file={file} />
      {premium ? (
        <LockedReason locked={locked}>
          <Button
            size="small"
            variant="outlined"
            disabled={locked}
            onClick={() =>
              download([
                {
                  kind: 'install',
                  modId,
                  fileId: file.fileId,
                  name: modName,
                  fileName: file.fileName,
                  version: file.version,
                },
              ]).catch(reportUnexpected)
            }
          >
            {t`Download and add on top`}
          </Button>
        </LockedReason>
      ) : (
        <Button
          size="small"
          variant="outlined"
          onClick={() => openPage(nexusModUrl(modId, nexusDomain(), 'files'))}
        >
          {t`Open files page`}
        </Button>
      )}
    </Box>
  )
}
