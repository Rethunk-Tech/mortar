import { BrandMark } from '../BrandMark.tsx'
import nexusmods from '../vendor/nexusmods.svg'
import { ICONS } from './sourceIcons.ts'

// The Nexus artwork leaves a margin inside its square that the vector marks do not, so it is drawn larger to match.
const NEXUS_SCALE = 1.2

function SourceLogo({ id, size }: { id: string; size: number }) {
  if (id === 'nexus') {
    const drawn = Math.round(size * NEXUS_SCALE)
    return (
      <img
        src={nexusmods}
        alt=""
        width={drawn}
        height={drawn}
        style={{ margin: (size - drawn) / 2 }}
      />
    )
  }
  const icon = ICONS[id]
  if (!icon) {
    return null
  }
  return <BrandMark path={icon.path} size={size} fill={icon.fill} />
}

export { SourceLogo }
