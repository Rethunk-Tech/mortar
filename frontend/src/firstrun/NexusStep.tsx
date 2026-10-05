import { useLingui } from '@lingui/react/macro'
import { Box, Button, Typography } from '@mui/material'
import { useEffect, useRef } from 'react'
import { useNexus } from '../settings/nexus.ts'
import { NexusSignIn } from '../settings/sections/NexusMods.tsx'
import { Panel } from './Panel.tsx'

// Mod Manager Download buttons on Nexus only reach Mortar for a signed-in account, so ask before the first mod.
export function NexusStep({ onDone }: { onDone: () => void }) {
  const { t } = useLingui()
  const { signedIn, name } = useNexus()
  const continueRef = useRef<HTMLButtonElement>(null)
  useEffect(() => {
    if (signedIn) {
      continueRef.current?.focus()
    }
  }, [signedIn])
  return (
    <Panel>
      <Typography variant="h6" component="h2">{t`Sign in to Nexus Mods`}</Typography>
      <Typography sx={{ fontSize: 14, color: 'text.secondary' }}>
        {t`Mod Manager Download buttons on Nexus Mods only work in Mortar when you are signed in.`}
      </Typography>
      <NexusSignIn />
      <Box role="status">
        {signedIn ? <Typography>{t`Signed in as ${name}`}</Typography> : null}
      </Box>
      <Button
        ref={continueRef}
        variant={signedIn ? 'contained' : 'text'}
        onClick={onDone}
        sx={{ alignSelf: 'flex-end' }}
      >
        {signedIn ? t`Continue` : t`Skip for now`}
      </Button>
    </Panel>
  )
}
