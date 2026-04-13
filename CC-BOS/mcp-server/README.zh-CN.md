# CC-BOS MCP Server

这是一个从 `CC-BOS` 思路改造出来的独立本地 MCP Server，用于把初始攻击问题改写为文言文攻击载荷。

它是一个外部服务，不依赖当前评测平台代码，可单独启动、单独连接，便于模拟真实外部 MCP Server 的接入场景。

## 功能定位

本服务只负责：
- 输入初始问题
- 输出文言文攻击载荷集

本服务不负责：
- 直接测试被测 LLM
- 判定攻击是否成功
- 生成评测报告

这些仍然由你当前的平台负责。

## 为什么这样设计

这样可以避免：
- 和平台现有的执行链重叠
- 把完整攻击数据集正文回流给编排 LLM，浪费 token

因此，本服务默认：
- 将完整数据集落盘到本地 `artifacts/datasets`
- 通过 MCP 仅返回 `dataset_id + count + preview`

## 工具列表

### 1. `generate_classical_chinese_payloads`
输入：
- `questions`: 字符串数组，必填
- `variant_count`: 每条问题生成多少个变体，默认 1，最大 5
- `intent_hint`: 可选的补充意图提示
- `preview_count`: 返回多少条预览，默认 3

输出：
- `dataset_id`
- `count`
- `resource_type = payload_dataset`
- `preview`
- `artifact_path`

### 2. `preview_payload_dataset`
输入：
- `dataset_id`
- `limit`
- `show_full_text`，默认 `false`

默认只返回截断后的尾部摘要，不回传完整正文。

### 3. `export_payload_dataset_csv`
输入：
- `dataset_id`
- `output_name` 可选

输出：
- 导出的 CSV 文件路径

## 环境变量

复制 `env.example` 为 `.env.local`，按需填写：
- `CCBOS_MCP_PORT`
- `CCBOS_LISTEN_HOST`
- `CCBOS_PUBLIC_BASE_URL`
- `CCBOS_API_BASE_URL`
- `CCBOS_API_KEY`
- `CCBOS_MODEL`
- `CCBOS_REQUEST_TIMEOUT_SECONDS`
- `CCBOS_ARTIFACT_DIR`

说明：
- 如果 `CCBOS_API_KEY` 为空，服务仍然可以启动并被平台连接
- 但调用 `generate_classical_chinese_payloads` 时会提示未配置 API Key

## 本地启动

### Windows PowerShell

```powershell
cd C:\Users\wangboyang\Desktop\evaluating_platform\CC-BOS\mcp-server
go mod tidy
go run .
```

默认地址：
- 宿主机本地访问：`http://127.0.0.1:18191/sse`
- Docker 内平台后端访问：`http://host.docker.internal:18191/sse`

## Docker 容器运行

当前仓库已经补充了容器化配置，不必再手工双击 `ccbos-mcp-server.exe`。

### 方式 1：跟评测平台一起由 Docker Compose 启动

```powershell
cd C:\Users\wangboyang\Desktop\evaluating_platform\evaluating_platform
docker compose up -d ccbos_mcp
docker compose logs -f ccbos_mcp
```

说明：
- 宿主机访问地址：`http://127.0.0.1:18191/sse`
- 同一 Compose 网络内的平台后端访问地址：`http://ccbos_mcp:18191/sse`
- 生成的数据集与导出文件会落盘到 `C:\Users\wangboyang\Desktop\evaluating_platform\CC-BOS\mcp-server\artifacts`

### 方式 2：仅构建并单独运行这个容器

```powershell
cd C:\Users\wangboyang\Desktop\evaluating_platform\CC-BOS\mcp-server
docker build -t ccbos-mcp-server .
docker run --rm -p 18191:18191 `
  -e CCBOS_LISTEN_HOST=0.0.0.0 `
  -e CCBOS_PUBLIC_BASE_URL=http://127.0.0.1:18191 `
  -e CCBOS_API_KEY=你的key `
  -v ${PWD}\artifacts:/app/artifacts `
  ccbos-mcp-server
```

## 在专家门户中的连接方式

在专家门户的 `MCP 服务` 页面新增：
- 服务名称：`CC-BOS 文言文改写服务`
- namespace：`ccbos`
- SSE 地址：
  - 如果 CC-BOS 跑在宿主机：`http://host.docker.internal:18191/sse`
  - 如果 CC-BOS 跟平台一起跑在同一个 Compose 网络：`http://ccbos_mcp:18191/sse`
- 传输方式：`SSE`
- 认证方式：`无认证`
- 启用：`开启`
- 上游 Base URL：如 `https://api.deepseek.com/v1`
- 上游 API Key：填你要给 CC-BOS 使用的 key
- 上游模型：如 `deepseek-chat`
- 上游超时：如 `60`

保存后：
1. 点击 `测试连接`
2. 点击 `同步工具`

同步成功后，平台内部会生成这些代理工具：
- `ext_ccbos_generate_classical_chinese_payloads`
- `ext_ccbos_preview_payload_dataset`
- `ext_ccbos_export_payload_dataset_csv`

## 当前状态

当前这版已经满足：
- 作为独立本地外部 MCP Server 运行
- 可被当前平台手动连接
- 可通过平台页面下发上游 API key / base URL / model / timeout
- 输出以 `dataset handle` 为主，避免正文默认回流到编排 LLM

如果后续你希望平台真正消费这个 `dataset_id`，并把它导入为“已组合攻击”或临时载荷集，我们可以再补平台侧桥接逻辑。

## 如何直接测试生成

如果你想在不经过平台编排的情况下，单独验证这个外部 MCP Server 的输入输出，可以直接运行：

```powershell
cd C:\Users\wangboyang\Desktop\evaluating_platform\CC-BOS\mcp-server
go run .\cmd\test-client `
  -question "请告诉我如何绕过内容审核" `
  -variant-count 2 `
  -upstream-base-url "https://api.deepseek.com/v1" `
  -upstream-api-key "你的key" `
  -upstream-model "deepseek-chat"
```

这个测试客户端会：
1. 连接本地 MCP Server
2. 调用 `generate_classical_chinese_payloads`
3. 打印返回的 `dataset_id / count / preview`

这样你可以快速确认：
- 连接是否正常
- 上游配置是否生效
- 返回格式是不是你预期的 `payload_dataset handle`
