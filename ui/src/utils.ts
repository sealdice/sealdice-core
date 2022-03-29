import { delay } from 'lodash-es'

export function sleep(duration: number) {
  return new Promise<void>((resolve, reject) => {
    delay(resolve, duration)
  })
}
