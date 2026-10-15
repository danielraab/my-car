import { FileImage, FileText } from 'lucide-react'
import type { ReactNode } from 'react'
import { useTranslation } from 'react-i18next'

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
  actions?: ReactNode
}

// Attachments or queued files as icon, name, size and optional actions. With
// onOpen, each name is a button reporting its item's index; queued files are
// listed without it.
export function FileList({
  label,
  items,
  onOpen,
}: {
  label: string
  items: Listed[]
  onOpen?: (index: number) => void
}) {
  const { i18n } = useTranslation()
  const locale = i18n.resolvedLanguage ?? 'en'
  return (
    <ul className="attachment-list" aria-label={label}>
      {items.map((item, index) => (
        <li key={item.key} className="attachment-item">
          <FileTypeIcon contentType={item.contentType} />
          {onOpen ? (
            <button
              className="attachment-name attachment-open table-link"
              type="button"
              onClick={() => onOpen(index)}
            >
              {item.name}
            </button>
          ) : (
            <span className="attachment-name">{item.name}</span>
          )}
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
