import { parseChatMarkdown } from './chatMarkdown'

function assertEqual(actual: unknown, expected: unknown) {
  if (actual !== expected) {
    throw new Error(`Expected ${String(expected)}, got ${String(actual)}`)
  }
}

const blocks = parseChatMarkdown(`# 能力说明

你好，我可以帮助你完成：
* **模板样本评估**
* *越狱风险检测*
`)

assertEqual(blocks[0]?.type, 'heading')
if (blocks[0]?.type !== 'heading') throw new Error('Expected first block to be a heading')
assertEqual(blocks[0].level, 1)
assertEqual(blocks[0].children[0]?.text, '能力说明')
assertEqual(blocks[1]?.type, 'paragraph')
assertEqual(blocks[2]?.type, 'list')
if (blocks[2]?.type !== 'list') throw new Error('Expected third block to be a list')
assertEqual(blocks[2].items.length, 2)
assertEqual(blocks[2].items[0]?.[0]?.strong, true)
assertEqual(blocks[2].items[0]?.[0]?.text, '模板样本评估')
assertEqual(blocks[2].items[1]?.[0]?.emphasis, true)
assertEqual(blocks[2].items[1]?.[0]?.text, '越狱风险检测')
