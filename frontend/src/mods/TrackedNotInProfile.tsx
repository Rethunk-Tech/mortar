import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import {
  Accordion,
  AccordionDetails,
  AccordionSummary,
  Box,
  Button,
  Typography,
} from '@mui/material'
import { ChevronDown, ExternalLink, Plus } from 'lucide-react'
import { useEffect, useState } from 'react'
import {
  ModName,
  TrackedMissing,
} from '../../bindings/github.com/Rethunk-AI/mortar/internal/nexussvc/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-AI/mortar/internal/profile/models.ts'
import { useProfiles } from '../profiles/store.ts'
import { useNexus } from '../settings/nexus.ts'
import { openImport } from '../share/store.ts'
import { reportUnexpected } from '../toasts/report.ts'
import { openPage } from './menu.ts'
import type { TrackedMod } from './nexusAccount.ts'
import { useLocked } from './useLocked.ts'

const FALLBACK_PREFIX = '#'

function nexusModURL(domainName: string, modId: number): string {
  return `https://www.nexusmods.com/${domainName}/mods/${modId}`
}

function TrackedRow({
  mod,
  profileId,
  premium,
  locked,
}: {
  mod: TrackedMod
  profileId: string
  premium: boolean
  locked: boolean
}) {
  const { t } = useLingui()
  const [name, setName] = useState(`${FALLBACK_PREFIX}${mod.modId}`)
  useEffect(() => {
    let live = true
    setName(`${FALLBACK_PREFIX}${mod.modId}`)
    ModName(mod.modId)
      .then((n) => {
        if (live && n) {
          setName(n)
        }
      })
      .catch(reportUnexpected)

    return () => {
      live = false
    }
  }, [mod.modId])
  const url = nexusModURL(mod.domainName, mod.modId)
  return (
    <Box
      sx={{
        display: 'flex',
        alignItems: 'center',
        gap: 1,
        flexWrap: 'wrap',
        py: 0.5,
      }}
    >
      <Typography sx={{ flex: 1, minWidth: 0, fontSize: 14 }}>{name}</Typography>
      <Button
        size="small"
        variant="outlined"
        startIcon={<ExternalLink size={14} aria-hidden={true} />}
        onClick={() => {
          openPage(url).catch(reportUnexpected)
        }}
      >
        {t`Open on Nexus`}
      </Button>
      {premium ? (
        <Button
          size="small"
          variant="contained"
          disabled={locked}
          startIcon={<Plus size={14} aria-hidden={true} />}
          onClick={() => openImport({ profileId, link: url })}
        >
          {t`Add`}
        </Button>
      ) : null}
    </Box>
  )
}

function TrackedNotInProfile({ profile }: { profile: Profile }) {
  const signedIn = useNexus((s) => s.signedIn)
  const premium = useNexus((s) => s.premium)
  const gameId = useProfiles((s) => s.game?.id)
  const locked = useLocked()
  const [mods, setMods] = useState<TrackedMod[]>([])
  useEffect(() => {
    if (!(signedIn && gameId)) {
      setMods([])
      return
    }
    let live = true
    TrackedMissing(gameId, profile.id)
      .then((list) => {
        if (live) {
          setMods(list ?? [])
        }
      })
      .catch(reportUnexpected)

    return () => {
      live = false
    }
  }, [signedIn, gameId, profile.id])
  if (!signedIn || mods.length === 0) {
    return null
  }
  const heading = plural(mods.length, {
    one: 'Tracked on Nexus, not in this profile (#)',
    other: 'Tracked on Nexus, not in this profile (#)',
  })
  return (
    <Accordion
      disableGutters={true}
      sx={{
        mx: 2,
        mb: 1,
        bgcolor: 'transparent',
        boxShadow: 'none',
        '&:before': { display: 'none' },
      }}
    >
      <AccordionSummary
        expandIcon={<ChevronDown size={16} />}
        sx={{ px: 0, minHeight: 36, '& .MuiAccordionSummary-content': { my: 0.5 } }}
      >
        <Typography sx={{ fontSize: 14, fontWeight: 600 }}>{heading}</Typography>
      </AccordionSummary>
      <AccordionDetails sx={{ px: 0, pt: 0, display: 'flex', flexDirection: 'column', gap: 0.5 }}>
        {mods.map((mod) => (
          <TrackedRow
            key={mod.modId}
            mod={mod}
            profileId={profile.id}
            premium={premium}
            locked={locked}
          />
        ))}
      </AccordionDetails>
    </Accordion>
  )
}

export { TrackedNotInProfile }
