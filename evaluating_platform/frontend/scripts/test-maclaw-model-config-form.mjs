import assert from 'node:assert/strict'
import fs from 'node:fs'
import path from 'node:path'
import vm from 'node:vm'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const ts = require('typescript')

function loadTSModule(relativePath) {
  const filename = path.resolve(relativePath)
  const source = fs.readFileSync(filename, 'utf8')
  const compiled = ts.transpileModule(source, {
    compilerOptions: {
      module: ts.ModuleKind.CommonJS,
      target: ts.ScriptTarget.ES2022,
      esModuleInterop: true,
    },
    fileName: filename,
  }).outputText
  const module = { exports: {} }
  const sandbox = { module, exports: module.exports, require, console }
  vm.runInNewContext(compiled, sandbox, { filename })
  return module.exports
}

const { configToForm, formToConfig } = loadTSModule('src/pages/admin/maclawModelConfigForm.ts')

const deepseekConfig = formToConfig({
  provider_name: '',
  url: 'https://api.deepseek.com',
  key: 'sk-test',
  model: 'deepseek-v4',
  wire_api: 'responses',
  timeout_sec: 60,
})

assert.equal(deepseekConfig.maclaw_llm_current_provider, 'deepseek-prod')
assert.equal(deepseekConfig.maclaw_llm_providers[0].name, 'deepseek-prod')
assert.equal(deepseekConfig.maclaw_llm_providers[0].wire_api, 'chat_completions')
assert.equal(deepseekConfig.maclaw_llm_providers[0].model, 'deepseek-v4-flash')

const form = configToForm({
  maclaw_llm_current_provider: 'deepseek',
  maclaw_llm_providers: [{
    name: 'deepseek',
    url: 'https://api.deepseek.com',
    model: 'deepseek-v4',
    wire_api: 'responses',
  }],
})

assert.equal(form.provider_name, 'deepseek')
assert.equal(form.wire_api, 'chat_completions')
assert.equal(form.model, 'deepseek-v4-flash')

console.log('maclaw model config form tests passed')
