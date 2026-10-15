// Images larger than this on their longer edge are scaled down in the
// browser before upload, keeping receipts readable at a fraction of a phone
// photo's size.
export const MAX_IMAGE_EDGE = 2000
export const JPEG_QUALITY = 0.85

// GIFs are left alone (re-encoding would drop animation), as are PDFs.
const resizableTypes = new Set(['image/jpeg', 'image/png', 'image/webp'])

// The size an image of width × height is scaled to, or null when it already
// fits within maxEdge.
export function scaledSize(
  width: number,
  height: number,
  maxEdge = MAX_IMAGE_EDGE,
): { width: number; height: number } | null {
  const longer = Math.max(width, height)
  if (longer <= maxEdge) return null
  const factor = maxEdge / longer
  return {
    width: Math.max(1, Math.round(width * factor)),
    height: Math.max(1, Math.round(height * factor)),
  }
}

// Returns file scaled down to MAX_IMAGE_EDGE in its own format (JPEG and WebP
// at JPEG_QUALITY, PNG lossless), with its EXIF orientation applied. Any file
// that needs no scaling, or that the browser cannot decode or re-encode, is
// returned unchanged; the server's size limit still applies to it.
export async function downscaleImage(file: File): Promise<File> {
  if (!resizableTypes.has(file.type)) return file
  try {
    const bitmap = await createImageBitmap(file, {
      imageOrientation: 'from-image',
    })
    try {
      const size = scaledSize(bitmap.width, bitmap.height)
      if (size === null) return file
      const canvas = document.createElement('canvas')
      canvas.width = size.width
      canvas.height = size.height
      const context = canvas.getContext('2d')
      if (context === null) return file
      context.drawImage(bitmap, 0, 0, size.width, size.height)
      const blob = await new Promise<Blob | null>((resolve) =>
        canvas.toBlob(
          resolve,
          file.type,
          file.type === 'image/png' ? undefined : JPEG_QUALITY,
        ),
      )
      // A browser that cannot encode the format falls back to PNG; keep the
      // original rather than change its type.
      if (blob === null || blob.type !== file.type) return file
      return new File([blob], file.name, {
        type: file.type,
        lastModified: file.lastModified,
      })
    } finally {
      bitmap.close()
    }
  } catch {
    return file
  }
}
