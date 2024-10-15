import { createRequest } from "..";

const baseUrl = '/im_connections/'
const request = createRequest(baseUrl)

export function getConnectionList() {
    return request<DiceConnection[]>('get', 'list')
}

export function getConnectQQVersion() {
    return request<{ result: true, versions: string[] } | { result: false }>('get', 'qq/get_versions')
}

export function postGoCqHttpRelogin(id: string) {
    return request<DiceConnection>('post', 'gocqhttpRelogin', { id }, 'json', { timeout: 65000 })
}

export function postAddGocq(
    account: number,
    password: string,
    protocol: number,
    appVersion: string,
    useSignServer: boolean,
    signServerConfig: SignServerConfig
) {
    return request<DiceConnection>(
        'post', 'addGocq',
        { account, password, protocol, appVersion, useSignServer, signServerConfig }, 'json',
        { timeout: 65000 }
    )
}

export function postAddWalleQ(
    account: number,
    password: string,
    protocol: number
) {
    return request<DiceConnection>('post', 'addWalleQ',
        { account, password, protocol }, 'json',
        { timeout: 65000 }

    )
}

export function postAddDiscord(
    token: string,
    proxyURL: string,
    reverseProxyUrl: string,
    reverseProxyCDNUrl: string
) {
    return request<DiceConnection>('post', 'addDiscord',
        { token, proxyURL, reverseProxyUrl, reverseProxyCDNUrl }, 'json',
        { timeout: 65000 }
    )
}

export function postAddKook(
    token: string,
) {
    return request<DiceConnection>('post', 'addKook',
        { token }, 'json',
        { timeout: 65000 }
    )
}

export function postAddTelegram(
    token: string,
    proxyURL: string,
) {
    return request<DiceConnection>('post', 'addTelegram',
        { token, proxyURL }, 'json',
        { timeout: 65000 }
    )
}

export function postAddMinecraft(url: string) {
    return request<DiceConnection>('post', 'addMinecraft',
        { url }, 'json',
        { timeout: 65000 }
    )
}

export function postAddDodo(clientID: string, token: string) {
    return request<DiceConnection>('post', 'addDodo',
        { clientID, token }, 'json',
        { timeout: 65000 }
    )
}

export function postAddGocqSeparate(
    relWorkDir: string,
    connectUrl: string,
    accessToken: string,
    account: number
) {
    return request<DiceConnection>('post', 'addGocqSeparate',
        { relWorkDir, connectUrl, accessToken, account }, 'json',
        { timeout: 65000 }
    )
}

export function postAddRed(
    host: string,
    port: string,
    token: string
) {
    return request<DiceConnection>('post', 'addRed',
        { host, port, token }, 'json',
        { timeout: 65000 }
    )
}

export function postAddDingtalk(
    clientID: string,
    token: string,
    nickname: string,
    robotCode: string
) {
    return request<DiceConnection>('post', 'addDingtalk',
        { clientID, token, nickname, robotCode }, 'json',
        { timeout: 65000 }
    )
}

export function postAddSlack(
    botToken: string,
    appToken: string
) {
    return request<DiceConnection>('post', 'addSlack',
        { botToken, appToken }, 'json',
        { timeout: 65000 }
    )
}

export function postAddOfficialQQ(
    appID: number,
    appSecret: string,
    token: string,
    onlyQQGuild: boolean
) {
    return request<DiceConnection>('post', 'addOfficialQQ',
        { appID, appSecret, token, onlyQQGuild }, 'json',
        { timeout: 65000 }
    )
}

export function postAddOnebot11ReverseWs(
    account: number,
    reverseAddr?: string
) {
    return request<DiceConnection>('post', 'addOnebot11ReverseWs',
        { account, reverseAddr }, 'json',
        { timeout: 65000 }
    )
}

export function postaddSealChat(
    url: string,
    token: string
) {
    return request<DiceConnection>('post', 'addSealChat',
        { url, token }, 'json',
        { timeout: 65000 }
    )
}

export function postAddSatori(
    platform: string,
    host: string,
    port: string,
    token: string
) {
    return request<DiceConnection>('post', 'addSatori',
        { platform, host, port, token }, 'json',
        { timeout: 65000 }
    )
}

export function postAddLagrange(
    account: number,
    signServerUrl: string,
    signServerVersion: string
) {
    return request<DiceConnection>('post', 'addLagrange',
        { account, signServerUrl, signServerVersion }, 'json',
        { timeout: 65000 }
    )
}

export function postConnectionDel(id: string) {
    return request<DiceConnection>('post', 'del', { id })
}

export function postConnectionQrcode(id: string) {
    return request<{ img: string }>('post', 'qrcode', { id })
}

export function postSmsCodeSet(id: string, code: string) {
    return request('post', 'sms_code_set', { id, code })
}

export function postGoCqCaptchaSet(id: string, code: string) {
    return request('post', 'gocq_captcha_set', { id, code })
}

export function postConnectSetEnable(id: string, enable: boolean) {
    return request<DiceConnection>('post', 'set_enable', { id, enable })
}

export function postConnectSetData(
    id: string,
    { protocol, appVersion, ignoreFriendRequest, useSignServer, signServerConfig }:{ protocol: number, appVersion: string, ignoreFriendRequest: boolean, useSignServer?: boolean, signServerConfig?: SignServerConfig}
) {
    return request<DiceConnection>('post', 'set_data', {
        id,
        protocol, appVersion, ignoreFriendRequest, useSignServer, signServerConfig
    })
}

export function postSetSignServer(
    id: string,
    signServerUrl: ''|"sealdice"|"lagrange",
    w: boolean,
    signServerVersion: string
) {
    return request<{ result: false, err: string } | { result: true, signServerUrl: string, signServerVersion: string }>('post', 'set_sign_server',
        { id, signServerUrl, w, signServerVersion }
    )
}

interface DiceConnection {
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

interface AdapterQQ {
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
    signServerConfig: SignServerConfig;
    redVersion: string;
    host: string;
    port: number;
    appID: number;
    isReverse: boolean;
    reverseAddr: string;
    builtinMode: 'gocq' | 'lagrange'
}
enum goCqHttpStateCode {
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
//   type addImConnectionForm = {
//     accountType: number,
//     step: number,
//     isEnd: false,
//     account: string,
//     nickname: string,
//     password: string,
//     protocol: number,
//     appVersion: string,
//     implementation: string,
//     id: string,
//     token: string,
//     botToken: string,
//     appToken: string,
//     proxyURL: string,
//     reverseProxyUrl: string,
//     reverseProxyCDNUrl: string,
//     url: string,
//     clientID: string,
//     robotCode: string,
//     ignoreFriendRequest: false,
//     extraArgs: string,
//     endpoint: DiceConnection,

//     relWorkDir: string,
//     accessToken: string,
//     connectUrl: string,

//     host: string,
//     port: undefined,

//     appID: undefined,
//     appSecret: string,
//     onlyQQGuild: true,

//     useSignServer: false,
//     signServerConfig: {
//       signServers: [
//         {
//           url: string,
//           key: string,
//           authorization: string
//         }
//       ],
//       ruleChangeSignServer: number,
//       maxCheckCount: number,
//       signServerTimeout: number,
//       autoRegister: false,
//       autoRefreshToken: false,
//       refreshInterval: number
//     },
//     signServerType: number,
//     signServerUrl: string,
//     signServerKey: string,
//     signServerVersion: string,

//     reverseAddr: string,
//     platform: string,
//   }

type SignServerConfig = {
    signServers: ServerConfig[],
    ruleChangeSignServer: number,
    maxCheckCount: number,
    signServerTimeout: number,
    autoRegister: false,
    autoRefreshToken: false,
    refreshInterval: number
}
type ServerConfig = {
    url: string,
    key: string,
    authorization: string
}