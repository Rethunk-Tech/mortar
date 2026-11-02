import {
  Clock,
  Download,
  HardDrive,
  Heart,
  type LucideIcon,
  Star,
  Tag,
  ThumbsUp,
  Users,
} from 'lucide-react'
import type { CardStatPart, StatIcon } from './stats.ts'

const STAT_ICONS: Record<StatIcon, LucideIcon> = {
  like: Heart,
  endorsement: ThumbsUp,
  star: Star,
  follower: Users,
  download: Download,
  size: HardDrive,
  version: Tag,
  updated: Clock,
}

/** Icons and short figures; each stat's tooltip and accessible name has its whole wording. */
export function StatParts({ stats }: { stats: CardStatPart[] }) {
  return stats.map(({ id, icon, short, full }) => {
    const Icon = STAT_ICONS[icon]
    return (
      <span key={id} className="card-stat" role="img" aria-label={full} title={full}>
        <Icon size={12} aria-hidden={true} />
        {short}
      </span>
    )
  })
}
