import { FileImage, FileText } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

import { attachmentContentUrl } from '#/api/client'

import { formatBytes, isImage } from './files'

// An image or document icon, never a preview: listing files loads no file
// contents.
export function FileTypeIcon({ contentType }: { contentType: string }) {
  const { t } = useTranslation()
  const image = isImage(contentType)
  const Icon = image ? FileImage : FileText
  return (
    <span className="attachment-type">
      <Icon aria-hidden="true" size={18} />
      <span className="sr-only">
        {t(image ? 'attachments.types.image' : 'attachments.types.document')}
      </span>
    </span>
  )
}

type Listed = {
  key: string
  name: string
  contentType: string
  size: number
  // The stored attachment to open in a new tab; absent for a queued file.
  attachmentId?: string
  actions?: ReactNode
}

// Attachments or queued files as icon, name, size and optional actions.
export function FileList({ label, items }: { label: string; items: Listed[] }) {
  const { i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'
  return (
    <ul className="attachment-list" aria-label={label}>
      {items.map((item) => (
        <li key={item.key} className="attachment-item">
          <FileTypeIcon contentType={item.contentType} />
          <OpenLink name={item.name} attachmentId={item.attachmentId} />
          <span className="attachment-size">
            {formatBytes(item.size, locale)}
          </span>
          {item.actions ? (
            <span className="attachment-actions">{item.actions}</span>
          ) : null}
        </li>
      ))}
    </ul>
  )
}

function OpenLink({
  name,
  attachmentId,
}: {
  name: string
  attachmentId?: string
}) {
  const { t } = useTranslation()
  if (attachmentId === undefined) {
    return <span className="attachment-name">{name}</span>
  }
  return (
    <a
      className="attachment-name table-link"
      href={attachmentContentUrl(attachmentId)}
      target="_blank"
      rel="noopener noreferrer"
      title={t('attachments.open', { name })}
    >
      {name}
    </a>
  )
}
