import { QueryClientProvider } from '@tanstack/react-query'
import { fireEvent, screen, waitFor, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import type { ReactNode } from 'react'
import { beforeEach, describe, expect, it, vi } from 'vitest'

import type { Attachment, AttachmentOwner } from '#/api/client'
import { i18n } from '#/i18n'
import { createQueryClient } from '#/router'
import { renderInRouter } from '#/test/render-in-router'

import { AttachmentIndicator } from './attachment-indicator'
import { AttachmentManager } from './attachment-manager'
import { AttachmentQueue, useAttachmentQueue } from './attachment-queue'
import { AttachmentViewer } from './attachment-viewer'

// A stand-in for pdf.js: counts opened documents and serves one with the
// given number of pages, or fails.
const pdfjs = vi.hoisted(() => ({
  opened: 0,
  pages: 2 as number | Error,
}))
vi.mock('./pdf', () => {
  return {
    getDocument: () => {
      pdfjs.opened++
      return {
        promise:
          pdfjs.pages instanceof Error
            ? Promise.reject(pdfjs.pages)
            : Promise.resolve({
                numPages: pdfjs.pages,
                getPage: async () => ({
                  getViewport: ({ scale }: { scale: number }) => ({
                    width: 600 * scale,
                    height: 840 * scale,
                  }),
                  render: () => ({ promise: Promise.resolve(), cancel() {} }),
                }),
              }),
        destroy: async () => {},
      }
    },
  }
})

const owner: AttachmentOwner = {
  kind: 'refuels',
  id: '33333333-3333-3333-3333-333333333333',
}
const receipt: Attachment = {
  id: 'aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa',
  filename: 'receipt.jpg',
  contentType: 'image/jpeg',
  size: 412 * 1024,
  createdAt: '2026-10-15T08:00:00Z',
}
const invoice: Attachment = {
  id: 'bbbbbbbb-bbbb-4bbb-8bbb-bbbbbbbbbbbb',
  filename: 'invoice.pdf',
  contentType: 'application/pdf',
  size: 1200 * 1024,
  createdAt: '2026-10-15T09:00:00Z',
}
const limits = {
  maxFileBytes: 10 * 1024 * 1024,
  maxFilesPerEntry: 10,
  allowedContentTypes: [
    'image/jpeg',
    'image/png',
    'image/webp',
    'image/gif',
    'application/pdf',
  ],
}
const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { 'Content-Type': 'application/json' },
  })
const pdf = (name: string, size = 10) =>
  new File(['%'.repeat(size)], name, { type: 'application/pdf' })

// A fake backend holding the entry's attachments; uploads named in reject
// are refused with that error code.
function backend(
  initial: Attachment[] = [],
  reject: Record<string, [number, string]> = {},
) {
  let stored = [...initial]
  const calls: string[] = []
  vi.stubGlobal(
    'fetch',
    vi.fn(async (path: string, init?: RequestInit) => {
      const call = `${init?.method ?? 'GET'} ${path}`
      calls.push(call)
      const list = `/api/v1/${owner.kind}/${owner.id}/attachments`
      if (path === '/api/v1/attachments/limits') return json(limits)
      if (path.endsWith('/content')) return new Response('%PDF-1.7')
      if (call === `GET ${list}`)
        return json({ items: stored, nextCursor: null })
      if (call === `POST ${list}`) {
        const file = (init?.body as FormData | undefined)?.get('file') as File
        const refusal = reject[file.name]
        if (refusal) {
          return json({ code: refusal[1], message: 'refused' }, refusal[0])
        }
        const created: Attachment = {
          id: `cccccccc-cccc-4ccc-8ccc-${String(stored.length).padStart(12, '0')}`,
          filename: file.name,
          contentType: 'application/pdf',
          size: file.size,
          createdAt: '2026-10-15T10:00:00Z',
        }
        stored = [...stored, created]
        return json(created, 201)
      }
      if (call.startsWith('DELETE /api/v1/attachments/')) {
        const id = path.split('/').at(-1)
        stored = stored.filter((a) => a.id !== id)
        return new Response(null, { status: 204 })
      }
      throw new Error(call)
    }),
  )
  return calls
}

function render(ui: ReactNode) {
  return renderInRouter(
    <QueryClientProvider client={createQueryClient()}>
      {ui}
    </QueryClientProvider>,
  )
}

beforeEach(async () => {
  await i18n.changeLanguage('en')
  vi.unstubAllGlobals()
  pdfjs.pages = 2
  pdfjs.opened = 0
})

describe('AttachmentQueue', () => {
  function Harness() {
    const queue = useAttachmentQueue()
    return <AttachmentQueue queue={queue} disabled={false} />
  }

  it('shows the downscaling hint, queues accepted files and names refused ones', async () => {
    backend()
    const user = userEvent.setup({ applyAccept: false })
    await render(<Harness />)

    expect(
      await screen.findByText(
        'Images larger than 2000 px are scaled down before upload. Allowed: JPG, PNG, WebP, GIF and PDF, up to 10 MB per file and 10 files per entry.',
      ),
    ).toBeInTheDocument()

    await user.upload(screen.getByLabelText('Add files'), [
      pdf('invoice.pdf'),
      new File(['x'], 'notes.docx', { type: 'application/msword' }),
      pdf('huge.pdf', 10 * 1024 * 1024 + 1),
    ])

    const queued = await screen.findByRole('list', { name: 'Files to upload' })
    expect(within(queued).getByText('invoice.pdf')).toBeInTheDocument()
    expect(within(queued).queryByText('notes.docx')).not.toBeInTheDocument()
    expect(
      screen.getByText(
        'notes.docx: only images (JPG, PNG, WebP, GIF) and PDF documents can be added.',
      ),
    ).toBeInTheDocument()
    expect(
      screen.getByText('huge.pdf: the file is too large (max. 10 MB).'),
    ).toBeInTheDocument()
    expect(
      screen.getByText('The files are uploaded when you save.'),
    ).toBeInTheDocument()

    expect(
      within(queued).queryByRole('button', { name: 'invoice.pdf' }),
    ).not.toBeInTheDocument()

    await user.click(screen.getByRole('button', { name: 'Remove invoice.pdf' }))
    expect(
      screen.queryByRole('list', { name: 'Files to upload' }),
    ).not.toBeInTheDocument()
  })

  it('refuses files above the per-entry maximum', async () => {
    backend()
    const user = userEvent.setup()
    await render(<Harness />)
    const files = Array.from({ length: 11 }, (_, i) => pdf(`page-${i + 1}.pdf`))
    await user.upload(await screen.findByLabelText('Add files'), files)
    expect(
      within(
        await screen.findByRole('list', { name: 'Files to upload' }),
      ).getAllByRole('listitem'),
    ).toHaveLength(10)
    expect(
      screen.getByText('page-11.pdf: an entry can hold at most 10 files.'),
    ).toBeInTheDocument()
  })
})

describe('AttachmentManager', () => {
  it('lists files with type icons and opens them in a read-only viewer', async () => {
    backend([receipt, invoice])
    const user = userEvent.setup()
    await render(<AttachmentManager owner={owner} />)

    const list = await screen.findByRole('list', { name: 'Attached files' })
    const [first, second] = within(list).getAllByRole('listitem')
    expect(within(first).getByText('Image')).toBeInTheDocument()
    expect(within(second).getByText('PDF document')).toBeInTheDocument()
    expect(within(first).getByText('412 KB')).toBeInTheDocument()
    expect(document.querySelector('img')).toBeNull()
    expect(screen.queryByRole('link')).not.toBeInTheDocument()

    await user.click(within(first).getByRole('button', { name: 'receipt.jpg' }))
    const viewer = await screen.findByRole('dialog', { name: 'receipt.jpg' })
    expect(within(viewer).getByText('1 / 2')).toBeInTheDocument()
    expect(
      within(viewer).getByRole('link', { name: 'Download' }),
    ).toHaveAttribute('download', 'receipt.jpg')
    expect(
      within(viewer).getByRole('link', { name: 'Open in new tab' }),
    ).toHaveAttribute('target', '_blank')
    expect(
      within(viewer).queryByRole('button', { name: 'Delete file' }),
    ).not.toBeInTheDocument()
  })

  it('uploads picked files at once and reports server refusals', async () => {
    const calls = backend([], {
      'too-many.pdf': [409, 'attachment_limit_reached'],
    })
    const user = userEvent.setup()
    await render(<AttachmentManager owner={owner} />)
    expect(await screen.findByText('No attachments.')).toBeInTheDocument()

    await user.upload(screen.getByLabelText('Add files'), [
      pdf('invoice.pdf'),
      pdf('too-many.pdf'),
    ])

    const list = await screen.findByRole('list', { name: 'Attached files' })
    expect(within(list).getByText('invoice.pdf')).toBeInTheDocument()
    expect(
      await screen.findByText(
        'too-many.pdf: an entry can hold at most 10 files.',
      ),
    ).toBeInTheDocument()
    expect(calls.filter((call) => call.startsWith('POST '))).toHaveLength(2)
  })

  it('deletes a file only after the second confirmation', async () => {
    const calls = backend([receipt])
    const user = userEvent.setup()
    await render(<AttachmentManager owner={owner} />)
    const list = await screen.findByRole('list', { name: 'Attached files' })

    await user.click(within(list).getByRole('button', { name: 'Delete file' }))
    expect(calls.some((call) => call.startsWith('DELETE'))).toBe(false)
    expect(
      screen.getByText('Delete receipt.jpg? This cannot be undone.'),
    ).toBeInTheDocument()
    await user.click(screen.getByRole('button', { name: 'Yes, delete file' }))

    expect(await screen.findByText('No attachments.')).toBeInTheDocument()
    expect(calls).toContain(`DELETE /api/v1/attachments/${receipt.id}`)
  })
})

describe('AttachmentIndicator', () => {
  it('renders nothing without attachments', async () => {
    backend()
    await render(<AttachmentIndicator owner={owner} count={0} />)
    expect(screen.queryByRole('button')).not.toBeInTheDocument()
  })

  it('loads and lists the files only when opened', async () => {
    const calls = backend([receipt, invoice])
    const user = userEvent.setup()
    await render(<AttachmentIndicator owner={owner} count={2} />)
    const button = screen.getByRole('button', { name: '2 attachments' })
    expect(calls).toHaveLength(0)

    await user.click(button)
    const list = await screen.findByRole('list', { name: 'Attached files' })
    expect(
      within(list).getByRole('button', { name: 'invoice.pdf' }),
    ).toBeInTheDocument()
    await waitFor(() => expect(calls).toHaveLength(1))
  })

  it('closes the popover, opens the viewer and returns focus to the paperclip', async () => {
    backend([receipt, invoice])
    const user = userEvent.setup()
    await render(<AttachmentIndicator owner={owner} count={2} />)
    const button = screen.getByRole('button', { name: '2 attachments' })

    await user.click(button)
    await user.click(await screen.findByRole('button', { name: 'invoice.pdf' }))

    const viewer = await screen.findByRole('dialog', { name: 'invoice.pdf' })
    expect(within(viewer).getByText('2 / 2')).toBeInTheDocument()
    expect(
      screen.queryByRole('list', { name: 'Attached files' }),
    ).not.toBeInTheDocument()

    await user.keyboard('{Escape}')
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    )
    await waitFor(() => expect(button).toHaveFocus())
  })

  it('is labelled in German', async () => {
    backend()
    await i18n.changeLanguage('de')
    await render(<AttachmentIndicator owner={owner} count={1} />)
    expect(screen.getByRole('button', { name: '1 Anhang' })).toBeInTheDocument()
  })
})

describe('AttachmentViewer', () => {
  const photo: Attachment = {
    ...receipt,
    id: 'dddddddd-dddd-4ddd-8ddd-dddddddddddd',
    filename: 'odometer.png',
    contentType: 'image/png',
  }

  it('steps with the arrow keys and buttons without wrapping around', async () => {
    backend()
    const user = userEvent.setup()
    await render(
      <AttachmentViewer
        attachments={[receipt, photo, invoice]}
        startIndex={0}
        onClose={() => {}}
      />,
    )
    const previous = screen.getByRole('button', { name: 'Previous file' })
    const next = screen.getByRole('button', { name: 'Next file' })
    expect(screen.getByText('1 / 3')).toBeInTheDocument()
    expect(previous).toBeDisabled()

    await user.keyboard('{ArrowLeft}')
    expect(screen.getByText('1 / 3')).toBeInTheDocument()
    await user.keyboard('{ArrowRight}')
    expect(
      screen.getByRole('dialog', { name: 'odometer.png' }),
    ).toBeInTheDocument()
    await user.click(next)
    expect(screen.getByText('3 / 3')).toBeInTheDocument()
    expect(next).toBeDisabled()
    await user.keyboard('{ArrowRight}')
    expect(screen.getByText('3 / 3')).toBeInTheDocument()
    await user.click(previous)
    expect(screen.getByText('2 / 3')).toBeInTheDocument()
  })

  it('swipes to the next file', async () => {
    backend()
    await render(
      <AttachmentViewer
        attachments={[receipt, photo]}
        startIndex={0}
        onClose={() => {}}
      />,
    )
    const slide = screen.getByAltText('receipt.jpg').parentElement as Element
    fireEvent.pointerDown(slide, { clientX: 300, clientY: 200 })
    fireEvent.pointerUp(slide, { clientX: 200, clientY: 210 })
    expect(screen.getByText('2 / 2')).toBeInTheDocument()
    // Mostly vertical movement is scrolling, not a swipe.
    fireEvent.pointerDown(slide, { clientX: 100, clientY: 400 })
    fireEvent.pointerUp(slide, { clientX: 170, clientY: 100 })
    expect(screen.getByText('2 / 2')).toBeInTheDocument()
  })

  it('disables both buttons for a single file and closes on Escape', async () => {
    backend()
    const onClose = vi.fn()
    const user = userEvent.setup()
    await render(
      <AttachmentViewer
        attachments={[receipt]}
        startIndex={0}
        onClose={onClose}
      />,
    )
    expect(screen.getByText('1 / 1')).toBeInTheDocument()
    expect(screen.getByRole('button', { name: 'Previous file' })).toBeDisabled()
    expect(screen.getByRole('button', { name: 'Next file' })).toBeDisabled()
    await user.keyboard('{Escape}')
    expect(onClose).toHaveBeenCalled()
  })

  it('offers to open an image in a new tab when it cannot be shown', async () => {
    backend()
    await render(
      <AttachmentViewer
        attachments={[receipt]}
        startIndex={0}
        onClose={() => {}}
      />,
    )
    expect(screen.getByText('Loading file…')).toBeInTheDocument()
    fireEvent.error(screen.getByAltText('receipt.jpg'))
    expect(
      await screen.findByText('The image could not be shown.'),
    ).toBeInTheDocument()
    const fallback = screen.getByRole('alert')
    expect(
      within(fallback).getByRole('link', { name: 'Open in new tab' }),
    ).toHaveAttribute('href', `/api/v1/attachments/${receipt.id}/content`)
  })

  it('renders every page of a PDF, opening it only once it is shown', async () => {
    backend()
    const user = userEvent.setup()
    await render(
      <AttachmentViewer
        attachments={[receipt, invoice]}
        startIndex={0}
        onClose={() => {}}
      />,
    )
    expect(pdfjs.opened).toBe(0)

    await user.keyboard('{ArrowRight}')
    expect(
      await screen.findByRole('img', { name: 'Page 1 of 2' }),
    ).toBeInTheDocument()
    expect(screen.getByRole('img', { name: 'Page 2 of 2' })).toBeInTheDocument()
    expect(pdfjs.opened).toBe(1)
  })

  it('offers to open a PDF in a new tab when it cannot be rendered', async () => {
    backend()
    pdfjs.pages = new Error('broken')
    await render(
      <AttachmentViewer
        attachments={[invoice]}
        startIndex={0}
        onClose={() => {}}
      />,
    )
    expect(
      await screen.findByText('The PDF could not be shown.'),
    ).toBeInTheDocument()
    expect(
      within(screen.getByRole('alert')).getByRole('link', {
        name: 'Open in new tab',
      }),
    ).toBeInTheDocument()
  })

  it('is labelled in German', async () => {
    backend()
    await i18n.changeLanguage('de')
    await render(
      <AttachmentViewer
        attachments={[receipt, invoice]}
        startIndex={0}
        onClose={() => {}}
      />,
    )
    for (const name of [
      'Vorherige Datei',
      'Nächste Datei',
      'Ansicht schließen',
    ]) {
      expect(screen.getByRole('button', { name })).toBeInTheDocument()
    }
    for (const name of ['In neuem Tab öffnen', 'Herunterladen']) {
      expect(screen.getByRole('link', { name })).toBeInTheDocument()
    }
    expect(screen.getByText('Datei wird geladen…')).toBeInTheDocument()
  })
})
