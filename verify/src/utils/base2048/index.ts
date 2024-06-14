import alphabet from './base2048.txt?raw'

const encodeChars = alphabet.trim().split('\n')
  .map(c => c.charCodeAt(0))
const decodeMap = new Map(encodeChars.map((c, i) => [c, i]))

const tail = [
  0xf0d,
  0xf0e,
  0xf0f,
  0xf10,
  0xf11,
  0xf06,
  0xf08,
  0xf12
]
const tailMap = new Map(tail.map((c, i) => [c, i]))

class DecodeError extends Error {
}

export const decode = (src: string): Uint8Array => {
  const ret = []
  let remaining = 0
  let stage = 0
  let residue = 0

  let se = src.length - 1
  while (se >= 0 && (src[se] === '\r' || src[se] === '\n')) {
    se--
  }

  for (let si = 0; si < src.length; si++) {
    if (src[si] === '\r' || src[si] === '\n') {
      continue
    }
    residue = (residue + 11) % 8
    const c = src[si].charCodeAt(0)
    let newBitsCount = 0
    let newBits = decodeMap.get(c)
    if (newBits === undefined) {
      newBitsCount = 8 - remaining
      newBits = tailMap.get(c)
      if (newBits === undefined || si < se || newBits >= (1 << newBitsCount)) {
        throw new DecodeError(`Invalid character: ${c}`)
      }
    }
    else {
      if (si == se) {
        newBitsCount = 11 - residue
      } else {
        newBitsCount = 11
      }
    }

    stage = (stage << newBitsCount) | newBits
    remaining += newBitsCount

    while (remaining >= 8) {
      remaining -= 8
      ret.push(stage >> remaining)
      stage &= (1 << remaining) - 1
    }
  }
  return new Uint8Array(ret)
}
