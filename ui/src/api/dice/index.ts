import type { AdvancedConfig } from "~/type";
import { createRequest } from "..";

const baseUrl = '/dice/'
const request = createRequest(baseUrl)

export function getDiceConfig() {
    return request<DiceConfig>('get','config/get')
}

export function setDiceConfig(config: DiceConfig) {
    return request('post','config/set',config)
}

export function getAdvancedConfig() {
    return request<AdvancedConfig>('get','config/advanced/get')
}

export function setAdvancedConfig(config:AdvancedConfig) {
    return request('post','config/advanced/set',config)
}

export function postMailTest() {
    return request<{ result: true } | {
        result: false,
        err: string
      }>('post','config/mail_test')
}

export function postExec(message:string, messageType:'private' | 'group') {
    return request('post','exec',{message,messageType})
}

export function postUploadToUpgrade(files: Blob) {
    return request('post','upload_to_upgrade',{files},'formdata')
}

export function getRecentMessage() {
    return request<RecentMsg[]>('get','recentMessage')
}

export function postUpgrade() {
    return request<{text:string}>('post','upgrade')
}


export type DiceConfig = {
    commandPrefix: string[],  // 包含允许的命令前缀符号
    diceMasters: string[],  // 骰子大师的标识（如用户ID）
    noticeIds: string[],  // 通知ID的列表
    onlyLogCommandInGroup: boolean,  // 是否仅在群组中记录命令
    onlyLogCommandInPrivate: boolean,  // 是否仅在私聊中记录命令
    workInQQChannel: boolean,  // 是否启用QQ频道功能
    messageDelayRangeStart: number,  // 消息延迟的最小范围
    messageDelayRangeEnd: number,  // 消息延迟的最大范围
    uiPassword: string,  // UI密码
    helpDocEngineType: number,  // 帮助文档引擎类型
    masterUnlockCode: string,  // 主解锁码
    serveAddress: string,  // 服务器地址
    masterUnlockCodeTime: number,  // 解锁码的时间戳
    logPageItemLimit: number,  // 日志页面的条目限制
    friendAddComment: string,  // 添加好友时的备注
    QQChannelAutoOn: boolean,  // QQ频道是否自动开启
    QQChannelLogMessage: boolean,  // QQ频道是否记录消息
    refuseGroupInvite: boolean,  // 是否拒绝群组邀请
    quitInactiveThreshold: number,  // 退出不活跃状态的阈值
    quitInactiveBatchSize: number,  // 每次退出不活跃的批量大小
    quitInactiveBatchWait: number,  // 批量退出时的等待时间
    defaultCocRuleIndex: string,  // 默认的COC规则索引
    maxExecuteTime: string,  // 最大执行时间
    maxCocCardGen: string,  // 最大COC卡片生成数量
    extDefaultSettings: ExtensionSettings[],  // 扩展插件的默认设置
    botExtFreeSwitch: boolean,  // 机器人扩展是否自由开关
    trustOnlyMode: boolean,  // 是否启用信任模式
    aliveNoticeEnable: boolean,  // 是否启用存活通知
    aliveNoticeValue: string,  // 存活通知的值
    replyDebugMode: boolean,  // 是否启用调试模式回复
    customReplyConfigEnable: boolean,  // 是否启用自定义回复配置
    logSizeNoticeEnable: boolean,  // 是否启用日志大小通知
    logSizeNoticeCount: number,  // 日志大小通知的数量
    textCmdTrustOnly: boolean,  // 仅信任的文本命令
    ignoreUnaddressedBotCmd: boolean,  // 忽略未定向给机器人的命令
    QQEnablePoke: boolean,  // 是否启用QQ戳一戳功能
    playerNameWrapEnable: boolean,  // 是否启用玩家名包裹
    mailEnable: boolean,  // 是否启用邮件功能
    mailFrom: string,  // 邮件发件人
    mailPassword: string,  // 邮件密码
    mailSmtp: string,  // 邮件SMTP服务器地址
    rateLimitEnabled: boolean,  // 是否启用速率限制
    personalReplenishRate: string,  // 个人速率补充频率
    personalBurst: number,  // 个人速率突发上限
    groupReplenishRate: string,  // 群组速率补充频率
    groupBurst: number  // 群组速率突发上限
};

type ExtensionSettings = {
    name: string,  // 插件名称
    autoActive: boolean,  // 是否自动激活
    disabledCommand: { [key: string]: boolean },  // 禁用的命令
    loaded: boolean  // 是否加载
};


type RecentMsg = {
    content: string
    mode: string
}