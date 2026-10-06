import { Box, Link } from '@mui/material'
import type { ReactNode } from 'react'
import Markdown, { type Components } from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { openPage } from './menu.ts'

const WEB = /^https?:\/\//i

// Links open in the browser through Mortar rather than navigating the window; anything that is not a web address
// stays plain text.
function MdLink({ href, children }: { href?: string | undefined; children?: ReactNode }) {
  if (!(href && WEB.test(href))) {
    return <span>{children}</span>
  }
  return (
    <Link
      component="button"
      onClick={() => openPage(href)}
      sx={{ font: 'inherit', verticalAlign: 'baseline', textAlign: 'left' }}
    >
      {children}
    </Link>
  )
}

// Only https pictures load; a badge or screenshot over plain http or a data URL is dropped.
function MdImage({ src, alt }: { src?: string | undefined; alt?: string | undefined }) {
  if (!src?.startsWith('https://')) {
    return null
  }
  // A README picture's size is unknown until it loads, like the Browse card pictures drawn the same way.
  return (
    <Box component="img" src={src} alt={alt ?? ''} loading="lazy" referrerPolicy="no-referrer" />
  )
}

const components: Components = {
  a: ({ href, children }) => <MdLink href={href}>{children}</MdLink>,
  img: ({ src, alt }) => <MdImage src={typeof src === 'string' ? src : undefined} alt={alt} />,
}

// MarkdownView renders a mod's README or changelog. react-markdown builds React elements and raw HTML is dropped
// rather than shown as text, so nothing from a mod page reaches the DOM as markup.
export function MarkdownView({ source }: { source: string }) {
  return (
    <Box
      sx={{
        fontSize: 13,
        lineHeight: 1.5,
        overflowWrap: 'anywhere',
        '& > :first-of-type': { mt: 0 },
        '& h1, & h2, & h3, & h4, & h5, & h6': { fontSize: 14, fontWeight: 700, mt: 1.5, mb: 0.5 },
        '& h1, & h2': { fontSize: 15 },
        '& p': { my: 0.75 },
        '& ul, & ol': { my: 0.5, pl: 2.5 },
        '& li': { my: 0.25 },
        '& img': { maxWidth: '100%', height: 'auto', verticalAlign: 'middle' },
        '& code': {
          fontFamily: 'monospace',
          fontSize: 12,
          px: 0.5,
          borderRadius: '4px',
          bgcolor: 'var(--mortar-overlay-30)',
        },
        '& pre': {
          p: 1,
          borderRadius: '4px',
          overflowX: 'auto',
          bgcolor: 'var(--mortar-overlay-30)',
        },
        '& pre code': { p: 0, bgcolor: 'transparent' },
        '& blockquote': {
          m: 0,
          my: 0.75,
          pl: 1.5,
          borderLeft: '3px solid var(--mortar-hairline)',
          color: 'text.secondary',
        },
        '& hr': { border: 0, borderTop: '1px solid var(--mortar-hairline)', my: 1.5 },
        '& table': { borderCollapse: 'collapse', my: 0.75, display: 'block', overflowX: 'auto' },
        '& th, & td': {
          border: '1px solid var(--mortar-hairline)',
          px: 1,
          py: 0.5,
          textAlign: 'left',
        },
      }}
    >
      <Markdown remarkPlugins={[remarkGfm]} components={components} skipHtml={true}>
        {source}
      </Markdown>
    </Box>
  )
}
