import type QHeader from './components/QHeader.vue'
import type QMain from './components/QMain.vue'
import type QReply from './components/QReply.vue'
import type QText from './components/QText.vue'
import type QImage from './components/QImage.vue'
import type QFile from './components/QFile.vue'
import type QTip from './components/QTip.vue'
import type QVoice from './components/QVoice.vue'
import type QVoiceLegacy from './components/QVoiceLegacy.vue'
import type QMessageItem from './components/base/QMessageBase.vue'
import type QForward from './components/QForward.vue'

declare module 'vue' {
  export interface GlobalComponents {
    QHeader: typeof QHeader
    QMain: typeof QMain
    QReply: typeof QReply
    QText: typeof QText
    QImage: typeof QImage
    QFile: typeof QFile
    QTip: typeof QTip
    QVoice: typeof QVoice
    QVoiceLegacy: typeof QVoiceLegacy
    QForward: typeof QForward
    QMessageItem: typeof QMessageItem
  }
}
