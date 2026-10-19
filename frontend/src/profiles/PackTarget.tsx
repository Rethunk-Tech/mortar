import { plural } from '@lingui/core/macro'
import { useLingui } from '@lingui/react/macro'
import { Checkbox, FormControlLabel, MenuItem, Radio, RadioGroup, TextField } from '@mui/material'
import { listNames } from '../i18n/list.ts'
import type { PackTargetState } from './usePackTarget.ts'

// Where an imported pack goes: a new profile with a name of its own, or an existing profile the pack is added to, with
// the choice to remove what the pack does not list. An empty `target` is the new profile.
export function PackTarget({
  where,
  placeholder,
}: {
  where: PackTargetState
  placeholder: string
}) {
  const { t } = useLingui()
  const { target, name, existing, extras, dropExtras, replaceFiles } = where
  return (
    <>
      <RadioGroup
        sx={{ mt: 1 }}
        value={target === '' ? 'new' : 'update'}
        onChange={(event) =>
          where.choose(event.target.value === 'new' ? '' : (existing[0]?.id ?? ''))
        }
      >
        <FormControlLabel value="new" control={<Radio />} label={t`New profile`} />
        <FormControlLabel
          value="update"
          disabled={existing.length === 0}
          control={<Radio />}
          label={t`Update a profile`}
        />
      </RadioGroup>
      {target === '' ? (
        <TextField
          fullWidth={true}
          size="small"
          label={t`Profile name`}
          placeholder={placeholder}
          value={name}
          onChange={(event) => where.setName(event.target.value)}
        />
      ) : (
        <>
          <TextField
            select={true}
            fullWidth={true}
            size="small"
            label={t`Profile to update`}
            value={target}
            onChange={(event) => where.choose(event.target.value)}
          >
            {existing.map((p) => (
              <MenuItem key={p.id} value={p.id}>
                {p.name}
              </MenuItem>
            ))}
          </TextField>
          <FormControlLabel
            control={
              <Checkbox
                checked={replaceFiles}
                onChange={(event) => where.setReplaceFiles(event.target.checked)}
              />
            }
            label={t`Replace my config files with the pack's`}
          />
          {extras.length > 0 ? (
            <FormControlLabel
              control={
                <Checkbox
                  checked={dropExtras}
                  onChange={(event) => where.setDropExtras(event.target.checked)}
                />
              }
              label={t`${plural(extras.length, { one: 'Also remove # mod that is not in this code', other: 'Also remove # mods that are not in this code' })}: ${listNames(extras.map((e) => e.name))}`}
            />
          ) : null}
        </>
      )}
    </>
  )
}
