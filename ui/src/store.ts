import { apiFetch, backend } from './backend'

import { addImConnectionForm } from '../src/components/PageConnectInfoItems.vue'
import { defineStore } from 'pinia'

export enum goCqHttpStateCode {
  Init = 0,
  InLogin = 1,
  InLoginQrCode = 2,
  InLoginBar = 3,
  InLoginVerifyCode = 6,
  InLoginDeviceLock = 7,
  LoginSuccessed = 10,
  LoginFailed = 11,
  Closed = 20
}

export interface AdapterQQ {
  DiceServing: boolean
  connectUrl: string;
  curLoginFailedReason: string
  curLoginIndex: number
  loginState: goCqHttpStateCode
  inPackGoCqHttpLastRestricted: number
  inPackGoCqHttpProtocol: number
  inPackGoCqHttpAppVersion: string,
  implementation: string
  useInPackGoCqhttp: boolean;
  goCqHttpLoginVerifyCode: string;
  goCqHttpLoginDeviceLockUrl: string;
  ignoreFriendRequest: boolean;
  goCqHttpSmsNumberTip: string;
  useSignServer: boolean;
  signServerConfig: any;
  redVersion: string;
  host: string;
  port: number;
  appID: number;
  isReverse: boolean;
  reverseAddr: string;
  builtinMode: 'gocq' | 'lagrange'
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

export const urlPrefix = 'sd-api'

interface DiceServer {
  config: any
  customTextsHelpInfo: {
    [k: string]: {
      [k: string]: {
        filename: string[],
        origin: (string[])[],
        vars: string[],
        modified: boolean,
        notBuiltin: boolean,
        topOrder: number,
        subType: string,
        extraText: string,
      }
    }
  }
  customTexts: { [k: string]: { [k: string]: (string[])[] } }
  logs: { level: string, ts: number, caller: string, msg: string }[]
  conns: DiceConnection[]
  baseInfo: DiceBaseInfo
  qrcodes: { [key: string]: string }
}

interface DiceBaseInfo {
  appChannel: string
  version: string
  versionSimple: string
  versionNew: string
  versionNewNote: string
  versionCode: number
  versionNewCode: number
  memoryAlloc: number
  memoryUsedSys: number
  uptime: number
  OS: string
  arch: string
}



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
          content: '海豹已就绪。此界面可视为私聊窗口。\n设置中添加Master名为UI:1001\n即可在此界面使用master命令!',
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
          baseInfo: {
            appChannel: 'stable',
            version: '0.0',
            versionSimple: '0.0',
            versionNew: '0.0',
            memoryUsedSys: 0,
            memoryAlloc: 0,
            uptime: 0,
            versionNewNote: '',
            versionCode: 0,
            versionNewCode: 0,
            OS: '',
            arch: ''
          },
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
      await backend.post(urlPrefix + '/configs/customText/save', { data: this.curDice.customTexts[category], category })
    },

    async getBaseInfo() {
      const info = await backend.get(urlPrefix + '/baseInfo', { timeout: 5000 })
      if (!document.title.includes('-')) {
        if ((info as any).extraTitle && (info as any).extraTitle !== '') {
          document.title = `${(info as any).extraTitle} - ${document.title}`;
        }
      }
      this.curDice.baseInfo = info as any;
      return info
    },

    async getCustomText() {
      const info = await backend.get(urlPrefix + '/configs/customText')
      const data = info as any;
      this.curDice.customTexts = data.texts;
      this.curDice.customTextsHelpInfo = data.helpInfo;
      return info
    },

    async getImConnections() {
      const info = await backend.get(urlPrefix + '/im_connections/list')
      this.diceServers[this.index].conns = info as any;
      return info
    },

    async getSupportedQQVersions() {
      const info: { result: true, versions: string[] } | { result: false } = await backend.get(urlPrefix + '/im_connections/qq/get_versions')
      return info
    },

    async gocqhttpReloginImConnection(i: DiceConnection) {
      const info = await backend.post(urlPrefix + '/im_connections/gocqhttpRelogin', { id: i.id }, { timeout: 65000 })
      return info as any as DiceConnection
    },

    async news(): Promise<{ result: true, checked: boolean, news: string, newsMark: string } | { result: false, err?: string }> {
      const info = await backend.get(urlPrefix + '/utils/news')
      return info as any
    },

    async checkNews(newsMark: string): Promise<{ result: true; newsMark: string; } | { result: false }> {
      const info = await backend.post(urlPrefix + '/utils/check_news', { newsMark }, { timeout: 5000 })
      return info as any
    },

    async addImConnection(form: addImConnectionForm ) {
      const {
        accountType, nickname, account, password, protocol, appVersion, token, botToken, appToken, proxyURL, url, host, port, appID, appSecret, clientID, robotCode, implementation, relWorkDir, connectUrl, accessToken, useSignServer, signServerConfig, reverseAddr, onlyQQGuild, platform } = form
      let info = null
      switch (accountType) {
        //QQ
        case 0:
          if (implementation === 'gocq') {
            info = await backend.post(urlPrefix + '/im_connections/add', { account, password, protocol, appVersion, useSignServer, signServerConfig }, { timeout: 65000 })
          } else if (implementation === 'walle-q') {
            info = await backend.post(urlPrefix + '/im_connections/addWalleQ', { account, password, protocol }, { timeout: 65000 })
          }
          break
        case 1:
          info = await backend.post(urlPrefix + '/im_connections/addDiscord', { token, proxyURL }, { timeout: 65000 })
          break
        case 2:
          info = await backend.post(urlPrefix + '/im_connections/addKook', { token }, { timeout: 65000 })
          break
        case 3:
          info = await backend.post(urlPrefix + '/im_connections/addTelegram', { token, proxyURL }, { timeout: 65000 })
          break
        case 4:
          info = await backend.post(urlPrefix + '/im_connections/addMinecraft', { url }, { timeout: 65000 })
          break
        case 5:
          info = await backend.post(urlPrefix + '/im_connections/addDodo', { clientID, token }, { timeout: 65000 })
          break
        case 6:
          info = await backend.post(urlPrefix + '/im_connections/addGocqSeparate', { relWorkDir, connectUrl, accessToken, account }, { timeout: 65000 })
          break
        case 7:
          info = await backend.post(urlPrefix + '/im_connections/addRed', { host, port, token }, { timeout: 65000 })
          break
        case 8:
          info = await backend.post(urlPrefix + '/im_connections/addDingtalk', { clientID, token, nickname, robotCode }, { timeout: 65000 })
          break
        case 9:
          info = await backend.post(urlPrefix + '/im_connections/addSlack', { botToken, appToken }, { timeout: 65000 })
          break;
        case 10:
          info = await backend.post(urlPrefix + '/im_connections/addOfficialQQ', { appID: Number(appID), appSecret, token,onlyQQGuild }, { timeout: 65000 })
          break
        case 11:
          info = await backend.post(urlPrefix + '/im_connections/addOnebot11ReverseWs', { account, reverseAddr }, { timeout: 65000 })
          break
        case 13:
          info = await backend.post(urlPrefix + '/im_connections/addSealChat', { url, token }, { timeout: 65000 })
          break
        case 14:
          info = await backend.post(urlPrefix + '/im_connections/addSatori', { platform, host, port, token }, { timeout: 65000 })
          break
        case 15:
          info = await backend.post(urlPrefix + '/im_connections/addLagrange', { account, protocol }, { timeout: 65000 })
          break
      }
      return info as any as DiceConnection
    },

    async removeImConnection(i: DiceConnection) {
      const info = await backend.post(urlPrefix + '/im_connections/del', { id: i.id })
      return info as any as DiceConnection
    },

    async getImConnectionsQrCode(i: DiceConnection) {
      const info = await backend.post(urlPrefix + '/im_connections/qrcode', { id: i.id })
      return info as any as { img: string }
    },

    async ImConnectionsSmsCodeSet(i: DiceConnection, smsCode: string) {
      const info = await backend.post(urlPrefix + '/im_connections/sms_code_set', { id: i.id, code: smsCode })
      return info as any as {}
    },

    async ImConnectionsCaptchaSet(i: DiceConnection, code: string) {
      console.log('xxx', { id: i.id, code })
      const info = await backend.post(urlPrefix + '/im_connections/gocq_captcha_set', { id: i.id, code })
      return info as any as {}
    },

    async getImConnectionsSetEnable(i: DiceConnection, enable: boolean) {
      const info = await backend.post(urlPrefix + '/im_connections/set_enable', { id: i.id, enable })
      return info as any as DiceConnection
    },

    async getImConnectionsSetData(i: DiceConnection, { protocol, appVersion, ignoreFriendRequest, useSignServer, signServerConfig }: { protocol: number, appVersion: string, ignoreFriendRequest: boolean, useSignServer?: boolean, signServerConfig?: any }) {
      const info = await backend.post(urlPrefix + '/im_connections/set_data', { id: i.id, protocol, appVersion, ignoreFriendRequest, useSignServer, signServerConfig })
      return info as any as DiceConnection
    },

    async logFetchAndClear() {
      const info = await backend.get(urlPrefix + '/log/fetchAndClear')
      this.curDice.logs = info as any;
    },

    async diceConfigGet() {
      const info = await backend.get(urlPrefix + '/dice/config/get')
      this.curDice.config = info as any;
    },

    async diceConfigSet(data: any) {
      await backend.post(urlPrefix + '/dice/config/set', data)
      await this.diceConfigGet()
    },

    async diceMailTest() {
      const res: { result: true } | {
        result: false,
        err: string
      } = await backend.post(urlPrefix + '/dice/config/mail_test')
      return res
    },

    async diceExec(text: string) {
      const info = await backend.post(urlPrefix + '/dice/exec', { message: text })
      return info as any
    },

    async diceUploadToUpgrade({ form }: any) {
      const info = await backend.post(urlPrefix + '/dice/upload_to_upgrade', form, { headers: { "Content-Type": "multipart/form-data" } })
      return info as any
    },

    async getRecentMessage() {
      const info = await backend.get(urlPrefix + '/dice/recentMessage')
      return info as any
    },

    async setCustomReply(data: any) {
      const info = await backend.post(urlPrefix + '/configs/custom_reply/save', data)
      return info
    },

    async getCustomReply(filename: string) {
      const info = await backend.get(urlPrefix + '/configs/custom_reply', { params: { filename } })
      return info
    },

    async customReplyFileNew(filename: string) {
      const info = await backend.post(urlPrefix + '/configs/custom_reply/file_new', { filename });
      return info
    },

    async customReplyFileDelete(filename: string) {
      const info = await backend.post(urlPrefix + '/configs/custom_reply/file_delete', { filename });
      return info
    },

    async customReplyFileDownload(filename: string) {
      const info = await backend.post(urlPrefix + '/configs/custom_reply/file_download', { filename });
      return info
    },

    async customReplyFileUpload({ form }: any) {
      const info = await backend.post(urlPrefix + '/configs/custom_reply/file_upload', form,  { headers: { "Content-Type": "multipart/form-data" } })
      return info as any
    },

    async customReplyFileList() {
      const info = await backend.get(urlPrefix + '/configs/custom_reply/file_list')
      return info
    },

    async customReplyDebugModeGet() {
      const info = await backend.get(urlPrefix + '/configs/custom_reply/debug_mode')
      return info
    },

    async customReplyDebugModeSet(value: boolean) {
      const info = await backend.post(urlPrefix + '/configs/custom_reply/debug_mode', { value })
      return info
    },

    async backupList() {
      const info = await backend.get(urlPrefix + '/backup/list')
      return info as any
    },

    async backupConfigGet() {
      const info = await backend.get(urlPrefix + '/backup/config_get')
      return info as any
    },

    async backupConfigSave(data: any) {
      const info = await backend.post(urlPrefix + '/backup/config_set', data)
      return info as any
    },

    async backupDoSimple() {
      const info = await backend.post(urlPrefix + '/backup/do_backup')
      return info as any
    },

    async backupDelete(name: string) {
      const info = await backend.post(urlPrefix + '/backup/delete', {}, { params: { name } })
      return info as any
    },

    async backupBatchDelete(names: string[]) {
      const info: { result: true } | {
        result: false,
        fails: string[],
      } = await backend.post(urlPrefix + '/backup/batch_delete', { names }, { headers: { token: this.token } })
      return info
    },

    // ban list相关
    async banConfigGet() {
      const info = await backend.get(urlPrefix + '/banconfig/get')
      return info as any
    },

    async banConfigSet(data: any) {
      const info = await backend.post(urlPrefix + '/banconfig/set', data)
      return info as any
    },

    async banConfigMapGet() {
      const info = await backend.get(urlPrefix + '/banconfig/list')
      return info as any
    },

    async banConfigMapDeleteOne(data: any) {
      const info = await backend.post(urlPrefix + '/banconfig/map_delete_one', data)
      return info as any
    },

    async banConfigMapAddOne(id: string, rank: number, name: string, reason: string) {
      const info = await backend.post(urlPrefix + '/banconfig/map_add_one', {
        ID: id,
        rank,
        name,
        reasons: reason ? [reason] : []
      })
      return info as any
    },

    async banUpload({ form }: { form: FormData }): Promise<{ result: true } | {
      result: false,
      err: string
    }> {
      return await backend.post(urlPrefix + '/banconfig/import', form, { headers: { token: this.token, "Content-Type": "multipart/form-data" } })
    },

    // 群组列表
    async groupList() {
      const info = await backend.get(urlPrefix + '/group/list')
      return info as any
    },

    async groupSetOne(data: any) {
      const info = await backend.post(urlPrefix + '/group/set_one', data)
      return info as any
    },

    async setGroupQuit(data: any) {
      const info = await backend.post(urlPrefix + '/group/quit_one', data)
      return info
    },

    // 牌堆
    async deckList() {
      const info = await backend.get(urlPrefix + '/deck/list')
      return info as any
    },

    async deckReload() {
      const info = await backend.post(urlPrefix + '/deck/reload')
      return info as any
    },

    async deckSetEnable({ index, enable }: any) {
      const info = await backend.post(urlPrefix + '/deck/enable', { index, enable })
      return info as any
    },

    async deckDelete({ index }: any) {
      const info = await backend.post(urlPrefix + '/deck/delete', { index })
      return info as any
    },

    async deckUpload({ form }: any) {
      const info = await backend.post(urlPrefix + '/deck/upload', form, { headers: { "Content-Type": "multipart/form-data" } })
      return info as any
    },

    async deckCheckUpdate({ index }: any) {
      const info: { result: false, err: string } | {
        result: true,
        old: string,
        new: string,
        format: 'json' | 'yaml' | 'toml',
        tempFileName: string,
      } = await backend.post(urlPrefix + '/deck/check_update', { index })
      return info
    },

    async deckUpdate({ index, tempFileName }: any) {
      const res: { result: false, err: string } | {
        result: true,
      } = await backend.post(urlPrefix + '/deck/update', { index, tempFileName })
      return res
    },

    async jsStatus(): Promise<boolean> {
      const resp = await apiFetch(urlPrefix + '/js/status', { method: 'GET' })
      return resp.status
    },
    async jsList(): Promise<JsScriptInfo[]> {
      return await apiFetch(urlPrefix + '/js/list', {
        method: 'GET', headers: {
          token: this.token
        }
      })
    },
    async jsGetConfig() {
      return await apiFetch(urlPrefix + '/js/get_configs', {
        method: 'GET', headers: {
          token: this.token
        }
      })
    },
    async jsSetConfig(configs: any) {
      return await backend.post(urlPrefix + '/js/set_configs', configs)
    },
    async jsResetConfig(pluginName: any, key: any) {
      return await backend.post(urlPrefix + '/js/reset_config', { pluginName, key })
    },
    async jsDeleteUnusedConfig(pluginName: any, key: any) {
      return await backend.post(urlPrefix + '/js/delete_unused_config', { pluginName, key })
    },
    async jsGetRecord() {
      return await apiFetch(urlPrefix + '/js/get_record', {
        method: 'GET', headers: {
          token: this.token
        }
      }) as {
        outputs: string[]
      }
    },
    async jsUpload({ form }: any) {
      const info = await backend.post(urlPrefix + '/js/upload', form,  { headers: { "Content-Type": "multipart/form-data" } })
      return info as any
    },
    async jsDelete({ index }: any) {
      const info = await backend.post(urlPrefix + '/js/delete', { index })
      return info as any
    },
    async jsReload() {
      return await apiFetch(urlPrefix + '/js/reload', {
        headers: {
          token: this.token
        }
      })
    },
    async jsShutdown() {
      return await apiFetch(urlPrefix + '/js/shutdown', {
        headers: {
          token: this.token
        }
      })
    },
    async jsExec(code: string) {
      return await apiFetch(urlPrefix + '/js/execute', { body: { value: code } }) as {
        ret: any,
        outputs: string[],
        err: string,
      }
    },
    async jsEnable(body: any) {
      return await apiFetch(urlPrefix + '/js/enable', {
        headers: {
          token: this.token
        },
        body
      })
    },
    async jsDisable(body: any) {
      return await apiFetch(urlPrefix + '/js/disable', {
        headers: {
          token: this.token
        },
        body
      })
    },

    async jsCheckUpdate({ index }: any) {
      const info: { result: false, err: string } | {
        result: true,
        old: string,
        new: string,
        tempFileName: string,
      } = await backend.post(urlPrefix + '/js/check_update', { index })
      return info
    },

    async jsUpdate({ index, tempFileName }: any) {
      const res: { result: false, err: string } | {
        result: true,
      } = await backend.post(urlPrefix + '/js/update', { index, tempFileName })
      return res
    },

    async toolOnebot() {
      return await apiFetch(urlPrefix + '/tool/onebot', {
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
      const info = await backend.post(urlPrefix + '/dice/upgrade', null, { timeout: 120000 })
      return info
    },

    async signIn(password: string) {
      try {
        const ret = await backend.post(urlPrefix + '/signin', { password })
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
      return (await backend.get(urlPrefix + '/checkSecurity') as any).isOk
    },

    async trySignIn(): Promise<boolean> {
      this.salt = (await backend.get(urlPrefix + '/signin/salt') as any).salt
      const token = localStorage.getItem('t')
      try {
        await backend.get(urlPrefix + '/hello', {
          headers: { token: token as string }
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
    },

    async helpDocTree(): Promise<{ result: true, data: HelpDoc[] } | { result: false, err?: string }> {
      return await apiFetch(urlPrefix + '/helpdoc/tree', {
        method: 'GET', headers: {
          token: this.token
        }
      })
    },

    async helpDocReload(): Promise<{ result: true } | { result: false, err?: string }> {
      return await apiFetch(urlPrefix + '/helpdoc/reload', {
        method: 'POST', headers: {
          token: this.token
        }
      })
    },

    async helpDocUpload(form: any): Promise<{ result: true } | { result: false, err?: string }> {
      return await apiFetch(urlPrefix + '/helpdoc/upload', {
        method: 'POST', headers: {
          token: this.token,
        }, body: form
      })
    },

    async helpDocDelete(keys: string[]): Promise<{ result: true } | { result: false, err?: string }> {
      return await apiFetch(urlPrefix + '/helpdoc/delete', {
        method: 'POST',
        headers: { token: this.token },
        body: { keys: keys }
      })
    },

    async helpGetTextItemPage(param: HelpTextItemQuery): Promise<{ result: true; total: number; data: HelpTextItem[] } | { result: false; err?: string }> {
      return await apiFetch(urlPrefix + "/helpdoc/textitem/get_page", {
        method: "POST",
        body: param,
      });
    },

    async helpGetConfig(): Promise<{ aliases: { [key: string]: string[] } }> {
      return await apiFetch(urlPrefix + "/helpdoc/config", {
        method: "GET",
        headers: {
          token: this.token
        }
      })
    },

    async helpSetConfig(param: { aliases: { [key: string]: string[] } }): Promise<{ result: true } | { result: false, err?: string }> {
      console.log("param=", param)
      return await apiFetch(urlPrefix + "/helpdoc/config", {
        method: "POST",
        headers: {
          token: this.token
        },
        body: param
      })
    },
  }
})
