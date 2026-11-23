import { SetByKey } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/settings/service.ts'
import { useSettings } from '../settings/store.ts'
import { MenuCheck } from '../shell/MenuAction.tsx'
import { MenuHeading } from '../shell/TitleMenu.tsx'
import { reportUnexpected } from '../toasts/report.ts'

function persistView(key: 'implicitModsView' | 'modPartsView', view: string) {
  useSettings.setState({ [key]: view })
  SetByKey(key, view, '').catch(reportUnexpected)
}

// A list-view setting as a titled set of radio items: how implicit mods appear, and whether a download's extra mods do.
export function ViewItems({
  setting,
  heading,
  options,
}: {
  setting: 'implicitModsView' | 'modPartsView'
  heading: string
  options: { id: string; label: string }[]
}) {
  const view = useSettings((s) => s[setting])
  return (
    <>
      <MenuHeading>{heading}</MenuHeading>
      {options.map((item) => (
        <MenuCheck
          key={item.id}
          radio={true}
          checked={view === item.id}
          label={item.label}
          onClick={() => persistView(setting, item.id)}
        />
      ))}
    </>
  )
}
