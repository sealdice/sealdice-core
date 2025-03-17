import { Tag } from "@lezer/highlight"
import {
  LanguageSupport,
  StreamLanguage,
  HighlightStyle,
  TagStyle,
  syntaxHighlighting,
} from "@codemirror/language"
import { completeFromList } from "@codemirror/autocomplete"
import { CharItem } from "~/logManager/types"
import { Extension } from "@codemirror/state";
import * as twColors from 'tailwindcss/colors'

export const reNameLine = /^([^(<\n]+)(\([^(\n]+\)|\<[^(\n]+\>)?(\s+)(\d{4}\/\d{1,2}\/\d{1,2} )?(\d{1,2}:\d{1,2}:\d{2})( #\d+)?/
export const reNameLine2 = /([^(<\n]+)(\([^(\n]+\)|\<[^(\n]+\>)?(\s+)(\d{4}\/\d{1,2}\/\d{1,2} )?(\d{1,2}:\d{1,2}:\d{2})( #\d+)?/g

let nameReplace = (n: string) => {
  return n.replaceAll('.', '·').replaceAll(' ', '_').replaceAll(`/`, '_') //.replaceAll('(', '（').replaceAll(')', '）')
}

export function generateLang(pcList: CharItem[], options: any = undefined): Extension[] {
  let tagNameLine = Tag.define()
  let tagNameLineHost = Tag.define()
  let tagNameLineDice = Tag.define()
  let tagNameLineHide = Tag.define()

  const tokenMap: { [name: string]: Tag } = {
    'name-line': tagNameLine,
    'name-line-host': tagNameLineHost,
    'name-line-dice': tagNameLineDice,
    'name-line-hide': tagNameLineHide,
  }

  const highLights: TagStyle[] = [
    { tag: tagNameLine, color: twColors.blue['500'], fontWeight: '500' }, // 表示角色
    { tag: tagNameLineHost, color: twColors.pink['500'], fontWeight: '500' }, // 表示主持人
    { tag: tagNameLineDice, color: twColors.zinc['500'], fontWeight: '500' }, // 表示骰子
    {
      tag: tagNameLineHide,
      color: twColors.gray['500'],
      fontWeight: '500',
      fontStyle: 'italic',
      'text-decoration': 'line-through'
    }, // 带斜杠，表示隐藏
  ]

  const pcMap: { [name: string]: CharItem } = {}

  for (let i of pcList) {
    i.name = i.name.replaceAll('(', '（').replaceAll(')', '）');
    const theName = nameReplace(i.name)
    const tag = Tag.define()
    const tag2 = Tag.define()
    const tag3 = Tag.define()
    const tagImg = Tag.define()

    tokenMap[`speak-${theName}`] = tag
    highLights.push({ tag, color: i.color })

    if (options.offSiteHide) {
      tokenMap[`offsite-${theName}`] = tag2
      highLights.push({ tag: tag2, color: i.color, 'text-decoration': 'line-through' })
    } else {
      tokenMap[`offsite-${theName}`] = tag
    }

    if (options.commandHide) {
      tokenMap[`command-${theName}`] = tag3
      highLights.push({ tag: tag3, color: i.color, 'text-decoration': 'line-through' })
    } else {
      tokenMap[`command-${theName}`] = tag
    }

    tokenMap[`image-${theName}`] = tagImg
    if (options.imageHide) {
      highLights.push({ tag: tagImg, color: '#999', 'text-decoration': 'line-through' })
    } else {
      highLights.push({ tag: tagImg, color: '#999' })
    }

    pcMap[i.name] = i
  }

  const language = StreamLanguage.define<{ mode: number, name: string, nextN: string[], text: string, pc: CharItem }>({
    name: 'story-log',
    startState() {
      return {
        mode: 0,
        name: '',
        nextN: [],
        text: '',
        pc: {} as CharItem
      }
    },

    token(stream, state) {
      // if (state.nextN.length) {
      //   const val = state.nextN.shift()
      //   console.log('???', state.nextN.length, stream.pos, stream.start)
      //   stream.pos += 1
      //   return val as string
      // }

      switch (state.mode) {
        case 0: {
          const m = stream.match(reNameLine) as RegExpMatchArray
          if (m) {
            const mName = m[1]
            state.name = nameReplace(m[1])
            state.mode = 1
            const pc = pcMap[mName]
            state.pc = pc
            if (pc) {
              if (pc.role === '主持人') return "name-line-host"
              if (pc.role === '骰子') return "name-line-dice"
              if (pc.role === '隐藏') return "name-line-hide"
            }
            return "name-line"
          }
          break
        }
        case 1: {
          // 这个最优先，防止有人起奇怪的名字
          let m = stream.match(reNameLine) as RegExpMatchArray
          if (m) {
            state.mode = 0
            stream.pos -= m[0].length
            state.text = ''
            return `speak-${state.name}`
          }

          // let m = /\[CQ:image,[^\]]+\]/g.exec(state.text) as RegExpMatchArray
          m = stream.match(/^\[CQ:(image|face)(,summary=\[动画表情\])?,[^\]]+\]/g) as RegExpMatchArray
          if (m) {
            // stream.pos -= m[0].length
            // stream.start -= m[0].length
            state.text = ''
            return `image-${state.name}`
          }

          m = stream.match(/^\[mirai:(image|marketface):[^\]]+\]/g) as RegExpMatchArray
          if (m) {
            // stream.pos -= m[0].length
            // stream.start -= m[0].length
            state.text = ''
            return `image-${state.name}`
          }

          m = stream.match(/^\[(image|图):[^\]]+\]/g) as RegExpMatchArray
          if (m) {
            // stream.pos -= m[0].length
            // stream.start -= m[0].length
            state.text = ''
            return `image-${state.name}`
          }

          m = stream.match(/^[\.。]\S+.*$/) as RegExpMatchArray
          if (m && m[0] === stream.string) {
            return `command-${state.name}`
          }

          const skipOffSite = (state.pc && state.pc.role === '骰子')
          if (!skipOffSite) {
            // 塔骰会有这种: [xxxx]进行了xxx操作
            m = stream.match(/^[【(（].+$/) as RegExpMatchArray
            if (m && m[0] === stream.string) {
              return `offsite-${state.name}`
            }
          }
          break
        }
      }

      stream.next()

      if (state.mode === 1) {
        return `speak-${state.name}`
      }
      return "invalid"
    },

    tokenTable: tokenMap
  })

  const exampleCompletion = language.data.of({
    autocomplete: completeFromList([])
  })

  const myHighlightStyle = syntaxHighlighting(HighlightStyle.define(highLights))

  return [
    new LanguageSupport(language, [exampleCompletion]),
    myHighlightStyle
  ]
}
