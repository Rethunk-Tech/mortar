import type { StarterTemplate } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/components/models.ts'
import type { Want } from '../queue/actions.ts'

// The downloads of a starter template's packages, in the catalog's order. Each goes through the normal queue, so
// dependencies, sign-in prompts and Nexus's free-account clicks behave as for any other install.
export function starterWants(template: StarterTemplate): Want[] {
  return (template.packages ?? []).map(({ source, ref, name }): Want => {
    switch (source) {
      case 'nexus':
        return { kind: 'install', name, modId: Number(ref), latest: true }
      case 'github':
        return { kind: 'install', name, repo: ref }
      case 'thunderstore':
        return { kind: 'install', name, package: ref }
      default:
        return { kind: 'install', name, source, package: ref }
    }
  })
}
