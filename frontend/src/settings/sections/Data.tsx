import { useLingui } from '@lingui/react/macro'
import { Box, Link } from '@mui/material'
import { useEffect, useState } from 'react'
import type { DataFolder as DataInfo } from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/models.ts'
import {
  DataFolder,
  OpenDataFolder,
} from '../../../bindings/github.com/Rethunk-AI/mortar/internal/settings/service.ts'
import { reportUnexpected } from '../../toasts/report.ts'

const BYTES_PER_MB = 1_000_000

export function Data() {
  const { t } = useLingui()
  const [data, setData] = useState<DataInfo | null>(null)
  useEffect(() => {
    DataFolder().then(setData).catch(reportUnexpected)
  }, [])
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: '6px' }}>
      <Box sx={{ fontSize: 14, fontWeight: 600 }}>{t`Mortar's data`}</Box>
      <Box sx={{ fontFamily: '"IBM Plex Mono", monospace', fontSize: 13, wordBreak: 'break-all' }}>
        {data
          ? t`${data.path} · ${(data.size / BYTES_PER_MB).toFixed(1)} MB · store, profiles, backups`
          : t`Measuring…`}
      </Box>
      <Link
        component="button"
        onClick={() => OpenDataFolder().catch(reportUnexpected)}
        sx={{ alignSelf: 'flex-start', fontSize: 13, whiteSpace: 'nowrap' }}
      >
        {t`Open folder`}
      </Link>
    </Box>
  )
}
