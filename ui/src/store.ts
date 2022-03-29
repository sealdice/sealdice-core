import { backend } from './backend'

export interface DiceConnection {
  id: string;
  state: number;
  connectUrl: string;
  platform: string;
  workDir: string;
  enable: boolean;
  type: string;
  useInPackGoCqhttp: boolean;
  nickname: string;
  userId: number;
  groupNum: number;
  cmdExecutedNum: number;
  cmdExecutedLastTime: number;
  onlineTotalTime: number;
  inPackGoCqHttpLoginSuccess: boolean;
  inPackGoCqHttpRunning: boolean;
  inPackGoCqHttpQrcodeReady: boolean;
  inPackGoCqHttpNeedQrCode: boolean;
  inPackGoCqHttpLastRestricted: number;
  inPackGoCqHttpLoginDeviceLockUrl: string;
}

interface DiceServer {
  customTextsHelpInfo: { [k: string]: {
    [k: string]: {
      filename: string[],
      origin: (string[])[],
      vars: string[],
      modified: boolean
    }
  }}
  customTexts: { [k: string]: { [k: string]: (string[])[] } }
  logs: { level: string, ts: number, caller: string, msg: string }[]
  conns: DiceConnection[]
  baseInfo: DiceBaseInfo
  qrcodes: { [key: string]: string }
};

interface DiceBaseInfo {
  version: string
  memoryAlloc: number
  memoryUsedSys: number
  uptime: number
}

import { defineStore } from 'pinia'

export const useStore = defineStore('main', {
  state: () => {
    return {
      index: 0,
      diceServers: [] as DiceServer[]
    }
  },
  getters: {
    curDice(): DiceServer {
      if (this.diceServers.length === 0) {
        this.diceServers.push({
          baseInfo: { version: '0.0', memoryUsedSys: 0, memoryAlloc: 0, uptime: 0 },
          customTexts: {},
          customTextsHelpInfo: {},
          logs: [],
          conns: [],
          qrcodes: {}
        })
      }

      return this.diceServers[this.index]
    }

  },
  actions: {
    async customTextSave(category: string) {
      await backend.post('/configs/customText/save', { data: this.curDice.customTexts[category], category })
    },

    async getBaseInfo() {
      const info = await backend.get('/baseInfo')
      this.curDice.baseInfo = info as any;
      return info
    },

    async getCustomText() {
      const info = await backend.get('/configs/customText')
      const data = info as any;
      this.curDice.customTexts = data.texts;
      this.curDice.customTextsHelpInfo = data.helpInfo;
      return info
    },

    async getImConnections() {
      const info = await backend.get('/im_connections/list')
      this.diceServers[this.index].conns = info as any;
      return info
    },

    async gocqhttpReloginImConnection(i: DiceConnection) {
      const info = await backend.post('/im_connections/gocqhttpRelogin', { id: i.id })
      return info as any as DiceConnection
    },

    async addImConnection(form: { account: string, password: string, protocol: number }) {
      const { account, password, protocol } = form
      const info = await backend.post('/im_connections/add', { account, password, protocol })
      return info as any as DiceConnection
    },

    async removeImConnection(i: DiceConnection) {
      const info = await backend.post('/im_connections/del', { id: i.id })
      return info as any as DiceConnection
    },

    async getImConnectionsQrCode(i: DiceConnection) {
      const info = await backend.post('/im_connections/qrcode', { id: i.id })
      return info as any as { img: string }
    },

    async getImConnectionsSetEnable(i: DiceConnection, enable: boolean) {
      const info = await backend.post('/im_connections/set_enable', { id: i.id, enable })
      return info as any as DiceConnection
    },

    async logFetchAndClear() {
      const info = await backend.get('/log/fetchAndClear')
      this.curDice.logs = info as any;
    }

  }
})
