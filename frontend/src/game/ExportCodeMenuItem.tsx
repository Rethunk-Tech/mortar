import { useLingui } from '@lingui/react/macro'
import { Clipboard } from '@wailsio/runtime'
import { Package, Share2 } from 'lucide-react'
import { useState } from 'react'
import {
  ExportCode,
  ExportModpackDialog,
} from '../../bindings/github.com/Rethunk-Tech/mortar/internal/packsvc/service.ts'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { listNames } from '../i18n/list.ts'
import { hasThunderstore } from '../profiles/packImport.ts'
import { useProfiles } from '../profiles/store.ts'
import { ConfirmDialog } from '../shell/ConfirmDialog.tsx'
import { MenuAction } from '../shell/MenuAction.tsx'
import { reportError, reportUnexpected, toastError } from '../toasts/report.ts'
import { useToasts } from '../toasts/store.ts'

// Publishes the profile's Thunderstore packages as an r2modman code; it sends the list to a public service, so it asks
// first.
export function ExportCodeMenuItem({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  const [asking, setAsking] = useState(false)
  if (!(game && hasThunderstore(game))) {
    return null
  }
  const copy = async (code: string) => {
    try {
      await Clipboard.SetText(code)
    } catch (error) {
      reportUnexpected(error)
    }
  }
  const announce = (code: string) =>
    useToasts.getState().push({
      kind: 'success',
      title: t`Code published`,
      body: code,
      action: { label: t`Copy code`, run: () => copy(code) },
    })
  const publish = () => {
    setAsking(false)
    ExportCode(game.id, profile.id, true)
      .then(announce)
      .catch((error: unknown) => toastError(t`Could not publish the code`, error))
  }
  return (
    <>
      <MenuAction
        icon={<Share2 size={16} />}
        label={t`Share as r2modman code…`}
        onClick={() => {
          close()
          setAsking(true)
        }}
      />
      <ConfirmDialog
        open={asking}
        title={t`Share as r2modman code?`}
        body={t`This uploads the profile to Thunderstore and gives a public code.`}
        confirmLabel={t`Upload`}
        onCancel={() => setAsking(false)}
        onConfirm={publish}
      />
    </>
  )
}

// Saves the profile's enabled Thunderstore packages, with its config folder, as a Thunderstore modpack zip.
export function ExportModpackMenuItem({ profile, close }: { profile: Profile; close: () => void }) {
  const { t } = useLingui()
  const game = useProfiles((s) => s.game)
  if (!(game && hasThunderstore(game))) {
    return null
  }
  const save = async () => {
    const r = await ExportModpackDialog(game.id, profile.id, true)
    if (!r.path) {
      return
    }
    const left = [...(r.leftOut ?? []), ...(r.disabled ?? [])]
    useToasts.getState().push({
      kind: left.length > 0 ? 'warning' : 'success',
      title: t`Modpack saved`,
      body: left.length > 0 ? t`Left out: ${listNames(left)}` : r.path,
    })
  }
  return (
    <MenuAction
      icon={<Package size={16} />}
      label={t`Export as Thunderstore modpack…`}
      onClick={() => {
        close()
        save().catch(reportError(t`Could not export the modpack`))
      }}
    />
  )
}
