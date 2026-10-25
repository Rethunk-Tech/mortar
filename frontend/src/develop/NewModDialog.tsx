import { useLingui } from '@lingui/react/macro'
import {
  Alert,
  Box,
  Button,
  Checkbox,
  Dialog,
  DialogActions,
  DialogContent,
  DialogTitle,
  FormControlLabel,
  MenuItem,
  Radio,
  RadioGroup,
  TextField,
} from '@mui/material'
import { useEffect, useState } from 'react'
import type { Tools } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Create,
  Tools as ReadTools,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { PickFolder } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/picker/service.ts'
import { sourceLabel } from '../brand/sources/sourceLabel.ts'
import { useGameInfo } from '../games/info.ts'
import { useProfiles } from '../profiles/store.ts'
import { readStored, writeStored } from '../shell/useStoredState.ts'
import { space } from '../theme/density.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { buildProject } from './builds.ts'
import { type Draft, draftIssue, firstDraft, REMEMBERED, type Remembered } from './draft.ts'

// The profile picker's value for a new development profile; Create takes it as no profile.
const NEW_PROFILE = 'new'

function Prerequisites({ tools }: { tools: Tools | null }) {
  const { t } = useLingui()
  if (!tools) {
    return null
  }
  const missing = [
    (tools.dotnetSdks ?? []).length === 0 ? t`Install the .NET SDK to build mods.` : '',
    tools.git ? '' : t`Install git to keep your mod's history.`,
    tools.gameDir
      ? ''
      : t`Mortar cannot find the game, so the mod builds against its published libraries.`,
  ].filter((line) => line !== '')
  if (missing.length === 0) {
    return null
  }
  return (
    <Alert variant="compact" severity="warning">
      {missing.join(' ')}
    </Alert>
  )
}

function Source({
  draft,
  user,
  set,
}: {
  draft: Draft
  user: string
  set: (next: Partial<Draft>) => void
}) {
  const { t } = useLingui()
  return (
    <>
      <RadioGroup
        value={draft.github ? 'github' : 'copy'}
        onChange={(_, v) => set({ github: v === 'github' })}
      >
        <FormControlLabel
          value="github"
          disabled={user === ''}
          control={<Radio size="small" />}
          label={
            user
              ? t`Create a repository for it on ${{ source: sourceLabel('github') }}`
              : t`Sign in with gh to create a repository on ${{ source: sourceLabel('github') }}`
          }
        />
        <FormControlLabel
          value="copy"
          control={<Radio size="small" />}
          label={t`Copy the template here only`}
        />
      </RadioGroup>
      {draft.github ? (
        <Box sx={{ display: 'flex', gap: space.gap, alignItems: 'center' }}>
          <TextField
            size="small"
            sx={{ flex: 1 }}
            label={t`Account or organisation on ${{ source: sourceLabel('github') }}`}
            value={draft.owner}
            onChange={(e) => set({ owner: e.target.value })}
          />
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
    </>
  )
}

function Fields({ draft, set }: { draft: Draft; set: (next: Partial<Draft>) => void }) {
  const { t } = useLingui()
  const profiles = useProfiles((s) => s.profiles)
  const gameName = useGameInfo()?.name ?? ''
  const pick = async () => {
    const dir = await PickFolder(t`Folder for your mod projects`)
    if (dir) {
      set({ parent: dir })
    }
  }
  return (
    <>
      <TextField
        size="small"
        label={t`Mod name`}
        helperText={t`Letters and digits, starting with a capital: ShopDrop`}
        value={draft.name}
        onChange={(e) => set({ name: e.target.value })}
        autoFocus={true}
      />
      <TextField
        size="small"
        label={t`One-line description`}
        value={draft.description}
        onChange={(e) => set({ description: e.target.value })}
      />
      <Box sx={{ display: 'flex', gap: space.gap }}>
        <TextField
          size="small"
          label={t`Author`}
          value={draft.author}
          onChange={(e) => set({ author: e.target.value })}
        />
        <TextField
          size="small"
          label={t`Team on ${{ source: sourceLabel('thunderstore') }}`}
          value={draft.team}
          onChange={(e) => set({ team: e.target.value })}
        />
        <TextField
          size="small"
          label={t`Id prefix`}
          value={draft.guidPrefix}
          onChange={(e) => set({ guidPrefix: e.target.value })}
        />
      </Box>
      <Box sx={{ display: 'flex', gap: space.gap, alignItems: 'center' }}>
        <TextField
          size="small"
          label={t`Projects folder`}
          value={draft.parent}
          sx={{ flex: 1 }}
          slotProps={{ input: { readOnly: true } }}
        />
        <Button onClick={() => pick().catch(() => undefined)}>{t`Choose…`}</Button>
        <TextField
          size="small"
          label={t`Folder name`}
          value={draft.repo}
          onChange={(e) => set({ repo: e.target.value })}
        />
      </Box>
      <TextField
        select={true}
        size="small"
        label={t`Development profile`}
        value={draft.profile}
        onChange={(e) => set({ profile: e.target.value })}
      >
        <MenuItem
          value={NEW_PROFILE}
        >{t`New profile: ${{ name: `${gameName} Development` }}`}</MenuItem>
        {profiles.map((p) => (
          <MenuItem key={p.id} value={p.id}>
            {p.name}
          </MenuItem>
        ))}
      </TextField>
    </>
  )
}

export function NewModDialog({
  open,
  game,
  onClose,
}: {
  open: boolean
  game: string
  onClose: () => void
}) {
  const { t } = useLingui()
  const [tools, setTools] = useState<Tools | null>(null)
  const [draft, setDraft] = useState<Draft>(() =>
    firstDraft(readStored(REMEMBERED, {}, isRecord), ''),
  )
  const [pending, run] = usePending()
  useEffect(() => {
    if (!open) {
      return
    }
    ReadTools(game)
      .then((found) => {
        setTools(found)
        setDraft((d) =>
          d.owner || !found.githubUser
            ? d
            : { ...d, owner: found.githubUser, github: true, author: d.author || found.githubUser },
        )
      })
      .catch(() => setTools(null))
  }, [open, game])
  const set = (next: Partial<Draft>) => setDraft((d) => ({ ...d, ...next }))
  const issue = draftIssue(draft)
  const create = () =>
    run(
      async () => {
        const project = await Create({
          ...draft,
          game,
          repo: draft.repo || draft.name,
          profile: draft.profile === NEW_PROFILE ? '' : draft.profile,
        })
        await useProfiles.getState().load(game, useProfiles.getState().openId)
        writeStored(REMEMBERED, {
          author: draft.author,
          team: draft.team,
          guidPrefix: draft.guidPrefix,
          parent: draft.parent,
        })
        useToasts
          .getState()
          .push({ kind: 'success', title: t`Started ${{ name: project.name }}`, body: project.dir })
        onClose()
        await buildProject(game, project.dir)
      },
      { errorTitle: t`Could not start the mod` },
    )
  return (
    <Dialog open={open} onClose={pending ? undefined : onClose} maxWidth="sm" fullWidth={true}>
      <DialogTitle>{t`New mod`}</DialogTitle>
      <DialogContent
        sx={{
          display: 'flex',
          flexDirection: 'column',
          gap: space.pad,
          pt: `${space.gap} !important`,
        }}
      >
        <Prerequisites tools={tools} />
        <Fields draft={draft} set={set} />
        <Source draft={draft} user={tools?.githubUser ?? ''} set={set} />
      </DialogContent>
      <DialogActions sx={{ bgcolor: 'background.paper' }}>
        <Button onClick={onClose} disabled={pending}>
          {t`Cancel`}
        </Button>
        <Button
          variant="contained"
          onClick={create}
          disabled={pending || issue !== ''}
          title={issue}
        >
          {t`Start mod`}
        </Button>
      </DialogActions>
    </Dialog>
  )
}

const isRecord = (v: unknown): v is Remembered =>
  typeof v === 'object' && v !== null && Object.values(v).every((x) => typeof x === 'string')
