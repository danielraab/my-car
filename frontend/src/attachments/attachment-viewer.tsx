import { Dialog, DialogPanel, DialogTitle } from '@headlessui/react'
import {
  ChevronLeft,
  ChevronRight,
  Download,
  ExternalLink,
  X,
} from 'lucide-react'
import {
  type PointerEvent,
  type ReactNode,
  type RefObject,
  useCallback,
  useEffect,
  useRef,
  useState,
} from 'react'
import { useTranslation } from 'react-i18next'

import { type Attachment, attachmentContentUrl } from '#/api/client'

import { isImage } from './files'
import { ImageSlide } from './image-slide'
import { PdfSlide } from './pdf-slide'

// Horizontal travel, in CSS pixels, that counts as a swipe.
const swipeDistance = 50

type Opened = { attachments: Attachment[]; index: number }

// The viewer's state for a list of attachments. The list is taken as it was
// when a file was opened, so the order stays put while the viewer is open.
export function useAttachmentViewer(
  returnFocus?: RefObject<HTMLElement | null>,
): {
  open: (attachments: Attachment[], index: number) => void
  viewer: ReactNode
} {
  const [opened, setOpened] = useState<Opened>()
  const open = useCallback(
    (attachments: Attachment[], index: number) =>
      setOpened({ attachments, index }),
    [],
  )
  const close = useCallback(() => {
    setOpened(undefined)
    // The control that opened the viewer may be gone, such as a file in a
    // popover that closed on opening; focus then goes to the given element.
    const target = returnFocus?.current
    if (target) setTimeout(() => target.focus())
  }, [returnFocus])
  const viewer = opened ? (
    <AttachmentViewer
      attachments={opened.attachments}
      startIndex={opened.index}
      onClose={close}
    />
  ) : null
  return { open, viewer }
}

// A modal that shows one of an entry's attachments at a time. It steps
// through them with buttons, the arrow keys and swipes, without wrapping
// around, and offers only read-only actions.
export function AttachmentViewer({
  attachments,
  startIndex,
  onClose,
}: {
  attachments: Attachment[]
  startIndex: number
  onClose: () => void
}) {
  const { t } = useTranslation()
  const [index, setIndex] = useState(startIndex)
  const swipeStart = useRef<{ x: number; y: number }>(undefined)
  const current = attachments[index]
  const total = attachments.length
  const hasPrevious = index > 0
  const hasNext = index < total - 1
  const previous = () => setIndex((i) => Math.max(i - 1, 0))
  const next = () => setIndex((i) => Math.min(i + 1, total - 1))

  useEffect(() => {
    for (const neighbour of [attachments[index - 1], attachments[index + 1]]) {
      if (neighbour && isImage(neighbour.contentType)) {
        new Image().src = attachmentContentUrl(neighbour.id)
      }
    }
  }, [attachments, index])

  // On the window, so the keys work wherever focus is while the viewer is
  // open, including before the dialog has moved focus into itself.
  useEffect(() => {
    const onKeyDown = (event: globalThis.KeyboardEvent) => {
      if (event.key === 'ArrowLeft') setIndex((i) => Math.max(i - 1, 0))
      else if (event.key === 'ArrowRight')
        setIndex((i) => Math.min(i + 1, total - 1))
      else return
      event.preventDefault()
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [total])

  const onPointerDown = (event: PointerEvent) => {
    swipeStart.current = { x: event.clientX, y: event.clientY }
  }
  const onPointerUp = (event: PointerEvent) => {
    const start = swipeStart.current
    swipeStart.current = undefined
    if (!start) return
    const dx = event.clientX - start.x
    const dy = event.clientY - start.y
    if (Math.abs(dx) < swipeDistance || Math.abs(dx) < 2 * Math.abs(dy)) return
    if (dx < 0) next()
    else previous()
  }

  if (current === undefined) return null
  const url = attachmentContentUrl(current.id)
  return (
    <Dialog className="viewer-dialog" open onClose={onClose}>
      <div className="viewer-backdrop" aria-hidden="true" />
      <DialogPanel className="viewer-panel">
        <header className="viewer-header">
          <DialogTitle className="viewer-title">{current.filename}</DialogTitle>
          <div className="viewer-actions">
            <a
              className="icon-button"
              href={url}
              target="_blank"
              rel="noopener noreferrer"
              aria-label={t('attachments.viewer.openInTab')}
              title={t('attachments.viewer.openInTab')}
            >
              <ExternalLink aria-hidden="true" size={18} />
            </a>
            <a
              className="icon-button"
              href={url}
              download={current.filename}
              aria-label={t('attachments.viewer.download')}
              title={t('attachments.viewer.download')}
            >
              <Download aria-hidden="true" size={18} />
            </a>
            <button
              className="icon-button"
              type="button"
              aria-label={t('attachments.viewer.close')}
              title={t('attachments.viewer.close')}
              onClick={onClose}
            >
              <X aria-hidden="true" size={18} />
            </button>
          </div>
        </header>
        <div
          className="viewer-slide"
          onPointerDown={onPointerDown}
          onPointerUp={onPointerUp}
          onPointerCancel={() => {
            swipeStart.current = undefined
          }}
        >
          {isImage(current.contentType) ? (
            <ImageSlide key={current.id} attachment={current} />
          ) : (
            <PdfSlide key={current.id} attachment={current} />
          )}
        </div>
        <footer className="viewer-footer">
          <button
            className="icon-button"
            type="button"
            aria-label={t('attachments.viewer.previous')}
            disabled={!hasPrevious}
            onClick={previous}
          >
            <ChevronLeft aria-hidden="true" size={20} />
          </button>
          <span className="viewer-position">
            {t('attachments.viewer.position', {
              current: index + 1,
              total,
            })}
          </span>
          <button
            className="icon-button"
            type="button"
            aria-label={t('attachments.viewer.next')}
            disabled={!hasNext}
            onClick={next}
          >
            <ChevronRight aria-hidden="true" size={20} />
          </button>
        </footer>
      </DialogPanel>
    </Dialog>
  )
}
