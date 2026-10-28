import { useLingui } from '@lingui/react/macro'
import { Box, FormControlLabel, Radio, RadioGroup } from '@mui/material'
import type { AssetKind } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/components/models.ts'
import { Field } from '../shell/dialog/FormDialog.tsx'
import { space } from '../theme/density.ts'

// A Thunderstore Team-Name as people say it: x753-More_Suits is More Suits.
const packageTitle = (pkg: string) => pkg.slice(pkg.indexOf('-') + 1).replaceAll('_', ' ')

function Option({ value, label, detail }: { value: string; label: string; detail: string }) {
  return (
    <FormControlLabel
      value={value}
      sx={{ alignItems: 'flex-start', m: 0, gap: space.gap }}
      control={<Radio size="small" />}
      label={
        <Box sx={{ pt: 0.75 }}>
          <Box sx={{ fontSize: 'medium.fontSize', fontWeight: 500 }}>{label}</Box>
          <Box sx={{ fontSize: 'small.fontSize', color: 'text.secondary' }}>{detail}</Box>
        </Box>
      }
    />
  )
}

/** The wizard's first step when the game offers asset kinds: a code mod from the template, or one made only of files. */
export function KindStep({
  kinds,
  kind,
  set,
}: {
  kinds: AssetKind[]
  kind: string
  set: (kind: string) => void
}) {
  const { t } = useLingui()
  return (
    <Field label={t`Kind of mod`}>
      <RadioGroup value={kind} onChange={(_, v) => set(v)} sx={{ gap: space.gap }}>
        <Option
          value=""
          label={t`Code`}
          detail={t`A plugin in C#, built with the .NET SDK from the game's template.`}
        />
        {kinds.map((k) => (
          <Option
            key={k.id}
            value={k.id}
            label={k.label}
            detail={t`Files that ${{ name: packageTitle(k.dependency) }} loads. No SDK needed.`}
          />
        ))}
      </RadioGroup>
    </Field>
  )
}
