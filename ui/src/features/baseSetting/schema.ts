import { normalizeBaseSettingSchema, type BaseSettingFieldSchema } from './viewModel';

const officialQQNameCardDefaultTemplate = '<font color="#FF0000" size="1">{card}</font>';

function field(
  id: string,
  key: BaseSettingFieldSchema['key'] | '',
  label: string,
  kind: string,
  keywords: string[],
  options: Partial<BaseSettingFieldSchema> = {}
): BaseSettingFieldSchema {
  return { id, ...(key ? { key } : {}), label, kind, keywords, ...options };
}

// Layout and copy are static UI concerns; the backend owns values and validation.
export const baseSettingSchema = normalizeBaseSettingSchema({
  tabs: [
    {
      id: 'master-notice',
      title: 'Master 与通知',
      groups: [
        {
          id: 'master',
          title: 'Master 管理',
          fields: [
            field(
              'master-unlock-code',
              'masterUnlockCode',
              'Master 解锁码',
              'unlock-code',
              ['master', '解锁码', '抢占', '骰主'],
              {
                readonly: true,
                hint: '用于误操作或被抢占 Master。执行该指令可直接获取 Master 权限并踢掉其他所有人，指令有效期 20 分钟。',
              }
            ),
            field(
              'dice-masters',
              'diceMasters',
              'Master 列表',
              'master-list',
              ['骰主', 'master列表', '管理权限'],
              {
                hint: '单行格式：QQ:12345。也可添加群 ID，在指定的群中所有人均视为拥有 Master 权限。',
              }
            ),
          ],
        },
        {
          id: 'notice-mail',
          title: '通知与邮件',
          fields: [
            field(
              'notice-ids',
              'noticeIds',
              '消息通知列表',
              'notice-targets',
              ['通知列表', '通知ID', '邮件通知目标', '通知分类'],
              {
                hint: '每项可独立启用并选择通知分类。支持 QQ:、QQ-Group:、QQ-CH-Group:、Mail: 等前缀；Mail: 类型不支持禁用或分类筛选。',
              }
            ),
            field('mail-enable', 'mailEnable', '邮箱通知', 'boolean', ['邮件', '邮件通知'], {
              hint: '为处理某些平台通知频繁引起的问题，启用后只会通过邮件进行通知。',
            }),
            field('mail-from', 'mailFrom', '发件邮箱', 'text', ['邮箱', '发件人'], {
              hint: '特别提醒：QQ 号被冻结后，对应账号的邮箱也无法使用。',
            }),
            field(
              'mail-password',
              'mailPassword',
              '邮箱密钥',
              'password',
              ['邮箱密钥', '授权码', '邮箱密码'],
              {
                sensitive: true,
                hint: '不一定是邮箱密码，例如 QQ 邮箱需要在设置界面生成授权码。',
              }
            ),
            field('mail-smtp', 'mailSmtp', 'SMTP 地址', 'text', ['smtp', '邮箱服务器'], {
              hint: '例如 smtp.qq.com。',
            }),
            field('mail-test', '', '发送测试邮件', 'action', ['测试邮件', '验证邮件'], {
              hint: '向通知列表发送测试邮件，便于检查邮件相关配置。',
            }),
          ],
        },
      ],
    },
    {
      id: 'behavior-message',
      title: '行为与消息',
      groups: [
        {
          id: 'behavior',
          title: '行为控制',
          notes: [
            {
              tone: 'info',
              lines: [
                '国密源默认禁用。启用后约增加 32MB 常驻内存，用于国密随机源的熵采集；只有重启海豹才能释放。',
                '选择“国密 SM3”时会弹窗确认；如要关闭，请执行 .randalgo set gm off 后重启海豹。',
                '混合模式只有在国密源已启用时才会包含国密源。',
              ],
            },
          ],
          fields: [
            field('trust-only-mode', 'trustOnlyMode', '私骰模式', 'boolean', ['私骰', '信任用户'], {
              hint: '只允许信任用户拉入群聊。',
            }),
            field(
              'bot-ext-free-switch',
              'botExtFreeSwitch',
              '允许自由开关',
              'boolean',
              ['自由开关', 'bot on', 'ext on'],
              {
                hint: '默认开启。允许任何人执行 bot on/off 和 ext on/off，否则只有邀请者、管理员和 Master 可以操作。',
              }
            ),
            field(
              'text-cmd-trust-only',
              'textCmdTrustOnly',
              '限制 .text 指令',
              'boolean',
              ['.text', '信任限制'],
              {
                hint: '默认开启。开启后只有 Master 和信任用户可使用 .text 指令。如果不了解这个指令，建议保持开启。',
              }
            ),
            field(
              'ignore-unaddressed-bot',
              'ignoreUnaddressedBotCmd',
              '忽略 .bot 裸指令',
              'boolean',
              ['.bot', '裸指令'],
              {
                hint: '默认关闭。开启后在群聊中只有 @骰子 才会响应 .bot。',
              }
            ),
            field(
              'dice-random-mode',
              'diceRandomMode',
              '骰点随机模式',
              'select',
              ['随机源', '随机算法', 'randalgo'],
              {
                options: [
                  { label: 'PCG（默认）', value: 'pcg' },
                  { label: '国密 SM3', value: 'gm' },
                  { label: 'NIST AES-CTR-DRBG', value: 'nist' },
                  { label: '系统 CRNG', value: 'crng' },
                  { label: '混合模式', value: 'hybrid' },
                ],
                confirmMessage:
                  '启用国密随机源后，会常驻约 32MB 内存，用于采集系统熵并保证 SM3 随机数的随机性和不可预测性。这部分内存只有重启海豹才能释放。启用后混合模式会加入国密源，NIST 模式会混入国密种子。确定要启用吗？',
              }
            ),
            field(
              'player-name-wrap',
              'playerNameWrapEnable',
              '<玩家名> 外框',
              'boolean',
              ['玩家名外框', '名称外框', '刷屏防护'],
              {
                hint: '默认开启。建议保持开启，防止用户改名为指令（如 .bot）并利用播报唤醒其他骰子造成刷屏。',
                confirmMessage:
                  '不推荐：用户可能会改名为 .bot/.dismiss 等指令，并利用骰点播报让群内其他骰子刷屏，确定要关闭吗？',
              }
            ),
          ],
        },
        {
          id: 'message',
          title: '消息与日志',
          fields: [
            field(
              'alive-notice-enable',
              'aliveNoticeEnable',
              '存活确认',
              'boolean',
              ['骰狗', '存活确认'],
              {
                hint: '定期向通知列表发送消息，便于骰主知晓存活状态。',
              }
            ),
            field(
              'alive-notice-value',
              'aliveNoticeValue',
              '存活消息间隔',
              'text',
              ['存活间隔', 'cron'],
              {
                hint: '间隔写法请参阅 cron 文档。注意：重启骰子后会重新计时。',
              }
            ),
            field(
              'log-size-notice-enable',
              'logSizeNoticeEnable',
              '日志记录提示',
              'boolean',
              ['日志提示'],
              {
                hint: '每记录 N 条文本后主动发送一条提醒，避免忘记 log off。',
              }
            ),
            field('log-size-notice-count', 'logSizeNoticeCount', '日志提示阈值', 'number', [
              '日志条数阈值',
            ]),
            field('only-log-group', 'onlyLogCommandInGroup', '日志仅记录指令（群聊）', 'boolean', [
              '日志仅记录指令',
              '群聊',
            ]),
            field(
              'only-log-private',
              'onlyLogCommandInPrivate',
              '日志仅记录指令（私聊）',
              'boolean',
              ['日志仅记录指令', '私聊']
            ),
            field(
              'command-prefix',
              'commandPrefix',
              '指令前缀',
              'string-list',
              ['前缀', '指令前导', '命令前缀'],
              {
                hint: '举例：添加 ! 作为指令前缀后，运行 !r 即可骰点。',
              }
            ),
          ],
        },
      ],
    },
    {
      id: 'rate-limit',
      title: '刷屏警告',
      groups: [
        {
          id: 'rate-limit-main',
          title: '刷屏警告设置',
          notes: [
            {
              tone: 'info',
              lines: [
                '每群每用户独立有一个装令牌的桶，桶最多能装“上限”枚令牌。',
                '每次指令视作拿走一枚令牌；当桶里没有令牌时，将触发警告。',
                '桶会按“速率”自动补充令牌；所有更改重启后生效。',
              ],
            },
          ],
          fields: [
            field(
              'rate-limit-enabled',
              'rateLimitEnabled',
              '刷屏警告开关',
              'boolean',
              ['速率限制', '刷屏警告'],
              {
                hint: '默认关闭。开启后会对使用指令过快的用户和群组进行警告，警告后继续使用指令会增加怒气值，仅对 QQ 平台有效。',
              }
            ),
            field(
              'personal-rate',
              'personalReplenishRate',
              '个人速率',
              'text',
              ['个人速率', '每秒', '@every'],
              {
                hint: '支持 @every 3s（每 3 秒一个）或 3（每秒 3 个）格式。',
              }
            ),
            field('personal-burst', 'personalBurst', '个人上限', 'number', ['个人上限'], {
              hint: '个人令牌桶最多可容纳的令牌数。所有速率限制更改将在重启后生效。',
            }),
            field(
              'group-rate',
              'groupReplenishRate',
              '群组速率',
              'text',
              ['群组速率', '每秒', '@every'],
              {
                hint: '支持 @every 3s（每 3 秒一个）或 3（每秒 3 个）格式。',
              }
            ),
            field('group-burst', 'groupBurst', '群组上限', 'number', ['群组上限'], {
              hint: '群组令牌桶最多可容纳的令牌数。所有速率限制更改将在重启后生效。',
            }),
          ],
        },
      ],
    },
    {
      id: 'access-security',
      title: '访问与安全',
      groups: [
        {
          id: 'ui-access',
          title: 'UI 访问控制',
          fields: [
            field(
              'serve-address',
              'serveAddress',
              'UI 界面地址',
              'select',
              ['服务地址', 'UI地址', '端口', '公网'],
              {
                options: [
                  { label: '0.0.0.0:3211', value: '0.0.0.0:3211' },
                  { label: '127.0.0.1:3211', value: '127.0.0.1:3211' },
                ],
                allowCustomValue: true,
                hint: '0.0.0.0:3211 用于服务器，可在公网访问；127.0.0.1:3211 仅限本机访问。修改后需重启生效；开多个海豹时必须修改端口。',
              }
            ),
            field(
              'ui-password',
              'uiPassword',
              'UI 界面密码',
              'password',
              ['UI密码', '后台密码', '登录密码'],
              { sensitive: true, hint: '公网用户务必设置，登录后会自动记住一段时间。' }
            ),
          ],
        },
        {
          id: 'friend-group',
          title: '好友与群组',
          fields: [
            field(
              'friend-add-comment',
              'friendAddComment',
              '加好友验证',
              'text',
              ['好友验证', '加好友'],
              {
                hint: '加好友时必须输入正确的验证信息才能通过。若使用“回答问题并由我确认”，只写问题答案，多个答案用空格隔开；问题答案中不能有空格。',
              }
            ),
            field(
              'refuse-group-invite',
              'refuseGroupInvite',
              '拒绝加入新群',
              'boolean',
              ['拒绝加群', '群邀请'],
              {
                hint: '仅在非强制拉入时生效。',
              }
            ),
            field('tray-tooltip', 'trayTooltip', '托盘提示文本', 'text', ['托盘', '提示文本'], {
              maxLength: 10,
              hint: '自定义前缀最长 10 个字符；留空时使用“海豹TRPG骰点核心”，版本号和端口会自动追加。',
            }),
          ],
        },
      ],
    },
    {
      id: 'platform-special',
      title: '平台配置',
      groups: [
        {
          id: 'qq-official-bot',
          title: 'QQ 官方机器人配置',
          fields: [
            field(
              'officialqq-file-send-base64',
              'officialQQFileSendBase64',
              '以 Base64 发送文件',
              'boolean',
              ['官方QQ', 'Base64', '文件发送'],
              {
                hint: '是否使用 Base64 发送本地/非公网图片或语音文件，仅对 QQ 官方机器人有效。',
              }
            ),
            field(
              'officialqq-use-markdown',
              'officialQQUseMarkdown',
              '使用 Markdown',
              'boolean',
              ['官方QQ', 'Markdown', '消息格式'],
              {
                hint: '是否自动把发送的所有群聊和频道消息转为 Markdown 类型消息，仅对 QQ 官方机器人有效。',
              }
            ),
            field(
              'officialqq-use-name-card',
              'officialQQUseNameCard',
              '回复携带名片',
              'boolean',
              ['官方QQ', '名片', 'sn', 'Markdown'],
              {
                hint: '开启后，官方 QQ 的回复顶部会显示当前玩家的 .sn 名片。需要先启用 Markdown。',
                disabledWhen: {
                  key: 'officialQQUseMarkdown',
                  equals: false,
                },
                disabledHint: '请先启用 Markdown。',
              }
            ),
            field(
              'officialqq-name-card-template',
              'officialQQUseNameCardTemplate',
              '名片渲染模板',
              'text',
              ['官方QQ', '名片', '模板', 'font', 'Markdown'],
              {
                placeholder: officialQQNameCardDefaultTemplate,
                hint: '使用 {card} 代表名片内容。留空或缺少 {card} 时使用默认样式；预览显示 <font> 标签的颜色和字号。',
                maxLength: 500,
                previewSample: '木落 HP10/10',
                previewFallback: officialQQNameCardDefaultTemplate,
              }
            ),
          ],
        },
        {
          id: 'qq-platform',
          title: 'QQ 平台配置',
          fields: [
            field(
              'work-in-qq-channel',
              'workInQQChannel',
              '总开关',
              'boolean',
              ['QQ频道', '频道消息'],
              {
                hint: '如果关闭，将忽略任何频道消息。',
              }
            ),
            field(
              'qq-channel-auto-on',
              'QQChannelAutoOn',
              '自动 bot on',
              'boolean',
              ['自动开启', '频道自动开启'],
              {
                hint: '如果开启，需要在每一个子频道手动 bot off，推荐关闭。',
              }
            ),
            field(
              'qq-channel-log-message',
              'QQChannelLogMessage',
              '记录消息日志',
              'boolean',
              ['频道日志', 'QQ频道日志'],
              {
                hint: '是否记录频道消息到日志。频道较多时可能造成严重刷屏；关闭后仅记录指令，推荐关闭。',
              }
            ),
            field('qq-enable-poke', 'QQEnablePoke', '启用戳一戳', 'boolean', ['戳一戳', 'QQ特性']),
            field(
              'message-delay-range',
              '',
              'QQ 回复延迟(秒)',
              'number-pair',
              ['回复延迟', '消息延迟', 'QQ回复', 'QQ平台'],
              { keys: ['messageDelayRangeStart', 'messageDelayRangeEnd'] }
            ),
          ],
        },
      ],
    },
    {
      id: 'game-extension',
      title: '游戏与扩展',
      groups: [
        {
          id: 'game',
          title: '游戏配置',
          fields: [
            field(
              'default-coc-rule-index',
              'defaultCocRuleIndex',
              'COC 默认房规',
              'text',
              ['coc房规', 'dg'],
              {
                hint: '可设置为 0-5，以及 dg（Delta Green）。',
              }
            ),
            field('max-coc-card-gen', 'maxCocCardGen', 'COC 制卡上限', 'text', ['制卡上限'], {
              hint: '.coc n 中 n 的最大值，1-12 之间，默认 5。',
            }),
            field(
              'max-execute-time',
              'maxExecuteTime',
              '骰点轮数上限',
              'text',
              ['骰点上限', '.r n#'],
              {
                hint: '.r n# 中 n 的最大值，1-25 之间，默认 12。',
              }
            ),
          ],
        },
        {
          id: 'ext-default-settings',
          title: '扩展与扩展指令',
          collapsible: true,
          fields: [
            field(
              'ext-default-settings-field',
              'extDefaultSettings',
              '扩展默认设置',
              'ext-default-settings',
              ['扩展', '默认指令', '自动开启']
            ),
          ],
        },
      ],
    },
    {
      id: 'maintenance-low-frequency',
      title: '维护与低频',
      groups: [
        {
          id: 'quit-inactive',
          title: '自动退群',
          collapsible: true,
          fields: [
            field(
              'quit-inactive-threshold',
              'quitInactiveThreshold',
              '自动退群阈值',
              'number',
              ['不活跃退群', '多少天'],
              {
                hint: '每天凌晨 4 点检查所有群聊的不活跃天数，超过设定天数则自动退群；设置为 0 关闭该功能。',
              }
            ),
            field('quit-inactive-batch-size', 'quitInactiveBatchSize', '退群批次大小', 'number', [
              '退群批次',
            ]),
            field(
              'quit-inactive-batch-wait',
              'quitInactiveBatchWait',
              '退群批次间隔(分)',
              'number',
              ['退群间隔']
            ),
          ],
        },
        {
          id: 'upgrade',
          title: '固件升级',
          collapsible: true,
          notes: [
            {
              tone: 'info',
              lines: [
                '使用指定的压缩包对当前海豹进行覆盖，上传完成后会自动重启海豹。',
                '容器模式下禁止直接更新，请手动拉取最新镜像。',
                '尽量不要从高版本降低到低版本，数据库有可能不兼容。',
              ],
            },
          ],
          fields: [
            field('upgrade-package', '', '固件升级', 'upload', ['升级', '固件包', '上传压缩包']),
          ],
        },
      ],
    },
  ],
});
