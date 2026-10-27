import { useLingui } from '@lingui/react/macro'
import {
  Autocomplete,
  Box,
  Checkbox,
  FormControlLabel,
  Radio,
  RadioGroup,
  TextField,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type { GitHubOwner } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import { CheckRepo } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { LetterTile } from '../mods/LetterTile.tsx'
import { Field } from '../shell/dialog/FormDialog.tsx'
import { type Draft, OWNER } from './draft.ts'

const CHECK_DELAY_MS = 400
const AVATAR_PX = 24

const Avatar = ({ owner }: { owner: GitHubOwner }) => (
  <LetterTile
    mod={{ id: owner.login, name: owner.login, picture: owner.avatar }}
    size={AVATAR_PX}
  />
)

// Asks GitHub, once typing pauses, whether owner exists and already has repo; the answer lands in the draft's ownerIssue.
function useRepoCheck(draft: Draft, set: (next: Partial<Draft>) => void) {
  const { t } = useLingui()
  // Null while the check runs; owner null when GitHub has no such account.
  const [checked, setChecked] = useState<{ owner: GitHubOwner | null } | null>(null)
  const owner = draft.owner
  const repo = draft.repo || draft.name
  const on = draft.github && OWNER.test(owner)
  useEffect(() => {
    setChecked(null)
    if (!on) {
      set({ ownerIssue: '' })
      return
    }
    set({ ownerIssue: t`Checking ${{ owner }} on ${{ source: sourceLabel('github') }}` })
    let stale = false
    const timer = setTimeout(() => {
      CheckRepo(owner, repo)
        .then((check) => {
          if (stale) {
            return
          }
          setChecked({ owner: check.owner })
          let issue = ''
          if (!check.owner) {
            issue = t`${{ source: sourceLabel('github') }} has no account named ${{ owner }}.`
          } else if (check.taken) {
            issue = t`${{ owner }}/${{ repo }} already exists.`
          }
          set({ ownerIssue: issue })
        })
        .catch(() => stale || set({ ownerIssue: '' }))
    }, CHECK_DELAY_MS)
    return () => {
      stale = true
      clearTimeout(timer)
    }
  }, [on, owner, repo, set, t])
  return checked
}

function OwnerField({
  draft,
  owners,
  set,
}: {
  draft: Draft
  owners: GitHubOwner[]
  set: (next: Partial<Draft>) => void
}) {
  const { t } = useLingui()
  const checked = useRepoCheck(draft, set)
  const shown = checked?.owner ?? owners.find((o) => o.login === draft.owner)
  const bad = checked !== null && draft.ownerIssue !== ''
  return (
    <Autocomplete
      freeSolo={true}
      fullWidth={true}
      size="small"
      options={owners.map((o) => o.login)}
      inputValue={draft.owner}
      onInputChange={(_, owner) => set({ owner: owner.trim() })}
      renderOption={({ key, ...props }, login) => {
        const owner = owners.find((o) => o.login === login)
        return (
          <Box component="li" key={key} {...props} sx={{ display: 'flex', gap: 1 }}>
            {owner ? <Avatar owner={owner} /> : null}
            {login}
          </Box>
        )
      }}
      renderInput={(params) => (
        <TextField
          {...params}
          error={bad || (draft.owner !== '' && !OWNER.test(draft.owner))}
          slotProps={{
            ...params.slotProps,
            htmlInput: {
              ...params.slotProps.htmlInput,
              'aria-label': t`${{ source: sourceLabel('github') }} account`,
            },
            input: {
              ...params.slotProps.input,
              startAdornment: shown ? <Avatar owner={shown} /> : null,
            },
          }}
        />
      )}
    />
  )
}

/** Where the project's repository lives: only on this computer, or also on GitHub under one of the player's accounts. */
export function RepositoryChoice({
  draft,
  owners,
  set,
}: {
  draft: Draft
  owners: GitHubOwner[]
  set: (next: Partial<Draft>) => void
}) {
  const { t } = useLingui()
  const github = sourceLabel('github')
  const signedIn = owners.length > 0
  return (
    <Field label={t`Repository`}>
      <RadioGroup
        value={draft.github ? 'github' : 'copy'}
        onChange={(_, v) => set({ github: v === 'github' })}
      >
        <FormControlLabel
          value="copy"
          control={<Radio size="small" />}
          label={t`Keep it on this computer`}
        />
        <FormControlLabel
          value="github"
          disabled={!signedIn}
          control={<Radio size="small" />}
          label={
            signedIn
              ? t`Also create a repository on ${{ source: github }}`
              : t`Sign in with gh to create a repository on ${{ source: github }}`
          }
        />
      </RadioGroup>
      {draft.github ? (
        <Box sx={{ display: 'flex', gap: 2, alignItems: 'center', pl: 4, mt: 1 }}>
          <OwnerField draft={draft} owners={owners} set={set} />
          <FormControlLabel
            control={
              <Checkbox
                size="small"
                checked={draft.private}
                onChange={(_, on) => set({ private: on })}
              />
            }
            label={t`Private`}
          />
        </Box>
      ) : null}
    </Field>
  )
}
