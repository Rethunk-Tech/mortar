import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { Link2 } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { Hosted } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/models.ts'
import {
  CreateHostedShare,
  UpdateHostedShare,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/sharesvc/service.ts'
import { Field } from '../shell/dialog/FormDialog.tsx'
import { space } from '../theme/density.ts'
import { MONO } from '../theme/theme.ts'
import { usePending } from '../toasts/usePending.ts'
import { copyText } from './copyText.ts'
import type { ShownInfo } from './logic.ts'
import type { ShareInclude } from './shareDefaults.ts'
import { toShareInclude } from './shareDefaults.ts'

const note = { fontSize: 'base.fontSize', color: 'text.secondary', lineHeight: 1.5 } as const

function hostedOf(info: ShownInfo): Hosted | null {
  return info.hosted?.code ? info.hosted : null
}

export function HostedLink({
  game,
  profileId,
  keys,
  include,
  info,
}: {
  game: string
  profileId: string
  keys: string[]
  include: ShareInclude
  info: ShownInfo
}) {
  const { t } = useLingui()
  const [hosted, setHosted] = useState<Hosted | null>(() => hostedOf(info))
  const [busy, run] = usePending()
  useEffect(() => {
    setHosted(hostedOf(info))
  }, [info])
  if (info.tooLarge) {
    return null
  }
  const publish = (update: boolean) =>
    run(
      async () => {
        const next = await (update
          ? UpdateHostedShare(game, profileId, keys, toShareInclude(include))
          : CreateHostedShare(game, profileId, keys, toShareInclude(include)))
        setHosted(next)
        copyText(next.url, t`Short link copied`)
      },
      { errorTitle: update ? t`Could not update the short link` : t`Could not get a short link` },
    )
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
      <Typography sx={note}>
        {t`A short link stores the mod list on mortar.rethunk.tech. The long link keeps it private.`}
      </Typography>
      {hosted ? (
        <Field label={t`Short link`} hint={t`Version ${{ version: hosted.version }}`}>
          <Box
            sx={{
              display: 'flex',
              alignItems: 'center',
              height: space.control,
              px: 1.5,
              bgcolor: 'var(--mortar-raised)',
              borderRadius: '6px',
              fontFamily: MONO,
              fontSize: 'base.fontSize',
              userSelect: 'text',
            }}
          >
            <Typography noWrap={true} title={hosted.url} sx={{ font: 'inherit' }}>
              {hosted.url}
            </Typography>
          </Box>
        </Field>
      ) : null}
      <Box>
        <Button
          variant="outlined"
          color="inherit"
          startIcon={<Link2 size={16} />}
          disabled={busy || info.count === 0}
          onClick={() => publish(hosted !== null)}
        >
          {hosted ? t`Update short link` : t`Get short link`}
        </Button>
      </Box>
    </Box>
  )
}
