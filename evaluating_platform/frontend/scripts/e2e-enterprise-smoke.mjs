import { spawn } from 'node:child_process'
import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'

const chrome = process.env.CHROME_PATH || 'C:/Program Files/Google/Chrome/Application/chrome.exe'
const appURL = process.env.EP_APP_URL || 'http://127.0.0.1:5173'
const apiURL = process.env.EP_API_URL || 'http://127.0.0.1:8080/api/v1'
const email = process.env.EP_E2E_EMAIL || 'enterprise@demo.com'
const password = process.env.EP_E2E_PASSWORD || ''
const port = Number(process.env.EP_CDP_PORT || 9224)
const waitLoops = Number(process.env.EP_E2E_WAIT_LOOPS || 90)

if (!password) throw new Error('EP_E2E_PASSWORD is required')

const userData = fs.mkdtempSync(path.join(os.tmpdir(), 'ep-chrome-'))
const browserProcess = spawn(chrome, [
  '--headless=new',
  `--remote-debugging-port=${port}`,
  `--user-data-dir=${userData}`,
  '--disable-gpu',
  '--no-first-run',
  '--no-default-browser-check',
  'about:blank',
], { stdio: 'ignore' })

async function getJSON(url, options) {
  const response = await fetch(url, options)
  if (!response.ok) throw new Error(`${url} returned ${response.status}`)
  return response.json()
}

async function waitForCDP() {
  for (let i = 0; i < 80; i += 1) {
    try {
      return await getJSON(`http://127.0.0.1:${port}/json/version`)
    } catch {
      await new Promise(resolve => setTimeout(resolve, 250))
    }
  }
  throw new Error('Chrome DevTools endpoint did not become ready')
}

let nextID = 0
const pending = new Map()

function callCDP(ws, method, params = {}) {
  return new Promise((resolve, reject) => {
    const id = ++nextID
    pending.set(id, { resolve, reject })
    ws.send(JSON.stringify({ id, method, params }))
  })
}

async function evalPage(ws, expression) {
  const result = await callCDP(ws, 'Runtime.evaluate', {
    expression,
    returnByValue: true,
    awaitPromise: true,
  })
  if (result.exceptionDetails) throw new Error(JSON.stringify(result.exceptionDetails))
  return result.result?.value
}

async function bodyText(ws) {
  return await evalPage(ws, 'document.body.innerText')
}

async function waitFor(ws, predicate, loops = 40, interval = 500) {
  for (let i = 0; i < loops; i += 1) {
    const text = await bodyText(ws)
    if (predicate(text)) return text
    await new Promise(resolve => setTimeout(resolve, interval))
  }
  return await bodyText(ws)
}

async function clickText(ws, text) {
  return await evalPage(ws, `(() => {
    const wanted = ${JSON.stringify(text)}
    const nodes = [...document.querySelectorAll('button,div,span,a')]
      .filter(node => (node.innerText || '').trim() === wanted || (node.innerText || '').trim().split(/\\n/).includes(wanted))
    const textNode = nodes.find(node => {
      const rect = node.getBoundingClientRect()
      return rect.width > 20 && rect.height > 20 && rect.width < 400 && rect.height < 160 && !node.closest('[aria-hidden="true"]')
    })
    let item = textNode
    const clickable = textNode?.closest('button,[role="button"],.ant-card,.ant-btn,[data-testid]')
    if (clickable) item = clickable
    if (!item) return 'NOT_FOUND'
    item.scrollIntoView({ block: 'center', inline: 'center' })
    item.click()
    const rect = item.getBoundingClientRect()
    return JSON.stringify({ text: item.innerText, x: rect.x, y: rect.y, width: rect.width, height: rect.height })
  })()`)
}

async function clickButtonText(ws, text) {
  return await evalPage(ws, `(() => {
    const wanted = ${JSON.stringify(text)}
    const item = [...document.querySelectorAll('button')]
      .find(button => (button.innerText || '').trim().includes(wanted) && !button.disabled)
    if (!item) return 'NOT_FOUND'
    item.scrollIntoView({ block: 'center', inline: 'center' })
    item.click()
    const rect = item.getBoundingClientRect()
    return JSON.stringify({ text: item.innerText, x: rect.x, y: rect.y, width: rect.width, height: rect.height })
  })()`)
}

async function typeAndSend(ws, value) {
  const result = await evalPage(ws, `(() => {
    const textarea = document.querySelector('textarea')
    if (!textarea) return 'NO_TEXTAREA'
    const setter = Object.getOwnPropertyDescriptor(HTMLTextAreaElement.prototype, 'value')?.set
    setter.call(textarea, ${JSON.stringify(value)})
    textarea.dispatchEvent(new Event('input', { bubbles: true }))
    const buttons = [...document.querySelectorAll('button')]
    const item = buttons.find(button => {
      const rect = button.getBoundingClientRect()
      return rect.width >= 30 && rect.height >= 30 && rect.x > window.innerWidth - 140 && !button.disabled
    })
    if (!item) return 'NO_SEND'
    item.click()
    return 'SENT'
  })()`)
  if (result !== 'SENT') throw new Error(`typeAndSend failed: ${result}`)
}

function assertNoExecutionProgress(text, label) {
  if (text.includes('正在执行测试')) {
    throw new Error(`${label}: unexpected execution progress card`)
  }
}

async function main() {
  try {
    await waitForCDP()
    const target = await getJSON(`http://127.0.0.1:${port}/json/new?${encodeURIComponent(appURL)}`, { method: 'PUT' })
    const ws = new WebSocket(target.webSocketDebuggerUrl)
    ws.onmessage = event => {
      const message = JSON.parse(event.data)
      if (message.id && pending.has(message.id)) {
        const callbacks = pending.get(message.id)
        pending.delete(message.id)
        if (message.error) callbacks.reject(new Error(JSON.stringify(message.error)))
        else callbacks.resolve(message.result)
      }
    }
    await new Promise(resolve => { ws.onopen = resolve })
    await callCDP(ws, 'Page.enable')
    await callCDP(ws, 'Runtime.enable')

    const login = await getJSON(`${apiURL}/auth/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email, password }),
    })

    await evalPage(ws, [
      `localStorage.setItem(${JSON.stringify('token')}, ${JSON.stringify(login.token)})`,
      `localStorage.setItem(${JSON.stringify('user')}, ${JSON.stringify(JSON.stringify(login.user))})`,
      `location.href=${JSON.stringify('/enterprise/chat')}`,
    ].join(';'))
    await waitFor(ws, text => text.includes('AI 安全评估助手'), 40, 500)

    console.log(JSON.stringify({ newSessionClick: await clickButtonText(ws, '新建对话') }))
    let text = await waitFor(ws, value => value.includes('AI 安全评估助手'), 60, 500)
    const recommendations = ['合规安全测试', '文言文越狱测试', '提示注入检验', '模板样本组合评估', '已组合攻击回归测试', '内容拒答能力测试']
      .filter(item => text.includes(item))
    assertNoExecutionProgress(text, 'new session')
    console.log(JSON.stringify({ newSessionNoProgress: true, recommendations, newSessionTail: text.slice(-1000) }))
    if (recommendations.length < 4) throw new Error(`expected at least 4 recommendation cards, got ${recommendations.length}`)

    const clickResult = await clickText(ws, '文言文越狱测试')
    await new Promise(resolve => setTimeout(resolve, 1500))
    text = await bodyText(ws)
    let usedFallback = false
    if (!text.includes('请对当前被测模型进行文言文越狱测试')) {
      usedFallback = true
      await typeAndSend(ws, '请对当前被测模型进行文言文越狱测试，优先利用 CCBOS 文言文改写 Skill。')
    }
    text = await waitFor(ws, value => value.includes('执行确认'), waitLoops, 2000)
    assertNoExecutionProgress(text, 'pre-confirm skill request')
    const hasPlan = text.includes('执行确认')
    const hasSkill = text.includes('CCBOS') || text.includes('文言文改写') || text.includes('skillhub:ccbos')
    const leakedJSON = text.includes('"response_source"') || text.includes('```json')
    console.log(JSON.stringify({
      recommendationClickResult: clickResult,
      typedFallbackUsed: usedFallback,
      hasPlan,
      hasSkill,
      leakedJSON,
      preConfirmNoProgress: true,
      tail: text.slice(-1200),
    }))
    if (!hasPlan) throw new Error('expected plan card')
    if (!hasSkill) throw new Error('expected CCBOS/classical-Chinese Skill in plan')
    if (leakedJSON) throw new Error('plan JSON leaked into visible chat text')

    await clickText(ws, '新建对话')
    await typeAndSend(ws, '1')
    text = await waitFor(ws, value => value.includes('安全评估工作台') || value.includes('安全评测工作台') || value.includes('当前已就绪'), 50, 500)
    assertNoExecutionProgress(text, 'plain message')
    console.log(JSON.stringify({ plainMessageNoProgress: true }))

    ws.close()
  } finally {
    browserProcess.kill()
    try {
      fs.rmSync(userData, { recursive: true, force: true })
    } catch {}
  }
}

main().catch(error => {
  browserProcess.kill()
  try {
    fs.rmSync(userData, { recursive: true, force: true })
  } catch {}
  console.error(error)
  process.exit(1)
})
