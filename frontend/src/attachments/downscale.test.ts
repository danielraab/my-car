import { afterEach, describe, expect, it, vi } from 'vitest'

import { downscaleImage, JPEG_QUALITY, scaledSize } from './downscale'

describe('scaledSize', () => {
  it('scales the longer edge to the maximum, keeping the aspect ratio', () => {
    expect(scaledSize(4000, 3000)).toEqual({ width: 2000, height: 1500 })
    expect(scaledSize(3000, 4000)).toEqual({ width: 1500, height: 2000 })
  })

  it('leaves images within the maximum alone', () => {
    expect(scaledSize(2000, 1200)).toBeNull()
    expect(scaledSize(1200, 800)).toBeNull()
  })
})

describe('downscaleImage', () => {
  afterEach(() => {
    vi.unstubAllGlobals()
    vi.restoreAllMocks()
  })

  function stubImage(width: number, height: number, encodedType: string) {
    const close = vi.fn()
    const decode = vi.fn().mockResolvedValue({ width, height, close })
    vi.stubGlobal('createImageBitmap', decode)
    const drawImage = vi.fn()
    vi.spyOn(HTMLCanvasElement.prototype, 'getContext').mockReturnValue({
      drawImage,
    } as unknown as CanvasRenderingContext2D)
    const toBlob = vi
      .spyOn(HTMLCanvasElement.prototype, 'toBlob')
      .mockImplementation((callback) =>
        callback(new Blob(['small'], { type: encodedType })),
      )
    return { decode, drawImage, toBlob, close }
  }

  it('scales a large JPEG down in its own format', async () => {
    const { decode, drawImage, toBlob, close } = stubImage(
      4000,
      3000,
      'image/jpeg',
    )
    const original = new File(['x'.repeat(100)], 'receipt.jpg', {
      type: 'image/jpeg',
    })

    const result = await downscaleImage(original)

    expect(decode).toHaveBeenCalledWith(original, {
      imageOrientation: 'from-image',
    })
    expect(drawImage).toHaveBeenCalledWith(expect.anything(), 0, 0, 2000, 1500)
    expect(toBlob).toHaveBeenCalledWith(
      expect.any(Function),
      'image/jpeg',
      JPEG_QUALITY,
    )
    expect(result).not.toBe(original)
    expect(result.name).toBe('receipt.jpg')
    expect(result.type).toBe('image/jpeg')
    expect(result.size).toBe(5)
    expect(close).toHaveBeenCalled()
  })

  it('keeps PNGs lossless', async () => {
    const { toBlob } = stubImage(3000, 3000, 'image/png')
    await downscaleImage(new File(['x'], 'scan.png', { type: 'image/png' }))
    expect(toBlob).toHaveBeenCalledWith(
      expect.any(Function),
      'image/png',
      undefined,
    )
  })

  it('returns small images, GIFs and PDFs unchanged', async () => {
    const { decode } = stubImage(1200, 800, 'image/png')
    const small = new File(['x'], 'small.png', { type: 'image/png' })
    expect(await downscaleImage(small)).toBe(small)

    decode.mockClear()
    for (const file of [
      new File(['x'], 'anim.gif', { type: 'image/gif' }),
      new File(['x'], 'doc.pdf', { type: 'application/pdf' }),
    ]) {
      expect(await downscaleImage(file)).toBe(file)
    }
    expect(decode).not.toHaveBeenCalled()
  })

  it('keeps the original when the format cannot be re-encoded', async () => {
    stubImage(4000, 3000, 'image/png')
    const webp = new File(['x'], 'photo.webp', { type: 'image/webp' })
    expect(await downscaleImage(webp)).toBe(webp)
  })

  it('keeps the original when the image cannot be decoded', async () => {
    vi.stubGlobal(
      'createImageBitmap',
      vi.fn().mockRejectedValue(new Error('decode failed')),
    )
    const broken = new File(['x'], 'broken.jpg', { type: 'image/jpeg' })
    expect(await downscaleImage(broken)).toBe(broken)
  })
})
