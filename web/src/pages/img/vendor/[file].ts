import type { GetStaticPaths } from 'astro'
import { ART, brandArt } from '../../../lib/brands.ts'

export const getStaticPaths = (() =>
  Object.values(ART).map((file) => ({ params: { file } }))) satisfies GetStaticPaths

const TYPES: Record<string, string> = { svg: 'image/svg+xml', png: 'image/png' }

export const GET = ({ params }: { params: { file: string } }) =>
  new Response(new Uint8Array(brandArt(params.file)), {
    headers: { 'Content-Type': TYPES[params.file.split('.').pop() ?? ''] ?? 'application/octet-stream' },
  })
