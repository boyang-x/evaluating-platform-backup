package workflow

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/client"

	"evaluating_platform/internal/agent"
	"evaluating_platform/internal/model"
)

// Executor 工作流 DAG 执行引擎
// 按拓扑顺序依次通过 MCP Client 调用各工具节点，并将上游输出注入下游参数
type Executor struct {
	mcpClient *client.Client
}

// NewExecutor 创建工作流执行引擎
func NewExecutor(mcpClient *client.Client) *Executor {
	return &Executor{mcpClient: mcpClient}
}

// Run 按 DAG 拓扑顺序执行工作流中的所有工具节点，返回与 agent.Engine.Run 相同类型的结果。
//
// 数据流规则：
//   - 每个节点执行后，其 JSON 输出存入 nodeOutputs[nodeID]
//   - 后继节点执行前，其所有直接前驱的输出以 upstream_{predecessorID} 为键注入 params
//   - 节点自身的 Params 优先级最高，可覆盖全局参数和上游注入
func (e *Executor) Run(ctx context.Context, assessmentID string, cfg model.WorkflowConfig) (*agent.RunResult, error) {
	if len(cfg.Nodes) == 0 {
		return &agent.RunResult{Summary: "工作流无节点，跳过执行"}, nil
	}

	order, err := topoSort(cfg)
	if err != nil {
		return nil, fmt.Errorf("workflow DAG 排序失败: %w", err)
	}

	// 构建前驱映射：predecessors[nodeID] = []直接前驱 nodeID
	predecessors := make(map[string][]string, len(cfg.Nodes))
	for _, edge := range cfg.Edges {
		predecessors[edge.Target] = append(predecessors[edge.Target], edge.Source)
	}

	exec := agent.NewExecutor(e.mcpClient, assessmentID)
	result := &agent.RunResult{}
	nodeOutputs := make(map[string]string) // nodeID → JSON 输出

	for _, nodeID := range order {
		node := findNode(cfg.Nodes, nodeID)
		if node == nil {
			continue
		}

		// 参数优先级（低→高）：全局参数 < 上游输出注入 < 节点自身参数
		params := make(map[string]interface{})
		for k, v := range cfg.Params {
			params[k] = v
		}
		// 注入直接前驱的输出
		for _, predID := range predecessors[nodeID] {
			if output, ok := nodeOutputs[predID]; ok {
				var parsed interface{}
				if json.Unmarshal([]byte(output), &parsed) == nil {
					params["upstream_"+predID] = parsed
				}
			}
		}
		for k, v := range node.Params {
			params[k] = v
		}

		execResult := exec.Execute(ctx, node.ToolName, params)

		tr := agent.ToolResult{
			ToolName: node.ToolName,
			Input:    execResult.InputJSON,
			Severity: execResult.Severity,
		}
		if execResult.Error != nil {
			tr.Output = fmt.Sprintf(`{"error": "%s"}`, execResult.Error.Error())
		} else {
			tr.Output = execResult.OutputJSON
			// 保存输出供下游节点使用
			if execResult.OutputJSON != "" {
				nodeOutputs[nodeID] = execResult.OutputJSON
			}
		}

		result.Logs = append(result.Logs, tr)
	}

	result.Summary = fmt.Sprintf("工作流执行完成，共执行 %d 个节点", len(result.Logs))
	return result, nil
}

// topoSort 对工作流 DAG 进行拓扑排序（Kahn 算法），返回节点 ID 的执行顺序。
// 若图中存在环则返回错误。
func topoSort(cfg model.WorkflowConfig) ([]string, error) {
	inDegree := make(map[string]int, len(cfg.Nodes))
	adj := make(map[string][]string, len(cfg.Nodes))

	for _, node := range cfg.Nodes {
		if _, ok := inDegree[node.ID]; !ok {
			inDegree[node.ID] = 0
			adj[node.ID] = nil
		}
	}
	for _, edge := range cfg.Edges {
		adj[edge.Source] = append(adj[edge.Source], edge.Target)
		inDegree[edge.Target]++
	}

	queue := make([]string, 0, len(cfg.Nodes))
	for _, node := range cfg.Nodes {
		if inDegree[node.ID] == 0 {
			queue = append(queue, node.ID)
		}
	}

	sorted := make([]string, 0, len(cfg.Nodes))
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		sorted = append(sorted, curr)

		for _, next := range adj[curr] {
			inDegree[next]--
			if inDegree[next] == 0 {
				queue = append(queue, next)
			}
		}
	}

	if len(sorted) != len(cfg.Nodes) {
		return nil, fmt.Errorf("工作流 DAG 中存在环，无法执行")
	}
	return sorted, nil
}

// findNode 在节点列表中按 ID 查找节点
func findNode(nodes []model.WorkflowNode, id string) *model.WorkflowNode {
	for i := range nodes {
		if nodes[i].ID == id {
			return &nodes[i]
		}
	}
	return nil
}
