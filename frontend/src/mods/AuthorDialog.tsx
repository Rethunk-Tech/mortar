import { useLingui } from '@lingui/react/macro'
import {
  Box,
  Button,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  Link,
  Typography,
} from '@mui/material'
import { useMemo } from 'react'
import type {
  Mod,
  Profile,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { EmptyState } from '../shell/EmptyState.tsx'
import { nexusIdOf } from './lookup.ts'
import { openPage } from './menu.ts'
import { modsByAuthor } from './modsByAuthor.ts'
import { nexusAuthorPageUrl } from './nexusAuthorPage.ts'
import { useNexusEntry } from './nexusDetails.ts'

const text = { fontSize: 13 } as const
const muted = { fontSize: 12, color: 'text.secondary' } as const

function modPageUrl(profile: Profile, mod: Mod, gameId: string): string {
  const source = (profile.entries ?? []).find((e) => e.key === mod.key)?.source
  if (source?.kind !== 'nexus' || !source.modId) {
    return ''
  }
  const domain = gameId === 'stardew' ? 'stardewvalley' : gameId
  return `https://www.nexusmods.com/${domain}/mods/${source.modId}`
}

function ModRow({
  name,
  uniqueId,
  profiles,
  enabledLabel,
  disabledLabel,
}: {
  name: string
  uniqueId: string
  profiles: { profileName: string; version: string; enabled: boolean }[]
  enabledLabel: string
  disabledLabel: string
}) {
  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 0.5 }}>
      <Typography sx={{ fontWeight: 600, ...text }}>{name}</Typography>
      <Typography sx={muted}>{uniqueId}</Typography>
      {profiles.map((row) => (
        <Typography key={`${row.profileName}-${row.version}`} sx={text}>
          {`${row.profileName} · ${row.version} · ${row.enabled ? enabledLabel : disabledLabel}`}
        </Typography>
      ))}
    </Box>
  )
}

function NexusAuthorBlock({ nexusName, authorUrl }: { nexusName: string; authorUrl: string }) {
  const { t } = useLingui()
  if (nexusName === '') {
    return null
  }
  return (
    <Box>
      <Typography sx={muted}>{t`On Nexus`}</Typography>
      <Typography sx={text}>{nexusName}</Typography>
      {authorUrl === '' ? null : (
        <Link
          component="button"
          onClick={() => openPage(authorUrl)}
          sx={{ ...text, alignSelf: 'flex-start' }}
        >
          {t`Open author page on Nexus`}
        </Link>
      )}
    </Box>
  )
}

function AuthorDialogBody({
  author,
  seedMod,
  seedProfile,
}: {
  author: string
  seedMod?: Mod | undefined
  seedProfile?: Profile | undefined
}) {
  const { t } = useLingui()
  const gameId = useProfiles((s) => s.game?.id ?? '')
  const profiles = useProfiles((s) => s.profiles)
  const rows = useMemo(() => modsByAuthor(profiles, author), [profiles, author])
  const nexusId = seedMod && seedProfile ? nexusIdOf(seedProfile, seedMod) : 0
  const details = useNexusEntry(nexusId)?.details
  const page = details?.page
  const nexusName = page?.author?.trim() || page?.uploadedBy?.trim() || ''
  const modPage = useMemo(() => {
    if (!(seedMod && seedProfile)) {
      return ''
    }
    return modPageUrl(seedProfile, seedMod, gameId)
  }, [seedMod, seedProfile, gameId])
  const authorUrl =
    page && nexusName !== '' ? nexusAuthorPageUrl(page.uploaderUrl ?? '', modPage) : ''

  if (rows.length === 0) {
    return (
      <EmptyState compact={true} icon={null} title={t`No installed mods for this author.`}>
        {t`Try another spelling, or install a mod by ${author} in any profile.`}
      </EmptyState>
    )
  }

  return (
    <Box sx={{ display: 'flex', flexDirection: 'column', gap: 2 }}>
      <NexusAuthorBlock nexusName={nexusName} authorUrl={authorUrl} />
      <Box sx={{ display: 'flex', flexDirection: 'column', gap: 1.5 }}>
        <Typography
          sx={{ fontWeight: 700, fontSize: 14 }}
        >{t`Installed in your profiles`}</Typography>
        {rows.map((row) => (
          <ModRow
            key={row.uniqueId}
            name={row.name}
            uniqueId={row.uniqueId}
            profiles={row.profiles}
            enabledLabel={t`Enabled`}
            disabledLabel={t`Off`}
          />
        ))}
      </Box>
    </Box>
  )
}

export function AuthorDialog({
  open,
  author,
  seedMod,
  seedProfile,
  onClose,
}: {
  open: boolean
  author: string
  seedMod?: Mod | undefined
  seedProfile?: Profile | undefined
  onClose: () => void
}) {
  const { t } = useLingui()
  return (
    <Dialog open={open} onClose={onClose} fullWidth={true} maxWidth="sm">
      <DialogTitle>{t`Author · ${author}`}</DialogTitle>
      <DialogContent>
        {open ? (
          <AuthorDialogBody author={author} seedMod={seedMod} seedProfile={seedProfile} />
        ) : null}
      </DialogContent>
      <DialogActions>
        <Button onClick={onClose}>{t`Close`}</Button>
      </DialogActions>
    </Dialog>
  )
}
