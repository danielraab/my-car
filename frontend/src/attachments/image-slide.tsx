import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import { type Attachment, attachmentContentUrl } from '#/api/client'

import { SlideFallback, SlideLoading } from './slide-status'

// An image fitted inside the slide, never enlarged beyond its natural size.
export function ImageSlide({ attachment }: { attachment: Attachment }) {
  const { t } = useTranslation()
  const [status, setStatus] = useState<'loading' | 'loaded' | 'failed'>(
    'loading',
  )
  const url = attachmentContentUrl(attachment.id)
  if (status === 'failed') {
    return (
      <SlideFallback message={t('attachments.viewer.imageError')} url={url} />
    )
  }
  return (
    <>
      {status === 'loading' ? <SlideLoading /> : null}
      <img
        className="viewer-image"
        src={url}
        alt={attachment.filename}
        hidden={status === 'loading'}
        onLoad={() => setStatus('loaded')}
        onError={() => setStatus('failed')}
      />
    </>
  )
}
