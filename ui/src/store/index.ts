import { getCustomText, saveCustomText } from '~/api/configs';
import { getAdvancedConfig, getDiceConfig, setAdvancedConfig, setDiceConfig, type DiceConfig } from '~/api/dice';
import { getConnectionList, postAddDingtalk, postAddDiscord, postAddDodo, postAddGocq, postAddGocqSeparate, postAddKook, postAddLagrange, postAddMinecraft, postAddOfficialQQ, postAddOnebot11ReverseWs, postAddRed, postAddSatori, postaddSealChat, postAddSlack, postAddTelegram, postAddWalleQ, } from '~/api/im_connections';
import { getBaseInfo, getHello, getLogFetchAndClear, getPreInfo } from '~/api/others';
import { getSalt, signin } from '~/api/signin';

import type { addImConnectionForm } from '~/components/PageConnectInfoItems.vue'
import type {
  AdvancedConfig,
} from "~/type.d.ts";
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
  mode: 'private' | 'group'
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
  previewInfo: { [key:string]: { version: string, textV2: string, textV1: string, presetExists: boolean, errV1: string, errV2: string } }
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
  justForTest: boolean
  containerMode: boolean
}

export type ResourceType = 'image' | 'audio' | 'video'

export interface Resource {
  type: ResourceType | 'unknown',
  name: string
  ext: string,
  path: string,
  size: number,
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
          content: '海豹已就绪。此界面可视为私聊窗口。\n设置中添加 Master 名为 UI:1001\n即可在此界面使用 master 命令!',
          isSeal: true,
          mode: 'private',
        },
        {
          content: '海豹已就绪。此界面可视为群聊窗口。\n设置中添加 Master 名为 UI:1002\n即可在此界面使用 master 命令!',
          isSeal: true,
          mode: 'group',
        },
        {
          content: '（请注意，当前会话记录在刷新页面后会消失）',
          isSeal: true,
          mode: 'private',
        },
        {
          content: '（请注意，当前会话记录在刷新页面后会消失）',
          isSeal: true,
          mode: 'group',
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
            arch: '',
            justForTest: false,
            containerMode: false,
          },
          customTexts: {},
          customTextsHelpInfo: {},
          previewInfo: {},
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
      await saveCustomText(category,this.curDice.customTexts[category])
    },

    async getPreInfo() {
      const info: {
        testMode: boolean
      } = await getPreInfo()
      return info
    },

    async getBaseInfo() {
      const info = await getBaseInfo()
      if (!document.title.includes('-')) {
        if ((info).extraTitle && (info).extraTitle !== '') {
          document.title = `${(info).extraTitle} - ${document.title}`;
        }
      }
      this.curDice.baseInfo = info;
      return info
    },

    async getCustomText() {
      const data = await getCustomText()
      this.curDice.customTexts = data.texts;
      this.curDice.customTextsHelpInfo = data.helpInfo;
      this.curDice.previewInfo = data.previewInfo;
      return data
    },

    async getImConnections() {
      const info = await getConnectionList()
      this.diceServers[this.index].conns = info;
      return info
    },

    async addImConnection(form: addImConnectionForm ) {
      const {
        accountType,
        nickname,
        account,
        password,
        protocol,
        appVersion,
        token,
        botToken,
        appToken,
        proxyURL,
        reverseProxyUrl,
        reverseProxyCDNUrl,
        url,
        host,
        port,
        appID,
        appSecret,
        clientID,
        robotCode,
        implementation,
        relWorkDir,
        connectUrl,
        accessToken,
        useSignServer,
        signServerConfig,
        signServerUrl,
        signServerVersion,
        reverseAddr,
        onlyQQGuild,
        platform } = form

      let info = null
      switch (accountType) {
        //QQ
        case 0:
          if (implementation === 'gocq') {
            info = await postAddGocq(account,password,protocol,appVersion,useSignServer,signServerConfig)
          } else if (implementation === 'walle-q') {
            info = await postAddWalleQ( account, password, protocol )
          }
          break
        case 1:
          info = await postAddDiscord( token.trim(), proxyURL, reverseProxyUrl, reverseProxyCDNUrl )
          break
        case 2:
          info = await postAddKook( token.trim() )
          break
        case 3:
          info = await postAddTelegram(token.trim(),proxyURL)
          break
        case 4:
          info = await postAddMinecraft(url)
          break
        case 5:
          info = await postAddDodo(clientID.trim(),token.trim())
          break
        case 6: {
          // onebot11 正向
          let realUrl:string = connectUrl.trim()
          if (!realUrl.startsWith('ws://') && !realUrl.startsWith('wss://')) {
            realUrl = `ws://${realUrl}`
          }
          info = await postAddGocqSeparate(relWorkDir, realUrl, accessToken, account)
          break
        }
        case 7:
          info = await postAddRed( host, port, token)
          break
        case 8:
          info = await postAddDingtalk( clientID, token, nickname, robotCode)
          break
        case 9:
          info = await postAddSlack(botToken, appToken)
          break;
        case 10:
          info = await postAddOfficialQQ( Number(appID), appSecret, token,onlyQQGuild)
          break
        case 11:
          info = await postAddOnebot11ReverseWs(account, reverseAddr?.trim())
          break
        case 13:
          info = await postaddSealChat(url.trim(), token.trim())
          break
        case 14:
          info = await postAddSatori(platform, host, port, token)
          break
        case 15:{
          let version = ""
          if (signServerUrl === "sealdice" || signServerUrl === "lagrange") {
            version = signServerVersion
          }
          info = await postAddLagrange( account, signServerUrl, version )
        }
        break
      }
      return info as DiceConnection
    },
    async logFetchAndClear() {
      const info = await getLogFetchAndClear()
      this.curDice.logs = info;
    },

    async diceConfigGet() {
      const info = await getDiceConfig()
      this.curDice.config = info;
    },

    async diceConfigSet(data: DiceConfig) {
      await setDiceConfig(data)
      await this.diceConfigGet()
    },

    async diceAdvancedConfigGet() {
      const info: AdvancedConfig = await getAdvancedConfig()
      return info
    },

    async diceAdvancedConfigSet(data: AdvancedConfig) {
      await setAdvancedConfig(data)
      await this.diceAdvancedConfigGet()
    },

    // async toolOnebot() {
    //   return await backend.post(
    //     urlPrefix + '/tool/onebot',
    //     undefined,
    //     { headers: { token: this.token } }
    //   ) as {
    //     ok: boolean,
    //     ip: string,
    //     errText: string
    //   }
    // },

    async signIn(password: string) {
      try {
        const ret = await signin(password)
        const token = (ret).token
        this.token = token
        localStorage.setItem('t', token)
        this.canAccess = true
      } catch {
        this.canAccess = false
      }
    },
    async trySignIn(): Promise<boolean> {
      this.salt = (await getSalt()).salt
      const token = localStorage.getItem('t')
      try {
        await getHello()
        this.token = token as string
        this.canAccess = true
      } catch (e) {
        this.canAccess = false
        // 试图做一次登录，以获取token
        await this.signIn('defaultSignin')
      }
      return this.token != ''
    },
  }
})
