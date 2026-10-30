import { useLingui } from '@lingui/react/macro'
import { MenuItem } from '@mui/material'
import { FolderPlus } from 'lucide-react'
import { useEffect, useState } from 'react'
import type { AssetKind } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/components/models.ts'
import type { Adoption } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/models.ts'
import {
  Adopt,
  AssetKinds,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/devsvc/service.ts'
import { useProfiles } from '../profiles/store.ts'
import { ToneIcon } from '../shell/dialog/AppDialog.tsx'
import { DerivedValue, FormDialog, SelectField, TextInput } from '../shell/dialog/FormDialog.tsx'
import { reportUnexpected } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'
import { usePending } from '../toasts/usePending.ts'
import { useAdopting } from './adopting.ts'

const CODE = 'code'
const GENERIC = 'generic'
const CODE_NAME = /^[A-Za-z0-9][A-Za-z0-9._-]{0,63}$/
const MOD_NAME = /^[A-Z][A-Za-z0-9]{1,63}$/

export function AdoptHost() {
  const asked = useAdopting((s) => s.asked)
  const clear = useAdopting((s) => s.clear)
  return asked ? (
    <AdoptDialog game={asked.game} dir={asked.dir} found={asked.found} onClose={clear} />
  ) : null
}

/** The folder a player brought in that is not a Mortar project yet: what Mortar found, for them to confirm or change.
 * Nothing is written into the folder until Add. */
function AdoptDialog({
  game,
  dir,
  found,
  onClose,
}: {
  game: string
  dir: string
  found: Adoption
  onClose: () => void
}) {
  const { t } = useLingui()
  const openId = useProfiles((s) => s.openId)
  const [name, setName] = useState(found.name)
  const [kind, setKind] = useState(found.kind)
  const [assetKinds, setAssetKinds] = useState<AssetKind[]>([])
  const [pending, run] = usePending()
  useEffect(() => {
    AssetKinds(game)
      .then((kinds) => setAssetKinds(kinds ?? []))
      .catch(reportUnexpected)
  }, [game])
  const label = (id: string) => {
    if (id === CODE) {
      return t`Code (C# project)`
    }
    if (id === GENERIC) {
      return t`Files, installed as they are`
    }
    return assetKinds.find((k) => k.id === id)?.label ?? id
  }
  // A project of files is named like a mod, so a code project's name is tidied when the kind changes to one.
  const choose = (next: string) => {
    setKind(next)
    if (next !== CODE && !MOD_NAME.test(name.trim())) {
      setName(suggestedName(name))
    }
  }
  const add = () =>
    run(
      async () => {
        const project = await Adopt(game, dir, openId, { ...found, name: name.trim(), kind })
        useToasts
          .getState()
          .push({ kind: 'success', title: t`Added ${{ name: project.name }}`, body: project.dir })
        onClose()
      },
      { errorTitle: t`Could not add the project` },
    )
  const nameRule = kind === CODE ? CODE_NAME : MOD_NAME
  let issue = ''
  if (!nameRule.test(name.trim())) {
    issue =
      kind === CODE
        ? t`The name holds letters, digits, dots, dashes and underscores.`
        : t`The name starts with a capital letter and holds only letters and digits.`
  }
  return (
    <FormDialog
      open={true}
      onClose={onClose}
      size="sm"
      title={t`Add this folder as a project`}
      subtitle={t`Mortar writes only a small record into it, in .mortar/project.json.`}
      icon={
        <ToneIcon tone="neutral">
          <FolderPlus size={22} />
        </ToneIcon>
      }
      submitLabel={t`Add`}
      issue={issue}
      dirty={name !== found.name || kind !== found.kind}
      busy={pending}
      onSubmit={add}
    >
      <DerivedValue label={t`Folder`} value={dir} />
      <TextInput label={t`Name`} value={name} onChange={setName} maxLength={64} autoFocus={true} />
      <SelectField label={t`Kind`} value={kind} onChange={choose}>
        {(found.kinds ?? []).map((id) => (
          <MenuItem key={id} value={id}>
            {label(id)}
          </MenuItem>
        ))}
      </SelectField>
      {kind === CODE && found.csproj ? (
        <DerivedValue label={t`Project file`} value={found.csproj} />
      ) : null}
    </FormDialog>
  )
}

// A project of files is named like a mod: capitalised letters and digits.
function suggestedName(name: string) {
  const letters = name.replaceAll(/[^A-Za-z0-9]/g, '')
  return letters ? letters.charAt(0).toUpperCase() + letters.slice(1) : 'Mod'
}
