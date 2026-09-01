export type ChatMarkdownInline = {
  text: string
  strong?: boolean
  emphasis?: boolean
  code?: boolean
}

export type ChatMarkdownBlock =
  | { type: 'heading'; level: 1 | 2 | 3; children: ChatMarkdownInline[] }
  | { type: 'paragraph'; children: ChatMarkdownInline[] }
  | { type: 'list'; items: ChatMarkdownInline[][] }
  | { type: 'code_block'; text: string }

function pushText(parts: ChatMarkdownInline[], text: string) {
  if (!text) return
  const previous = parts[parts.length - 1]
  if (previous && !previous.strong && !previous.emphasis && !previous.code) {
    previous.text += text
    return
  }
  parts.push({ text })
}

export function parseMarkdownInline(value: string): ChatMarkdownInline[] {
  const parts: ChatMarkdownInline[] = []
  let index = 0

  while (index < value.length) {
    if (value[index] === '`') {
      const end = value.indexOf('`', index + 1)
      if (end > index + 1) {
        parts.push({ text: value.slice(index + 1, end), code: true })
        index = end + 1
        continue
      }
    }

    if (value.startsWith('**', index)) {
      const end = value.indexOf('**', index + 2)
      if (end > index + 2) {
        parts.push({ text: value.slice(index + 2, end), strong: true })
        index = end + 2
        continue
      }
    }

    if (value[index] === '*' && value[index + 1] !== '*') {
      const end = value.indexOf('*', index + 1)
      if (end > index + 1) {
        parts.push({ text: value.slice(index + 1, end), emphasis: true })
        index = end + 1
        continue
      }
    }

    const nextMarkers = [
      value.indexOf('`', index + 1),
      value.indexOf('**', index + 1),
      value.indexOf('*', index + 1),
    ].filter(position => position > -1)
    const next = nextMarkers.length > 0 ? Math.min(...nextMarkers) : value.length
    pushText(parts, value.slice(index, next))
    index = next
  }

  return parts
}

export function parseChatMarkdown(value: string): ChatMarkdownBlock[] {
  const blocks: ChatMarkdownBlock[] = []
  const lines = value.replace(/\r\n/g, '\n').split('\n')
  let paragraph: string[] = []
  let listItems: ChatMarkdownInline[][] = []
  let codeLines: string[] | null = null

  const flushParagraph = () => {
    if (paragraph.length === 0) return
    blocks.push({ type: 'paragraph', children: parseMarkdownInline(paragraph.join('\n')) })
    paragraph = []
  }

  const flushList = () => {
    if (listItems.length === 0) return
    blocks.push({ type: 'list', items: listItems })
    listItems = []
  }

  const flushCode = () => {
    if (codeLines === null) return
    blocks.push({ type: 'code_block', text: codeLines.join('\n') })
    codeLines = null
  }

  for (const line of lines) {
    if (line.trim().startsWith('```')) {
      if (codeLines !== null) {
        flushCode()
      } else {
        flushParagraph()
        flushList()
        codeLines = []
      }
      continue
    }

    if (codeLines !== null) {
      codeLines.push(line)
      continue
    }

    if (!line.trim()) {
      flushParagraph()
      flushList()
      continue
    }

    const heading = /^(#{1,3})\s+(.+)$/.exec(line)
    if (heading) {
      flushParagraph()
      flushList()
      blocks.push({
        type: 'heading',
        level: heading[1].length as 1 | 2 | 3,
        children: parseMarkdownInline(heading[2]),
      })
      continue
    }

    const list = /^\s*(?:[-*]|\d+\.)\s+(.+)$/.exec(line)
    if (list) {
      flushParagraph()
      listItems.push(parseMarkdownInline(list[1]))
      continue
    }

    flushList()
    paragraph.push(line)
  }

  flushCode()
  flushParagraph()
  flushList()

  return blocks.length > 0 ? blocks : [{ type: 'paragraph', children: [{ text: value }] }]
}
