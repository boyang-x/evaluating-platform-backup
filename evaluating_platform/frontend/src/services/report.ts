import api from './api'

export interface Finding {
  id: string
  title: string
  severity: string
  category: string
  tool: string
  description: string
  evidence: string
  suggestion: string
}

export interface Report {
  id: string
  assessment_id: string
  title: string
  summary: string
  risk_level: string
  findings: Finding[]
  metrics: {
    totalTests?: number
    failedTests?: number
    passedTests?: number
    durationSecs?: number
    risk_score?: number
    [key: string]: unknown
  }
  raw_content: string
  pdf_url: string
  html_url: string
  created_at: string
}

export const reportService = {
  async get(id: string): Promise<Report> {
    const res = await api.get<Report>(`/reports/${id}`)
    return res.data
  },

  async getByAssessmentID(assessmentID: string): Promise<Report> {
    const res = await api.get<Report>(`/reports/${assessmentID}`)
    return res.data
  },

  async download(id: string, format: 'pdf' | 'html' = 'pdf'): Promise<unknown> {
    const res = await api.get(`/reports/${id}/download`, { params: { format } })
    return res.data
  },

  async downloadPDF(assessmentId: string): Promise<Blob> {
    const res = await api.get(`/reports/${assessmentId}/download`, {
      params: { format: 'pdf' },
      responseType: 'blob',
    })
    return res.data
  },
}
