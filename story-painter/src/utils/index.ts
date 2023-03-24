export * from './types';
// export * from './helpers';

/**
* Uses canvas.measureText to compute and return the width of the given text of given font in pixels.
* 
* @param {String} text The text to be rendered.
* @param {String} font The css font descriptor that text is to be rendered with (e.g. "bold 14px verdana").
* 
* @see https://stackoverflow.com/questions/118241/calculate-text-width-with-javascript/21015393#21015393
*/
export function getTextWidth(text: string, font: string): any {
  // re-use canvas object for better performance
  const canvas = (getTextWidth as any).canvas || ((getTextWidth as any).canvas = document.createElement("canvas"));
  const context = canvas.getContext("2d");
  context.font = font;
  const metrics = context.measureText(text);
  return metrics.width;
}

function getCssStyle(element: any, prop: any) {
  return window.getComputedStyle(element, null).getPropertyValue(prop);
}

export function getCanvasFontSize(el = document.body) {
  const fontWeight = getCssStyle(el, 'font-weight') || 'normal';
  const fontSize = getCssStyle(el, 'font-size') || '16px';
  const fontFamily = getCssStyle(el, 'font-family') || 'Times New Roman';

  return `${fontWeight} ${fontSize} ${fontFamily}`;
}

export function msgImageFormat(msg: string, options: any, htmlText = false) {
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

  return msg;
}

export function msgOffTopicFormat(msg: string, options: any, isDice = false) {
  // 替换场外发言
  if (options.offTopicHide && (!isDice)) {
    msg = msg.replaceAll(/^\S*[(（].+?$/gm, '') // 【
  }
  return msg;
}

export function msgCommandFormat(msg: string, options: any) {
  // 替换指令
  if (options.commandHide) {
    msg = msg.replaceAll(/^[\.。\/](.|\n)*$/g, '')
  }
  return msg;
}

// TODO: 名字不贴切，这个是一个收尾替换
export function msgIMUseridFormat(msg: string, options: any, isDice = false) {
  // 替换残留QQ号
  if (options.userIdHide) {
    // for (let i of allUserIds) {
    //   msg = msg.replaceAll(`(${i})`, '')
    // }
  }
  
  if (isDice) {
    // 替换角色的<>
    msg = msg.replaceAll('<', '')
    msg = msg.replaceAll('>', '')
  }

  // 过滤其他任何CQ码
  msg = msg.replaceAll(/\[CQ:(?!image).+?,[^\]]+\]/g, '')
  // 过滤mirai
  msg = msg.replaceAll(/\[mirai:(?!image).+?:[^\]]+\]/g, '')

  // 这个trim是消灭单行空白，例如“@xxxx\nXXXX”虽然还是会造成中间断行，但先不管
  msg = msg.trim();

  return msg;
}
