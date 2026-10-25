import { useLingui } from '@lingui/react/macro'
import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'
import { Field } from '../shell/dialog/FormDialog.tsx'
import { PROFILE_ICONS } from './appearance.ts'
import { ProfileMark } from './ProfileMark.tsx'
import { ColorSwatches, IconSwatches } from './Swatches.tsx'

export function AppearancePickers({
  profile,
  color,
  icon,
  onColor,
  onIcon,
}: {
  profile: Profile
  color: string
  icon: string
  onColor: (next: string) => void
  onIcon: (next: string) => void
}) {
  const { t } = useLingui()
  const iconName = (token: string) => {
    switch (token) {
      case 'sprout':
        return t`Sprout`
      case 'leaf':
        return t`Leaf`
      case 'wheat':
        return t`Wheat`
      case 'fish':
        return t`Fish`
      case 'hammer':
        return t`Hammer`
      case 'pickaxe':
        return t`Pickaxe`
      case 'star':
        return t`Star`
      case 'heart':
        return t`Heart`
      case 'mountain':
        return t`Mountain`
      case 'sun':
        return t`Sun`
      case 'moon':
        return t`Moon`
      case 'sparkles':
        return t`Sparkles`
      default:
        return token
    }
  }
  return (
    <>
      <Field label={t`Colour`}>
        <ColorSwatches label={t`Colour`} value={color} onChange={onColor} />
      </Field>
      <Field label={t`Icon`}>
        <IconSwatches
          label={t`Icon`}
          value={icon}
          onChange={onIcon}
          options={PROFILE_ICONS.map((name) => ({
            value: name,
            name: iconName(name),
            face: <ProfileMark profile={{ ...profile, color, icon: name }} size={28} />,
          }))}
        />
      </Field>
    </>
  )
}
