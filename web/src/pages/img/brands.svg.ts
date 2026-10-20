import { brandSprite } from '../../lib/brands.ts'

export const GET = () =>
  new Response(brandSprite(), { headers: { 'Content-Type': 'image/svg+xml' } })
