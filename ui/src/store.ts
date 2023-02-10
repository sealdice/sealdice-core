import { apiFetch, backend } from './backend'

export enum goCqHttpStateCode {
  Init              = 0,
	InLogin           = 1,
	InLoginQrCode     = 2,
	InLoginBar        = 3,
	InLoginVerifyCode = 6,
	InLoginDeviceLock = 7,
	LoginSuccessed    = 10,
	LoginFailed       = 11,
	Closed            = 20
}

export interface AdapterQQ {
  DiceServing: boolean
  connectUrl: string;
  curLoginFailedReason: string
  curLoginIndex: number
  goCqHttpState: goCqHttpStateCode
  inPackGoCqHttpLastRestricted: number
  inPackGoCqHttpProtocol: number
  useInPackGoCqhttp: boolean;
  goCqHttpLoginVerifyCode: string;
  goCqHttpLoginDeviceLockUrl: string
}

interface TalkLogItem {
  name?: string
  content: string
  isSeal?: boolean
}

export interface DiceConnection {
  id: string;
  state: number;
  platform: string;
  workDir: string;
  enable: boolean;
  protocolType: string;
  nickname: string;
  userId: number;
  groupNum: number;
  cmdExecutedNum: number;
  cmdExecutedLastTime: number;
  onlineTotalTime: number;

  adapter: AdapterQQ;
}

const urlPrefix = 'sd-api'

interface DiceServer {
  config: any
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
  versionNew: string
  versionNewNote: string
  versionCode: number
  versionNewCode: number
  memoryAlloc: number
  memoryUsedSys: number
  uptime: number
}

import { defineStore } from 'pinia'

export const useStore = defineStore('main', {
  state: () => {
    return {
      salt: '',
      token: '',
      index: 0,
      canAccess: false,
      diceServers: [] as DiceServer[],

      talkLogs: [
        {
          content: '海豹，正在等待。\n设置中添加Master名为UI:1001\n即可在此界面使用master命令!',
          isSeal: true
        },
        {
          content: '（请注意，当前会话记录在刷新页面后会消失）',
          isSeal: true
        },
      ] as TalkLogItem[]
    }
  },
  getters: {
    curDice(): DiceServer {
      if (this.diceServers.length === 0) {
        this.diceServers.push({
          baseInfo: { version: '0.0', versionNew: '0.0', memoryUsedSys: 0, memoryAlloc: 0, uptime: 0, versionNewNote: '', versionCode: 0, versionNewCode: 0 },
          customTexts: {},
          customTextsHelpInfo: {},
          logs: [],
          conns: [],
          qrcodes: {},
          config: {}
        })
      }

      return this.diceServers[this.index]
    }

  },
  actions: {
    async customTextSave(category: string) {
      await backend.post(urlPrefix+'/configs/customText/save', { data: this.curDice.customTexts[category], category })
    },

    async getBaseInfo() {
      const info = await backend.get(urlPrefix+'/baseInfo', { timeout: 5000 })
      if (!document.title.includes('-')) {
        if ((info as any).extraTitle && (info as any).extraTitle !== '') {
          document.title = `${(info as any).extraTitle} - ${document.title}`;
        }
      }
      this.curDice.baseInfo = info as any;
      return info
    },

    async getCustomText() {
      const info = await backend.get(urlPrefix+'/configs/customText')
      const data = info as any;
      this.curDice.customTexts = data.texts;
      this.curDice.customTextsHelpInfo = data.helpInfo;
      return info
    },

    async getImConnections() {
      const info = await backend.get(urlPrefix+'/im_connections/list')
      this.diceServers[this.index].conns = info as any;
      return info
    },

    async gocqhttpReloginImConnection(i: DiceConnection) {
      const info = await backend.post(urlPrefix+'/im_connections/gocqhttpRelogin', { id: i.id }, { timeout: 65000 })
      return info as any as DiceConnection
    },

    async addImConnection(form: {accountType: number, account: string, password: string, protocol: number, token: string, url: string, clientID: string}) {
      const {accountType, account, password, protocol, token, url, clientID} = form
      let info = null
      switch (accountType) {
        //QQ
        case 0:
          info = await backend.post(urlPrefix+'/im_connections/add', { account, password, protocol }, { timeout: 65000 })
          break
        case 1:
          info = await backend.post(urlPrefix+'/im_connections/addDiscord', {token}, { timeout: 65000 })
          break
        case 2:
          info = await backend.post(urlPrefix+'/im_connections/addKook', {token}, { timeout: 65000 })
          break
        case 3:
          info = await backend.post(urlPrefix+'/im_connections/addTelegram', {token}, { timeout: 65000 })
          break
        case 4:
          info = await backend.post(urlPrefix+'/im_connections/addMinecraft', {url}, { timeout: 65000 })
          break
        case 5:
          info = await backend.post(urlPrefix+'/im_connections/addDodo', {clientID, token}, { timeout: 65000 })
      }
      return info as any as DiceConnection
    },

    async removeImConnection(i: DiceConnection) {
      const info = await backend.post(urlPrefix+'/im_connections/del', { id: i.id })
      return info as any as DiceConnection
    },

    async getImConnectionsQrCode(i: DiceConnection) {
      const info = await backend.post(urlPrefix+'/im_connections/qrcode', { id: i.id })
      return info as any as { img: string }
    },

    async getImConnectionsSetEnable(i: DiceConnection, enable: boolean) {
      const info = await backend.post(urlPrefix+'/im_connections/set_enable', { id: i.id, enable })
      return info as any as DiceConnection
    },

    async getImConnectionsSetData(i: DiceConnection, { protocol }: { protocol: number }) {
      const info = await backend.post(urlPrefix+'/im_connections/set_data', { id: i.id, protocol })
      return info as any as DiceConnection
    },

    async logFetchAndClear() {
      const info = await backend.get(urlPrefix+'/log/fetchAndClear')
      this.curDice.logs = info as any;
    },

    async diceConfigGet() {
      const info = await backend.get(urlPrefix+'/dice/config/get')
      this.curDice.config = info as any;
    },

    async diceConfigSet(data: any) {
      await backend.post(urlPrefix+'/dice/config/set', data)
      await this.diceConfigGet()
    },

    async diceExec(text: string) {
      const info = await backend.post(urlPrefix+'/dice/exec', { message: text })
      return info as any
    },

    async setCustomReply(data: any) {
      const info = await backend.post(urlPrefix+'/configs/custom_reply/save', data)
      return info
    },

    async getCustomReply(filename: string) {
      const info = await backend.get(urlPrefix+'/configs/custom_reply', { params: { filename } })
      return info
    },

    async customReplyFileNew(filename: string) {
      const info = await backend.post(urlPrefix+'/configs/custom_reply/file_new', { filename });
      return info
    },

    async customReplyFileDelete(filename: string) {
      const info = await backend.post(urlPrefix+'/configs/custom_reply/file_delete', { filename });
      return info
    },

    async customReplyFileDownload(filename: string) {
      const info = await backend.post(urlPrefix+'/configs/custom_reply/file_download', { filename });
      return info
    },

    async customReplyFileUpload({ form }: any) {
      const info = await backend.post(urlPrefix+'/configs/custom_reply/file_upload', form)
      return info as any
    },

    async customReplyFileList() {
      const info = await backend.get(urlPrefix+'/configs/custom_reply/file_list')
      return info
    },

    async customReplyDebugModeGet() {
      const info = await backend.get(urlPrefix+'/configs/custom_reply/debug_mode')
      return info
    },

    async customReplyDebugModeSet(value: boolean) {
      const info = await backend.post(urlPrefix+'/configs/custom_reply/debug_mode', { value })
      return info
    },

    async backupList() {
      const info = await backend.get(urlPrefix+'/backup/list')
      return info as any
    },

    async backupConfigGet() {
      const info = await backend.get(urlPrefix+'/backup/config_get')
      return info as any
    },

    async backupConfigSave(data: any) {
      const info = await backend.post(urlPrefix+'/backup/config_set', data)
      return info as any
    },

    async backupDoSimple() {
      const info = await backend.post(urlPrefix+'/backup/do_backup')
      return info as any
    },

    // ban list相关
    async banConfigGet() {
      const info = await backend.get(urlPrefix+'/banconfig/get')
      return info as any
    },

    async banConfigSet(data: any) {
      const info = await backend.post(urlPrefix+'/banconfig/set', data)
      return info as any
    },

    async banConfigMapGet() {
      const info = await backend.get(urlPrefix+'/banconfig/map_get')
      return info as any
    },

    async banConfigMapDeleteOne(data: any) {
      const info = await backend.post(urlPrefix+'/banconfig/map_delete_one', data)
      return info as any
    },

    async banConfigMapAddOne(id: string, rank: number, name: string, reason: string) {
      const info = await backend.post(urlPrefix+'/banconfig/map_add_one', {
        ID: id,
        rank,
        name,
        reasons: reason ? [reason] : []
      })
      return info as any
    },

    // 群组列表
    async groupList() {
      const info = await backend.get(urlPrefix+'/group/list')
      return info as any
    },

    async groupSetOne(data: any) {
      const info = await backend.post(urlPrefix+'/group/set_one', data)
      return info as any
    },

    async setGroupQuit(data: any) {
      const info = await backend.post(urlPrefix+'/group/quit_one', data)
      return info
    },

    // 牌堆
    async deckList() {
      const info = await backend.get(urlPrefix+'/deck/list')
      return info as any
    },

    async deckReload() {
      const info = await backend.post(urlPrefix+'/deck/reload')
      return info as any
    },

    async deckSetEnable({ index, enable }: any) {
      const info = await backend.post(urlPrefix+'/deck/enable', { index, enable })
      return info as any
    },

    async deckDelete({ index }: any) {
      const info = await backend.post(urlPrefix+'/deck/delete', { index })
      return info as any
    },

    async deckUpload({ form }: any) {
      const info = await backend.post(urlPrefix+'/deck/upload', form)
      return info as any
    },

    async jsList(): Promise<JsScriptInfo[]> {
      return await apiFetch(urlPrefix+'/js/list', { method: 'GET', headers: {
        token: this.token
      }})
    },
    async jsGetRecord() {
      return await apiFetch(urlPrefix+'/js/get_record', { method: 'GET', headers: {
        token: this.token
      }}) as {
        outputs: string[]
      }
    },
    async jsUpload({ form }: any) {
      const info = await backend.post(urlPrefix+'/js/upload', form)
      return info as any
    },
    async jsDelete({ index }: any) {
      const info = await backend.post(urlPrefix+'/js/delete', { index })
      return info as any
    },
    async jsReload() {
      return await apiFetch(urlPrefix+'/js/reload', {
        headers: {
          token: this.token
        }
      })
    },
    async jsExec(code: string) {
      return await apiFetch(urlPrefix+'/js/execute', {body: { value: code }}) as {
        ret: any,
        outputs: string[],
        err: string,
      }
    },

    async toolOnebot() {
      return await apiFetch(urlPrefix+'/tool/onebot', {
        headers: {
          token: this.token
        }
      }) as {
        ok: boolean,
        ip: string,
        errText: string
      }
    },

    async upgrade() {
      const info = await backend.post(urlPrefix+'/dice/upgrade')
      return info
    },

    async signIn(password: string) {
      try {
        const ret = await backend.post(urlPrefix+'/signin', { password })
        const token = (ret as any).token
        this.token = token
        backend.defaults.headers.common['token'] = token
        localStorage.setItem('t', token)
        this.canAccess = true
      } catch {
        this.canAccess = false
      }
    },

    async checkSecurity(): Promise<boolean> {
      return (await backend.get(urlPrefix+'/checkSecurity') as any).isOk
    },

    async trySignIn(): Promise<boolean> {
      this.salt = (await backend.get(urlPrefix+'/signin/salt') as any).salt
      let token = localStorage.getItem('t')
      try {
        await backend.get(urlPrefix+'/hello', {
          headers: {token: token as string}
        })
        this.token = token as string
        backend.defaults.headers.common['token'] = this.token
        this.canAccess = true
      } catch (e) {
        this.canAccess = false
        // 试图做一次登录，以获取token
        await this.signIn('defaultSignin')
      }
      return this.token != ''
    }
  }
})
