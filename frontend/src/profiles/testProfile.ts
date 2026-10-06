import type { Profile } from '../../bindings/github.com/Rethunk-Tech/mortar/internal/profile/models.ts'

export const testProfile = (partial: Partial<Profile> & Pick<Profile, 'id' | 'name'>): Profile => ({
  notes: '',
  cover: '',
  order: 0,
  created: '',
  updated: '',
  entries: null,
  ...partial,
})
