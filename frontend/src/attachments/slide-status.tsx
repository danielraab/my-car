import { ExternalLink, LoaderCircle } from 'lucide-react'
import { useTranslation } from 'react-i18next'

export function SlideLoading() {
  const { t } = useTranslation()
  return (
    <output className="viewer-status">
      <LoaderCircle className="spin" aria-hidden="true" size={20} />
      {t('attachments.viewer.loading')}
    </output>
  )
}

// Shown when a file can't be displayed in the viewer; the browser may still
// manage to open it on its own.
export function SlideFallback({
  message,
  url,
}: {
  message: string
  url: string
}) {
  const { t } = useTranslation()
  return (
    <div className="viewer-status" role="alert">
      <p>{message}</p>
      <a
        className="button button-secondary button-compact"
        href={url}
        target="_blank"
        rel="noopener noreferrer"
      >
        <ExternalLink aria-hidden="true" size={16} />
        {t('attachments.viewer.openInTab')}
      </a>
    </div>
  )
}
