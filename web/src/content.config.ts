import { defineCollection } from 'astro:content'
import { glob } from 'astro/loaders'
import { z } from 'astro/zod'

// A release post names its version and takes its date and notes from the release; a how-to carries its own date.
const news = defineCollection({
  loader: glob({ pattern: '*.md', base: './src/content/news' }),
  schema: z.object({
    kind: z.enum(['release', 'howto']),
    title: z.string(),
    summary: z.string(),
    version: z.string().optional(),
    date: z.coerce.date().optional(),
    image: z.string().optional(),
    highlights: z
      .array(z.object({ area: z.string(), title: z.string(), text: z.string(), image: z.string() }))
      .optional(),
  }),
})

export const collections = { news }
