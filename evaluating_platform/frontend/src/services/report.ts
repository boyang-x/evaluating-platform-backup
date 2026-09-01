import api from './api'

export const reportService = {
  async downloadMaclawReport(reportId: string, format: 'pdf' | 'markdown' | 'json' = 'pdf'): Promise<Blob> {
    const res = await api.get(`/maclaw/evaluation/reports/${encodeURIComponent(reportId)}/export`, {
      params: { format },
      responseType: 'blob',
    })
    return res.data
  },
}

export function triggerBrowserDownload(blob: Blob, filename: string, fallbackType = 'application/pdf') {
  const typedBlob = blob.type ? blob : new Blob([blob], { type: fallbackType })
  const url = window.URL.createObjectURL(typedBlob)
  const link = document.createElement('a')
  link.href = url
  link.download = filename
  link.rel = 'noopener'
  document.body.appendChild(link)
  link.click()
  document.body.removeChild(link)
  window.setTimeout(() => window.URL.revokeObjectURL(url), 60_000)
}
