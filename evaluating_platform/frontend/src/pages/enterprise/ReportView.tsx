import { useEffect, useState } from 'react'
import { Card, Tag, Typography, Progress, Divider, Button, Row, Col, Timeline, Spin, message } from 'antd'
import { DownloadOutlined, ShareAltOutlined } from '@ant-design/icons'
import { useParams } from 'react-router-dom'
import { reportService, triggerBrowserDownload, type Report } from '../../services/report'

const { Title, Text, Paragraph } = Typography

const severityConfig: Record<string, { color: string; label: string }> = {
  critical: { color: '#ff4d4f', label: '严重' },
  high: { color: '#ff7a45', label: '高危' },
  medium: { color: '#ffc53d', label: '中危' },
  low: { color: '#52c41a', label: '低危' },
  info: { color: '#40a9ff', label: '信息' },
}

const riskLabel: Record<string, string> = {
  critical: '严重风险', high: '高风险', medium: '中等风险', low: '低风险', info: '信息',
}

export function ReportView() {
  const { id } = useParams<{ id: string }>()
  const [report, setReport] = useState<Report | null>(null)
  const [loading, setLoading] = useState(true)
  const [downloading, setDownloading] = useState(false)

  useEffect(() => {
    if (!id) return
    reportService.get(id).catch(() => reportService.getByAssessmentID(id)).then(r => {
      setReport(r)
    }).catch(() => {}).finally(() => setLoading(false))
  }, [id])

  if (loading) return <div style={{ textAlign: 'center', padding: 80 }}><Spin size="large" /></div>
  if (!report) return <div style={{ color: 'var(--text-secondary)', textAlign: 'center', padding: 80 }}>报告不存在或加载失败</div>

  const findings = Array.isArray(report.findings) ? report.findings : []
  const score = typeof report.metrics?.risk_score === 'number' ? report.metrics.risk_score : 50
  const riskColor = severityConfig[report.risk_level]?.color || '#4d96ff'
  const handleDownloadPDF = async () => {
    setDownloading(true)
    try {
      const blob = await reportService.downloadPDF(report.id)
      triggerBrowserDownload(blob, `report-${report.assessment_id || report.id}.pdf`)
    } catch {
      message.error('PDF 下载失败，请稍后重试')
    } finally {
      setDownloading(false)
    }
  }

  return (
    <div style={{ maxWidth: 960, margin: '0 auto' }}>
      <Card style={{
        background: 'linear-gradient(135deg, #0f1117 0%, #141820 100%)',
        border: '1px solid var(--border-color)',
        marginBottom: 20,
      }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
          <div>
            <div style={{ marginBottom: 8, display: 'flex', gap: 8, alignItems: 'center' }}>
              <Tag color={riskColor} style={{ fontWeight: 600 }}>{riskLabel[report.risk_level] || '未知'}</Tag>
              <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>{report.created_at?.slice(0, 10)}</Text>
            </div>
            <Title level={4} style={{ color: 'var(--text-primary)', margin: '0 0 8px 0' }}>{report.title}</Title>
            <div style={{ display: 'flex', gap: 24 }}>
              <div><Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>发现问题</Text>
                <div><Text style={{ color: '#ff7a45', fontWeight: 600 }}>{findings.length}</Text></div></div>
            </div>
          </div>
          <div style={{ textAlign: 'center' }}>
            <div style={{ fontSize: 48, fontWeight: 800, color: riskColor, lineHeight: 1 }}>{score}</div>
            <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>安全评分 /100</Text>
            <Progress percent={score} size="small" showInfo={false} strokeColor={riskColor}
              trailColor='var(--border-color)' style={{ width: 120, marginTop: 4 }} />
          </div>
        </div>

        <Divider style={{ borderColor: 'var(--border-color)', margin: '16px 0' }} />

        <Paragraph style={{ color: 'var(--text-secondary)', margin: 0, lineHeight: 1.8 }}>
          {report.summary}
        </Paragraph>

        <div style={{ marginTop: 16, display: 'flex', gap: 10 }}>
          <Button type="primary" icon={<DownloadOutlined />} loading={downloading}
            onClick={handleDownloadPDF}>
            下载 PDF 报告
          </Button>
          <Button icon={<ShareAltOutlined />} style={{ color: 'var(--text-secondary)' }}
            onClick={() => reportService.download(report.id, 'html')}>
            导出 HTML
          </Button>
        </div>
      </Card>

      <Row gutter={16}>
        <Col span={findings.length > 0 ? 16 : 24}>
          <Title level={5} style={{ color: 'var(--text-primary)' }}>发现详情</Title>
          {findings.length === 0 && (
            <Card style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}>
              <Text style={{ color: 'var(--text-secondary)' }}>暂无发现的安全问题</Text>
            </Card>
          )}
          {findings.map((f, idx) => (
            <Card key={f.id || idx} style={{
              background: 'var(--bg-card)',
              border: `1px solid ${severityConfig[f.severity]?.color || 'var(--border-color)'}40`,
              marginBottom: 12,
            }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', marginBottom: 10 }}>
                <div style={{ display: 'flex', gap: 8, alignItems: 'center' }}>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>{f.id}</Text>
                  <Text style={{ color: 'var(--text-primary)', fontWeight: 600 }}>{f.title}</Text>
                  <Tag color={severityConfig[f.severity]?.color}>{severityConfig[f.severity]?.label}</Tag>
                </div>
                <Tag style={{ color: 'var(--text-muted)', background: 'var(--bg-surface)', border: '1px solid var(--border-color)' }}>
                  {f.tool}
                </Tag>
              </div>
              <Text style={{ color: 'var(--text-secondary)', fontSize: 13, display: 'block', marginBottom: 8 }}>
                {f.description}
              </Text>
              {f.evidence && (
                <div style={{
                  padding: '8px 12px', background: 'var(--bg-surface)',
                  border: '1px solid var(--border-color)', borderRadius: 4,
                  fontFamily: 'monospace', fontSize: 12, color: '#ffc53d', marginBottom: 10,
                }}>
                  证据：{f.evidence}
                </div>
              )}
              {f.suggestion && (
                <div>
                  <Text style={{ color: 'var(--text-muted)', fontSize: 12 }}>修复建议：</Text>
                  <Text style={{ color: '#52c41a', fontSize: 12 }}>{f.suggestion}</Text>
                </div>
              )}
            </Card>
          ))}
        </Col>

        {findings.length > 0 && (
          <Col span={8}>
            <Title level={5} style={{ color: 'var(--text-primary)' }}>风险概况</Title>
            <Card style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}>
              <Timeline
                items={findings.map(f => ({
                  color: severityConfig[f.severity]?.color || '#8890a4',
                  children: (
                    <div>
                      <Text style={{ color: 'var(--text-primary)', fontSize: 13, fontWeight: 500 }}>
                        {f.title}
                      </Text>
                      <br />
                      <Text style={{ color: 'var(--text-secondary)', fontSize: 12 }}>
                        {severityConfig[f.severity]?.label} · {f.tool}
                      </Text>
                    </div>
                  ),
                }))}
              />
            </Card>
          </Col>
        )}
      </Row>
    </div>
  )
}
