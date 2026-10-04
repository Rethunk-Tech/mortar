import { useLingui } from '@lingui/react/macro'
import { Box, Chip, Typography } from '@mui/material'
import { File, Folder } from 'lucide-react'
import { useCallback, useEffect, useRef, useState } from 'react'
import type { Preview } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/archive/models.ts'
import { ArchivePreview as ReadArchive } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/archivesvc/service.ts'
import { formatBytes } from '../i18n/bytes.ts'
import { ErrorRetry } from '../shell/ErrorRetry.tsx'
import { LoadingRow } from '../shell/LoadingRow.tsx'
import { type InlineError, inlineError } from '../toasts/report.ts'
import { groupEntries, type TreeRow } from './archiveTree.ts'

const INDENT = 14

function Row({ row }: { row: TreeRow }) {
  const { manifest } = row
  const Icon = row.isDir ? Folder : File
  const label = `${row.name}${manifest ? ` · ${manifest.name} ${manifest.version}` : ''}`
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 0.75,
        pl: `${row.depth * INDENT}px`,
        py: '1px',
        fontSize: 12,
        ...(manifest
          ? { bgcolor: 'var(--mortar-card-hover)', borderRadius: '3px', fontWeight: 600 }
          : {}),
      }}
    >
      <Icon size={13} style={{ flexShrink: 0 }} />
      <Box
        component="span"
        title={label}
        sx={{ flex: 1, minWidth: 0, overflow: 'hidden', textOverflow: 'ellipsis' }}
      >
        {label}
      </Box>
      {row.isDir ? null : (
        <Box component="span" sx={{ color: 'text.secondary', flexShrink: 0, pr: 0.5 }}>
          {formatBytes(row.size)}
        </Box>
      )}
    </Box>
  )
}

function Tree({ preview }: { preview: Preview }) {
  const { t } = useLingui()
  const groups = groupEntries(preview.entries ?? [], preview.manifests ?? [])
  return (
    <Box
      tabIndex={0}
      role="region"
      aria-label={t`Archive contents`}
      sx={{ flex: 1, minHeight: 0, overflowY: 'auto' }}
    >
      {groups.map((group) => (
        <Box key={group.folder} sx={{ mb: 1 }}>
          {group.rows.map((row) => (
            <Row key={row.path} row={row} />
          ))}
        </Box>
      ))}
    </Box>
  )
}

/** What an archive holds, read from its listing: a tree by top folder, its SMAPI manifests and a FOMOD flag. */
export function ArchivePreview({ path }: { path: string }) {
  const { t } = useLingui()
  const [preview, setPreview] = useState<Preview | null>(null)
  const [error, setError] = useState<InlineError | null>(null)
  const gen = useRef(0)
  const load = useCallback(() => {
    gen.current += 1
    const token = gen.current
    setPreview(null)
    setError(null)
    ReadArchive(path)
      .then((read) => token === gen.current && setPreview(read))
      .catch((e: unknown) => token === gen.current && setError(inlineError(e)))
  }, [path])
  useEffect(() => {
    load()
    return () => {
      gen.current += 1
    }
  }, [load])
  if (error) {
    return <ErrorRetry error={error} onRetry={load} />
  }
  if (!preview) {
    return <LoadingRow>{t`Reading the archive…`}</LoadingRow>
  }
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', minHeight: 0, gap: 1 }}>
      <Box sx={{ display: 'flex', alignItems: 'center', gap: 1, flexWrap: 'wrap' }}>
        <Typography sx={{ fontSize: 13, color: 'text.secondary' }}>
          {t`${formatBytes(preview.totalSize)} in total`}
        </Typography>
        {preview.fomod ? <Chip size="small" color="info" label={t`FOMOD installer`} /> : null}
      </Box>
      {preview.truncated ? (
        <Typography sx={{ fontSize: 12, color: 'text.secondary' }}>
          {t`Showing the first 2,000 files`}
        </Typography>
      ) : null}
      <Tree preview={preview} />
    </Box>
  )
}
