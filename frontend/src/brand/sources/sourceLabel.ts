const LABELS: Record<string, string> = {
  nexus: 'Nexus',
  github: 'GitHub',
  moddrop: 'ModDrop',
  thunderstore: 'Thunderstore',
}

export const sourceLabel = (id: string): string => LABELS[id] ?? id
