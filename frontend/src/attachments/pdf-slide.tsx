import { useCallback, useEffect, useRef, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { type Attachment, attachmentContentUrl } from '#/api/client'

import type { PDFDocumentProxy } from './pdf'
import { SlideFallback, SlideLoading } from './slide-status'

// A PDF rendered by pdf.js into one canvas per page, independent of the
// browser's own PDF viewer (absent on Android, blocked by the downloads'
// sandbox in Chromium). The bytes are fetched as data, so the protective
// download headers stay as they are.
export function PdfSlide({ attachment }: { attachment: Attachment }) {
  const { t } = useTranslation()
  const [doc, setDoc] = useState<PDFDocumentProxy>()
  const [failed, setFailed] = useState(false)
  const fail = useCallback(() => setFailed(true), [])
  const url = attachmentContentUrl(attachment.id)

  useEffect(() => {
    let cancelled = false
    let destroy: (() => Promise<void>) | undefined
    ;(async () => {
      try {
        const { getDocument } = await import('./pdf')
        const response = await fetch(url, { credentials: 'same-origin' })
        if (!response.ok) throw new Error(`status ${response.status}`)
        const data = new Uint8Array(await response.arrayBuffer())
        if (cancelled) return
        const task = getDocument({ data })
        destroy = () => task.destroy()
        const loaded = await task.promise
        if (!cancelled) setDoc(loaded)
      } catch {
        if (!cancelled) setFailed(true)
      }
    })()
    return () => {
      cancelled = true
      void destroy?.()
    }
  }, [url])

  if (failed) {
    return (
      <SlideFallback message={t('attachments.viewer.pdfError')} url={url} />
    )
  }
  if (doc === undefined) return <SlideLoading />
  return (
    <div className="viewer-pdf">
      {Array.from({ length: doc.numPages }, (_, i) => (
        <PdfPage
          // biome-ignore lint/suspicious/noArrayIndexKey: pages are fixed
          key={i}
          doc={doc}
          pageNumber={i + 1}
          onError={fail}
        />
      ))}
    </div>
  )
}

// One page, drawn when it first scrolls into view, at the slide's width.
function PdfPage({
  doc,
  pageNumber,
  onError,
}: {
  doc: PDFDocumentProxy
  pageNumber: number
  onError: () => void
}) {
  const { t } = useTranslation()
  const canvasRef = useRef<HTMLCanvasElement>(null)
  const [visible, setVisible] = useState(false)
  const [aspect, setAspect] = useState<number>()

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || visible) return
    if (typeof IntersectionObserver === 'undefined') {
      setVisible(true)
      return
    }
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries.some((entry) => entry.isIntersecting)) setVisible(true)
      },
      { rootMargin: '200px 0px' },
    )
    observer.observe(canvas)
    return () => observer.disconnect()
  }, [visible])

  useEffect(() => {
    const canvas = canvasRef.current
    if (!canvas || !visible) return
    let cancelled = false
    let cancelRender: (() => void) | undefined
    ;(async () => {
      try {
        const page = await doc.getPage(pageNumber)
        if (cancelled) return
        const natural = page.getViewport({ scale: 1 })
        const width = canvas.parentElement?.clientWidth || natural.width
        const ratio = window.devicePixelRatio || 1
        const viewport = page.getViewport({
          scale: (width / natural.width) * ratio,
        })
        canvas.width = Math.floor(viewport.width)
        canvas.height = Math.floor(viewport.height)
        setAspect(natural.width / natural.height)
        const task = page.render({ canvas, viewport })
        cancelRender = () => task.cancel()
        await task.promise
      } catch (error) {
        if (!cancelled && !isCancellation(error)) onError()
      }
    })()
    return () => {
      cancelled = true
      cancelRender?.()
    }
  }, [doc, pageNumber, visible, onError])

  return (
    <canvas
      ref={canvasRef}
      className="viewer-pdf-page"
      style={aspect ? { aspectRatio: aspect } : undefined}
      role="img"
      aria-label={t('attachments.viewer.page', {
        page: pageNumber,
        total: doc.numPages,
      })}
    />
  )
}

function isCancellation(error: unknown): boolean {
  return error instanceof Error && error.name === 'RenderingCancelledException'
}
