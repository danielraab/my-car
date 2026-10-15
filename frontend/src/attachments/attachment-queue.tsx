import { X } from 'lucide-react'
import { useState } from 'react'
import { useTranslation } from 'react-i18next'

import type { AttachmentOwner } from '#/api/client'

import { FileList } from './attachment-list'
import { FilePicker } from './file-picker'
import { uploadQueuedFiles } from './upload'

export type AttachmentQueueState = ReturnType<typeof useAttachmentQueue>

// Files picked on a create form, kept until the entry exists. upload sends
// them for the new entry and resolves to the names of those that failed.
export function useAttachmentQueue() {
  const [files, setFiles] = useState<File[]>([])
  const [progress, setProgress] = useState<{
    done: number
    total: number
  } | null>(null)
  return {
    files,
    setFiles,
    progress,
    upload: async (owner: AttachmentOwner) => {
      if (files.length === 0) return []
      try {
        return await uploadQueuedFiles(owner, files, (done, total) =>
          setProgress({ done, total }),
        )
      } finally {
        setProgress(null)
      }
    },
  }
}

// The attachment part of a create form: picked files are listed and can be
// removed, and are uploaded once the form is saved.
export function AttachmentQueue({
  queue,
  disabled,
}: {
  queue: AttachmentQueueState
  disabled: boolean
}) {
  const { t } = useTranslation()
  const { files, setFiles, progress } = queue
  return (
    <fieldset className="attachment-section attachment-section-inline">
      <legend>{t('attachments.title')}</legend>
      {files.length > 0 ? (
        <>
          <FileList
            label={t('attachments.queueLabel')}
            items={files.map((file, index) => ({
              key: `${file.name}-${file.size}-${index}`,
              name: file.name,
              contentType: file.type,
              size: file.size,
              actions: (
                <button
                  className="icon-button"
                  type="button"
                  disabled={disabled}
                  aria-label={t('attachments.remove', { name: file.name })}
                  onClick={() =>
                    setFiles((current) => current.filter((f) => f !== file))
                  }
                >
                  <X aria-hidden="true" size={16} />
                </button>
              ),
            }))}
          />
          <p className="field-hint">{t('attachments.queuedHint')}</p>
        </>
      ) : null}
      {progress ? (
        <output className="attachment-progress">
          {t('attachments.uploadProgress', progress)}
        </output>
      ) : null}
      <FilePicker
        existingCount={files.length}
        disabled={disabled}
        onFiles={(picked) => setFiles((current) => [...current, ...picked])}
      />
    </fieldset>
  )
}
