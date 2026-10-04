import { useLingui } from '@lingui/react/macro'
import { Box, Link, Typography } from '@mui/material'
import { CalendarDays, Clock, Coins } from 'lucide-react'
import type { ReactNode } from 'react'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import type { Fit } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/savessvc/models.ts'
import { formatWhen } from '../i18n/formatWhen.ts'
import { useNow } from '../i18n/useNow.ts'
import { When } from '../i18n/When.tsx'
import { absoluteWhen } from '../i18n/when.ts'
import { useProfiles } from '../profiles/store.ts'
import { goldText, hoursPlayed } from './card.ts'
import { AddAll, MissingChips } from './lackChips.tsx'

const nowrap = { whiteSpace: 'nowrap' } as const

function Stat({ icon, children }: { icon: ReactNode; children: ReactNode }) {
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 0.75,
        fontSize: 13,
        color: 'var(--mortar-ink-85)',
        ...nowrap,
      }}
    >
      <Box component="span" sx={{ display: 'flex', color: 'text.secondary' }}>
        {icon}
      </Box>
      {children}
    </Box>
  )
}

export function SaveDetails({
  fit,
  profile,
  game,
  lastLine,
  lastGone,
  status,
}: {
  fit: Fit
  profile: Profile
  game: string
  lastLine: string
  lastGone: boolean
  status: ReactNode
}) {
  const { t, i18n } = useLingui()
  useNow()
  const seasons = [t`Spring`, t`Summer`, t`Fall`, t`Winter`]
  const hours = hoursPlayed(fit.millisecondsPlayed)
  const missing = fit.missing ?? []
  return (
    <>
      <Box sx={{ display: 'flex', flexWrap: 'wrap', columnGap: 2, rowGap: 0.75 }}>
        {fit.day > 0 ? (
          <Stat icon={<CalendarDays size={14} />}>
            {t`${seasons[fit.season] ?? ''} ${fit.day}, year ${fit.year}`}
          </Stat>
        ) : null}
        {hours > 0 ? <Stat icon={<Clock size={14} />}>{t`${hours}h played`}</Stat> : null}
        {fit.money ? <Stat icon={<Coins size={14} />}>{goldText(fit.money)}</Stat> : null}
      </Box>
      <Box
        sx={{
          display: 'flex',
          alignItems: 'center',
          gap: 1,
          pt: 1,
          borderTop: '1px solid var(--mortar-hairline-muted)',
        }}
      >
        <Typography
          sx={{ flex: 1, minWidth: 0, fontSize: 12, color: 'text.secondary' }}
          noWrap={true}
          title={fit.played ? absoluteWhen(fit.played, i18n.locale) || undefined : undefined}
        >
          {t`Last played ${formatWhen(fit.played)}`}
        </Typography>
        {fit.lastProfileId ? (
          <Typography
            sx={{ fontSize: 12, color: 'text.secondary' }}
            noWrap={true}
            title={lastGone ? fit.lastProfileId : lastLine}
          >
            {lastGone || fit.lastProfileId === profile.id ? (
              lastLine
            ) : (
              <Link
                component="button"
                onClick={() => useProfiles.getState().open(fit.lastProfileId)}
                sx={{ fontSize: 'inherit', color: 'inherit', verticalAlign: 'baseline' }}
              >
                {lastLine}
              </Link>
            )}
            {fit.lastProfileAt ? (
              <>
                {' '}
                <When value={fit.lastProfileAt} />
              </>
            ) : null}
          </Typography>
        ) : null}
        {status}
        {missing.length === 0 ? null : <AddAll missing={missing} profile={profile} />}
      </Box>
      <MissingChips fit={fit} profile={profile} game={game} />
    </>
  )
}
