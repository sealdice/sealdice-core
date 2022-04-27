import dayjs from "dayjs";
import { saveAs } from 'file-saver';
import { CharItem, LogItem } from "~/store";

// 注意，秒钟数一定要是2个，不然会出事
export const reNameLine = /^([^(\n]+)(\(\d+\))?(\s+)((\d{4}\/\d{1,2}\/\d{1,2} )?(\d{1,2}:\d{1,2}:\d{2}))/m

export function convertToLogItems(doc: string, pcList: CharItem[], options: any = undefined, htmlText: boolean = false) {
  let pos = 0
  let text = ''
  // console.log(doc)
  let results: LogItem[] = []

  let state = 0
  let curItem = {} as LogItem

  while (pos < doc.length) {
    const isLast = pos == doc.length - 1
    text += doc[pos]

    if (text.length > 2000) text = ''

    switch (state) {
      case 0: {
        let m = reNameLine.exec(text)
        if (m) {
          curItem.nickname = m[1]
          if (m[2] && m[2].length) {
            curItem.IMUserId = m[2].slice(1, m[2].length-1) as any
          }
          curItem.time = dayjs(m[4]).unix()
          if (isNaN(curItem.time)) {
            curItem.time = m[4] as any
          }
          text = ''
          state = 1
        }
        break
      }
      case 1: {
        const m = reNameLine.exec(text)
        if (m) {
          state = 0
          text = text.slice(0, text.length - m[0].length)
          pos -= (m[0].length + 1)
        }

        if (m || isLast) {
          curItem.message = text.trim()
          text = ''
          results.push(curItem)
          curItem = {} as LogItem
        }
        break
      }
    }

    pos += 1
  }

  let _pcDict: { [key: string]: any } = {}
  for (let i of pcList) {
    if (!_pcDict[i.name]) _pcDict[i.name] = i
  }

  let findPC = (name: string) => {
    return _pcDict[name]
    // for (let i of pcList) {
    //   if (i.name === name) {
    //     return i
    //   }
    // }
  }

  let allUserIds = []
  for (let i of results) {
    allUserIds.push(i.IMUserId)
  }

  let finalResults = []
  for (let i of results) {
    let msg = i.message

    const pc = findPC(i.nickname)
    if (pc?.role === '隐藏') {
      continue
    }

    i.color = pc?.color
    i.isDice = pc?.role === '骰子'

    // 替换图片、表情
    if (options.imageHide) {
      msg = msg.replaceAll(/\[CQ:(image|face),[^\]]+\]/g, '')
    } else {
      if (htmlText) {
        msg = msg.replaceAll(/\[CQ:image,[^\]]+?url=([^\]]+)\]/g, '<img style="max-width: 300px" src="$1" />')
      }
    }

    if (options.imageHide) {
      msg = msg.replaceAll(/\[mirai:(image|marketface):[^\]]+\]/g, '')
    } else {
      if (htmlText) {
        msg = msg.replaceAll(/\[mirai:image:\{([A-Z0-9]+)-([A-Z0-9]+)-([A-Z0-9]+)-([A-Z0-9]+)-([A-Z0-9]+)}([^\]]+?)\]/g, '<img style="max-width: 300px" src="https://gchat.qpic.cn/gchatpic_new/0/0-0-$1$2$3$4$5/0?term=2" />')
      }
    }

    if (options.imageHide) {
      msg = msg.replaceAll(/\[(image|图):[^\]]+\]/g, '')
    } else {
      if (htmlText) {
        msg = msg.replaceAll(/\[(?:image|图):([^\]]+)?([^\]]+)\]/g, '<img style="max-width: 300px" src="$1" />')
      }
    }

    // 过滤其他任何CQ码
    msg = msg.replaceAll(/\[CQ:.+?,[^\]]+\]/g, '')
    // 过滤mirai
    msg = msg.replaceAll(/\[mirai:.+?:[^\]]+\]/g, '')

    // 替换场外发言
    if (options.offSiteHide && (!i.isDice)) {
      msg = msg.replaceAll(/^[【(（].+?$/gm, '')
    }

    // 替换指令
    if (options.commandHide) {
      msg = msg.replaceAll(/^[\.。]\S+.*$/gm, '')
    }

    // 替换残留QQ号
    if (options.userIdHide) {
      for (let i of allUserIds) {
        msg = msg.replaceAll(`(${i})`, '')
      }
    }

    if (msg) {
      // 换行处理
      if (msg.includes('\n')) {
        msg = msg
      }
      msg = msg.replaceAll('\n', '<br />')
      i.message = msg
      finalResults.push(i)
    }
  }

  // console.log(finalResults)
  return finalResults
}

export function timeConvert(str: string) {

}

export function exportFileQQ(results: LogItem[], options: any = undefined) {
  let text = ''
  for (let i of results) {
    let timeText = i.time.toString()
    if (typeof i.time === 'number') {
      timeText = dayjs.unix(i.time).format(options.yearHide ? 'HH:mm:ss' : 'YYYY/MM/DD HH:mm:ss')
    }
    if (options.timeHide) {
      timeText = ''
    }
    let userid = '(' + i.IMUserId + ')'
    if (options.userIdHide) {
      userid = ''
    }
    text += `${i.nickname}${userid} ${timeText}\n${i.message}\n\n`
  }

  saveAs(new Blob([text],  {type: "text/plain;charset=utf-8"}), '跑团记录(QQ风格).txt')
  return text
}

export function exportFileIRC(results: LogItem[], options: any = undefined) {
  let text = ''
  for (let i of results) {
    let timeText = i.time.toString()
    if (typeof i.time === 'number') {
      timeText = dayjs.unix(i.time).format(options.yearHide ? 'HH:mm:ss' : 'YYYY/MM/DD HH:mm:ss')
    }
    if (options.timeHide) {
      timeText = ''
    }
    let userid = '(' + i.IMUserId + ')'
    if (options.userIdHide) {
      userid = ''
    }
    text += `${timeText}<${i.nickname}${userid}>:${i.message}\n\n`
  }

  saveAs(new Blob([text],  {type: "text/plain;charset=utf-8"}), '跑团记录(主流风格).txt')
  return text
}

export function exportFileRaw(doc: string) {
  saveAs(new Blob([doc],  {type: "text/plain;charset=utf-8"}), '跑团记录(未处理).txt')
}

export function exportFileDocx(el: HTMLDivElement, options: any = undefined) {
  const solveImg = (el: Element) => {
    if (el.tagName === 'IMG') {
      el.setAttribute('width', `${el.clientWidth}`)
      el.setAttribute('height', `${el.clientHeight}`)
    }
    for (let i = 0; i < el.children.length; i += 1) {
      solveImg(el.children[i])
    }
  }
  solveImg(el)

  const text = `MIME-Version: 1.0
Content-Type: multipart/related; boundary="----=_NextPart_WritingBug"

此文档为“单个文件网页”，也称为“Web 档案”文件。如果您看到此消息，但是您的浏览器或编辑器不支持“Web 档案”文件。请下载支持“Web 档案”的浏览器。

------=_NextPart_WritingBug
Content-Type: text/html; charset="utf-8"

<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body>
` + el.innerHTML +
`
</body>
</html>
------=_NextPart_WritingBug
Content-Transfer-Encoding: quoted-printable
Content-Type: text/xml; charset="utf-8"

<xml xmlns:o=3D"urn:schemas-microsoft-com:office:office">
 <o:MainFile HRef=3D"../file4969.htm"/>
 <o:File HRef=3D"themedata.thmx"/>
 <o:File HRef=3D"colorschememapping.xml"/>
 <o:File HRef=3D"header.htm"/>
 <o:File HRef=3D"filelist.xml"/>
</xml>
------=_NextPart_WritingBug--`

  saveAs(new Blob([text],  {type: "application/msword"}), '跑团记录.doc')
  return text
}
