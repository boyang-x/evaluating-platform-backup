import { useState, useCallback, useEffect, useRef } from 'react'
import { Button, Typography, message, Input, Space, Drawer, Form, Collapse, Tag } from 'antd'
import {
  SaveOutlined, SendOutlined, PlusOutlined,
  ApiOutlined, ExperimentOutlined, FileProtectOutlined, InboxOutlined,
} from '@ant-design/icons'
import ReactFlow, {
  Background, Controls, MiniMap,
  addEdge, useNodesState, useEdgesState,
  type Node, type Edge, type Connection,
  BackgroundVariant,
} from 'reactflow'
import 'reactflow/dist/style.css'
import { useSearchParams } from 'react-router-dom'
import { assetService, type Asset } from '../../services/asset'
import { expertService } from '../../services/expert'

const { Title, Text } = Typography

const categoryColor: Record<string, string> = {
  app_detector: '#52c41a',
  sample_tool: '#4d96ff',
  template_tool: '#722ed1',
  combo_tool: '#ff7a45',
}

const categoryIcon: Record<string, string> = {
  app_detector: '🔍', sample_tool: '🧪', template_tool: '📝', combo_tool: '📦',
}

interface ToolItem {
  id: string
  name: string
  category: string
  subType?: string
  meta?: any
}

// 将 WorkflowConfig 节点映射到 ReactFlow 节点
function toRFNode(wn: any, idx: number): Node {
  return {
    id: wn.id,
    type: 'default',
    position: { x: wn.position?.x ?? idx * 200, y: wn.position?.y ?? 100 },
    data: {
      label: wn.label || wn.tool_name,
      toolName: wn.tool_name,
      toolId: wn.tool_id || '',
      toolCategory: wn.tool_category || '',
      labelStr: wn.label,
      params: wn.params,
    },
    style: {
      background: 'rgba(26,109,255,0.12)',
      border: '1px solid rgba(26,109,255,0.5)',
      borderRadius: 8, color: '#fff', minWidth: 150, padding: '8px 12px',
    },
  }
}

function toRFEdge(we: any): Edge {
  return {
    id: we.id, source: we.source, target: we.target,
    style: { stroke: 'rgba(26,109,255,0.5)', strokeWidth: 2 }, animated: true,
  }
}

export function WorkflowEditor() {
  const [searchParams] = useSearchParams()
  const assetId = searchParams.get('id')

  const [nodes, setNodes, onNodesChange] = useNodesState([])
  const [edges, setEdges, onEdgesChange] = useEdgesState([])
  const [asset, setAsset] = useState<Asset | null>(null)
  const [assetName, setAssetName] = useState('新工作流')
  const [saving, setSaving] = useState(false)
  const [propDrawerOpen, setPropDrawerOpen] = useState(false)
  const [selectedNode, setSelectedNode] = useState<Node | null>(null)
  const [propForm] = Form.useForm()
  const idCounter = useRef(100)

  // 动态工具列表
  const [detectors, setDetectors] = useState<ToolItem[]>([])
  const [samples, setSamples] = useState<ToolItem[]>([])
  const [templates, setTemplates] = useState<ToolItem[]>([])
  const [combos, setCombos] = useState<ToolItem[]>([])

  useEffect(() => {
    // 加载四类工具
    Promise.all([
      expertService.listDetectors({ limit: 50 }).catch(() => ({ items: [] })),
      expertService.listSamples({ limit: 50 }).catch(() => ({ items: [] })),
      expertService.listTemplates({ limit: 50 }).catch(() => ({ items: [] })),
      expertService.listEvalPackages({ limit: 50 }).catch(() => ({ items: [] })),
    ]).then(([d, s, t, p]) => {
      setDetectors((d.items || []).map((i: any) => ({ id: i.id, name: i.name, category: 'app_detector', subType: i.sub_type, meta: i })))
      setSamples((s.items || []).map((i: any) => ({ id: i.id, name: i.name, category: 'sample_tool', subType: i.sub_type, meta: i })))
      setTemplates((t.items || []).map((i: any) => ({ id: i.id, name: i.name, category: 'template_tool', subType: i.sub_type, meta: i })))
      setCombos((p.items || []).map((i: any) => ({ id: i.id, name: i.name, category: 'combo_tool', meta: i })))
    })
  }, [])

  // 加载已有工作流
  useEffect(() => {
    if (!assetId) return
    assetService.get(assetId).then(a => {
      setAsset(a)
      setAssetName(a.name)
      const rfNodes = (a.config?.nodes || []).map((wn, i) => toRFNode(wn, i))
      const rfEdges = (a.config?.edges || []).map(toRFEdge)
      setNodes(rfNodes)
      setEdges(rfEdges)
    }).catch(() => message.error('加载工作流失败'))
  }, [assetId])

  const onConnect = useCallback(
    (params: Connection) => setEdges(eds => addEdge({
      ...params, style: { stroke: 'rgba(26,109,255,0.5)', strokeWidth: 2 }, animated: true,
    }, eds)),
    [setEdges],
  )

  const addToolNode = (tool: ToolItem) => {
    const id = `node_${idCounter.current++}`
    const color = categoryColor[tool.category] || '#4d96ff'
    const icon = categoryIcon[tool.category] || '🔧'
    const newNode: Node = {
      id,
      type: 'default',
      position: { x: 100 + nodes.length * 200, y: 100 + Math.random() * 80 },
      data: {
        label: (
          <div style={{ textAlign: 'left', padding: '4px 0' }}>
            <Text style={{ color: '#fff', fontWeight: 600, fontSize: 13 }}>{icon} {tool.name}</Text>
            <br />
            <Tag color={color} style={{ fontSize: 10, marginTop: 2 }}>{tool.subType || tool.category}</Tag>
          </div>
        ),
        toolId: tool.id,
        toolName: tool.name,
        toolCategory: tool.category,
        toolSubType: tool.subType,
        labelStr: tool.name,
        params: {},
      },
      style: {
        background: `${color}20`,
        border: `1px solid ${color}80`,
        borderRadius: 8, color: '#fff', minWidth: 160, padding: '8px 12px',
      },
    }
    setNodes(nds => [...nds, newNode])
  }

  const handleNodeClick = (_: React.MouseEvent, node: Node) => {
    setSelectedNode(node)
    propForm.setFieldsValue({
      label: node.data.labelStr || node.data.toolName,
      toolName: node.data.toolName,
      toolCategory: node.data.toolCategory,
    })
    setPropDrawerOpen(true)
  }

  const handleSaveProp = () => {
    if (!selectedNode) return
    const values = propForm.getFieldsValue()
    const color = categoryColor[selectedNode.data.toolCategory] || '#4d96ff'
    const icon = categoryIcon[selectedNode.data.toolCategory] || '🔧'
    setNodes(nds => nds.map(n => n.id === selectedNode.id ? {
      ...n,
      data: {
        ...n.data,
        labelStr: values.label,
        label: (
          <div style={{ textAlign: 'left', padding: '4px 0' }}>
            <Text style={{ color: '#fff', fontWeight: 600, fontSize: 13 }}>{icon} {values.label}</Text>
            <br />
            <Tag color={color} style={{ fontSize: 10, marginTop: 2 }}>{n.data.toolSubType || n.data.toolCategory}</Tag>
          </div>
        ),
      },
    } : n))
    setPropDrawerOpen(false)
  }

  const toWorkflowConfig = (ns: Node[], es: Edge[]) => ({
    nodes: ns.map(n => ({
      id: n.id,
      tool_name: n.data.toolName || n.id,
      tool_id: n.data.toolId || '',
      tool_category: n.data.toolCategory || '',
      label: n.data.labelStr || n.id,
      params: n.data.params || {},
      position: n.position,
    })),
    edges: es.map(e => ({ id: e.id, source: e.source, target: e.target })),
  })

  const handleSave = async () => {
    const config = toWorkflowConfig(nodes, edges)
    setSaving(true)
    try {
      if (asset) {
        await assetService.update(asset.id, { name: assetName, config })
        message.success('工作流已保存')
      } else {
        const res = await assetService.create({ name: assetName, type: 'workflow', visibility: 'private', config })
        message.success(`工作流已创建 (ID: ${res.asset_id})`)
      }
    } catch (err: any) {
      message.error(err.message || '保存失败')
    } finally {
      setSaving(false)
    }
  }

  const handleSubmit = async () => {
    if (!asset) { message.warning('请先保存工作流再提交审核'); return }
    try {
      await assetService.submitForReview(asset.id)
      message.success('已提交审核')
    } catch (err: any) {
      message.error(err.message || '提交失败')
    }
  }

  const renderToolList = (items: ToolItem[], emptyText: string) => (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 4 }}>
      {items.length === 0 ? (
        <Text style={{ color: 'var(--text-muted)', fontSize: 11 }}>{emptyText}</Text>
      ) : items.map(tool => (
        <Button key={tool.id} size="small" icon={<PlusOutlined />}
          onClick={() => addToolNode(tool)}
          style={{ textAlign: 'left', justifyContent: 'flex-start', fontSize: 11, height: 'auto', padding: '4px 8px', whiteSpace: 'normal' }}>
          {tool.name}
        </Button>
      ))}
    </div>
  )

  return (
    <div>
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: 16 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 12 }}>
          <Title level={4} style={{ color: 'var(--text-primary)', margin: 0 }}>工作流编排</Title>
          <Input value={assetName} onChange={e => setAssetName(e.target.value)}
            style={{ width: 200, background: 'var(--bg-surface)', color: 'var(--text-primary)' }} />
        </div>
        <Space>
          <Button icon={<SaveOutlined />} loading={saving} onClick={handleSave}>保存</Button>
          <Button type="primary" icon={<SendOutlined />} onClick={handleSubmit}>提交审核发布</Button>
        </Space>
      </div>

      <div style={{ display: 'flex', gap: 16 }}>
        {/* 左侧工具面板 — 四分组折叠 */}
        <div style={{ width: 200, flexShrink: 0 }}>
          <Collapse defaultActiveKey={['app_detector', 'sample_tool']} size="small"
            style={{ background: 'var(--bg-card)', border: '1px solid var(--border-color)' }}
            items={[
              {
                key: 'app_detector',
                label: <span><ApiOutlined style={{ color: '#52c41a' }} /> 应用检测工具</span>,
                children: renderToolList(detectors, '暂无检测工具'),
              },
              {
                key: 'sample_tool',
                label: <span><ExperimentOutlined style={{ color: '#4d96ff' }} /> 样本工具</span>,
                children: renderToolList(samples, '暂无样本'),
              },
              {
                key: 'template_tool',
                label: <span><FileProtectOutlined style={{ color: '#722ed1' }} /> 模版工具</span>,
                children: renderToolList(templates, '暂无模版'),
              },
              {
                key: 'combo_tool',
                label: <span><InboxOutlined style={{ color: '#ff7a45' }} /> 组合工具</span>,
                children: renderToolList(combos, '暂无评测包'),
              },
            ]}
          />
        </div>

        {/* React Flow 画布 */}
        <div style={{
          flex: 1, height: 520, background: '#0a0c10',
          border: '1px solid var(--border-color)', borderRadius: 8, overflow: 'hidden',
        }}>
          <ReactFlow
            nodes={nodes} edges={edges}
            onNodesChange={onNodesChange} onEdgesChange={onEdgesChange}
            onConnect={onConnect} onNodeClick={handleNodeClick}
            fitView proOptions={{ hideAttribution: true }}
          >
            <Background variant={BackgroundVariant.Dots} color="rgba(26,109,255,0.15)" gap={20} />
            <Controls />
            <MiniMap style={{ background: 'rgba(0,0,0,0.5)' }} nodeColor="rgba(26,109,255,0.5)" />
          </ReactFlow>
        </div>
      </div>

      {/* 节点属性面板 */}
      <Drawer title="节点属性" width={320} open={propDrawerOpen}
        onClose={() => setPropDrawerOpen(false)}
        footer={<Button type="primary" onClick={handleSaveProp}>保存属性</Button>}>
        <Form form={propForm} layout="vertical">
          <Form.Item name="label" label="节点标签"><Input placeholder="显示名称" /></Form.Item>
          <Form.Item name="toolName" label="工具名称"><Input disabled /></Form.Item>
          <Form.Item name="toolCategory" label="工具类别"><Input disabled /></Form.Item>
        </Form>
      </Drawer>
    </div>
  )
}
