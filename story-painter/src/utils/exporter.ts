import dayjs from "dayjs";
import { saveAs } from 'file-saver';
import { CharItem, LogItem } from "~/store";

// TODO: 移植到logMan/exporters
export function exportFileQQ(results: LogItem[], options: any = undefined) {
  let text = ''
  for (let i of results) {
    if (i.isRaw) continue;
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
    text += `${i.nickname}${userid} ${timeText}\n${i.message.replaceAll('<br />', '\n')}\n\n`
  }

  saveAs(new Blob([text],  {type: "text/plain;charset=utf-8"}), '跑团记录(QQ风格).txt')
  return text
}

export function exportFileIRC(results: LogItem[], options: any = undefined) {
  let text = ''
  for (let i of results) {
    if (i.isRaw) continue;
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
    text += `${timeText}<${i.nickname}${userid}>:${i.message.replaceAll('<br />', '\n')}\n\n`
  }

  saveAs(new Blob([text],  {type: "text/plain;charset=utf-8"}), '跑团记录(主流风格).txt')
  return text
}

export function exportFileRaw(doc: string) {
  saveAs(new Blob([doc],  {type: "text/plain;charset=utf-8"}), '跑团记录(未处理).txt')
}

export function exportFileDocx(html: string, options: any = undefined) {
  const text = `MIME-Version: 1.0
Content-Type: multipart/related; boundary="----=_NextPart_WritingBug"

此文档为“单个文件网页”，也称为“Web 档案”文件。如果您看到此消息，但是您的浏览器或编辑器不支持“Web 档案”文件。请下载支持“Web 档案”的浏览器。

------=_NextPart_WritingBug
Content-Type: text/html; charset="utf-8"

<!DOCTYPE html>
<html>
<head><meta charset="utf-8"></head>
<body>
` + html +
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
