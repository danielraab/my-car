import { describe, expect, it } from 'vitest'

import type { AttachmentLimits } from '#/api/client'

import { formatBytes, prepareFiles } from './files'

const limits: AttachmentLimits = {
  maxFileBytes: 10,
  maxFilesPerEntry: 3,
  allowedContentTypes: [
    'image/jpeg',
    'image/png',
    'image/webp',
    'image/gif',
    'application/pdf',
  ],
}

const file = (name: string, type: string, size = 5) =>
  new File(['x'.repeat(size)], name, { type })

describe('prepareFiles', () => {
  it('refuses unsupported types, oversized files and files above the count', async () => {
    const result = await prepareFiles(
      [
        file('notes.docx', 'application/msword'),
        file('logo.svg', 'image/svg+xml'),
        file('big.pdf', 'application/pdf', 11),
        file('a.pdf', 'application/pdf'),
        file('b.gif', 'image/gif'),
        file('c.pdf', 'application/pdf'),
      ],
      limits,
      1,
    )
    expect(result.accepted.map((f) => f.name)).toEqual(['a.pdf', 'b.gif'])
    expect(result.refused).toEqual([
      { name: 'notes.docx', reason: 'type' },
      { name: 'logo.svg', reason: 'type' },
      { name: 'big.pdf', reason: 'size' },
      { name: 'c.pdf', reason: 'count' },
    ])
  })
})

describe('formatBytes', () => {
  it('uses the largest fitting unit', () => {
    expect(formatBytes(512, 'en')).toBe('512 B')
    expect(formatBytes(1536, 'en')).toBe('1.5 KB')
    expect(formatBytes(10 * 1024 * 1024, 'de')).toBe('10 MB')
    expect(formatBytes(1.25 * 1024 * 1024, 'de')).toBe('1,3 MB')
  })
})
