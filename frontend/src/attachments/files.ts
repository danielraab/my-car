import type { AttachmentLimits } from '#/api/client'

import { downscaleImage } from './downscale'

// upload: the server refused or the upload failed.
export type RefusalReason = 'type' | 'size' | 'count' | 'upload'
export type Refusal = { name: string; reason: RefusalReason }

// Downscales picked files and sorts out those the server would refuse: a
// type outside the allowed ones, a size above the limit after downscaling,
// or more files than the entry may still take. The server checks again; this
// only spares the user a pointless upload.
export async function prepareFiles(
  picked: File[],
  limits: AttachmentLimits,
  existingCount: number,
): Promise<{ accepted: File[]; refused: Refusal[] }> {
  const accepted: File[] = []
  const refused: Refusal[] = []
  for (const original of picked) {
    if (!limits.allowedContentTypes.some((type) => type === original.type)) {
      refused.push({ name: original.name, reason: 'type' })
      continue
    }
    if (existingCount + accepted.length >= limits.maxFilesPerEntry) {
      refused.push({ name: original.name, reason: 'count' })
      continue
    }
    const file = await downscaleImage(original)
    if (file.size > limits.maxFileBytes) {
      refused.push({ name: original.name, reason: 'size' })
      continue
    }
    accepted.push(file)
  }
  return { accepted, refused }
}

export function isImage(contentType: string): boolean {
  return contentType.startsWith('image/')
}

// A byte count as B, KB or MB with one decimal, in the given locale.
export function formatBytes(bytes: number, locale: string): string {
  const units = ['B', 'KB', 'MB', 'GB']
  let value = bytes
  let unit = 0
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024
    unit++
  }
  const formatted = new Intl.NumberFormat(locale, {
    maximumFractionDigits: unit === 0 ? 0 : 1,
  }).format(value)
  return `${formatted} ${units[unit]}`
}
