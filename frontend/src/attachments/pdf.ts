// pdf.js and its worker, imported only through a dynamic import() so that
// they form their own chunks and are fetched when a PDF is first shown.
import { GlobalWorkerOptions, getDocument } from 'pdfjs-dist'
import workerUrl from 'pdfjs-dist/build/pdf.worker.min.mjs?url'

GlobalWorkerOptions.workerSrc = workerUrl

export type { PDFDocumentProxy } from 'pdfjs-dist'
export { getDocument }
