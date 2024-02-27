<template>
  <el-affix :offset="60" v-if="modified">
    <div class="tip-danger">
      <el-text type="danger" size="large" tag="strong">内容已修改，不要忘记保存！</el-text>
    </div>
  </el-affix>

  <el-form label-width="120px">
    <h2>Master管理</h2>
    <!-- <el-form-item label="">
      <template #label>
        <div>
          <span>重载脚本(临时)</span>
          <el-tooltip content="用于误操作或被抢占master。执行这条指令可以直接获取master权限并踢掉其他所有人，指令有效期20分钟">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <div>
        <el-button @click="store.scriptReload()">重载脚本</el-button>
      </div>
    </el-form-item> -->

    <el-form-item label="">
      <template #label>
        <div>
          <span>Master解锁码</span>
          <el-tooltip content="用于误操作或被抢占master。执行这条指令可以直接获取master权限并踢掉其他所有人，指令有效期20分钟">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <div>
        <el-button v-if="!isShowUnlockCode" @click="isShowUnlockCode = true">查看</el-button>
        <div v-else style="font-weight: bold;">.master unlock {{config.masterUnlockCode}}</div>
      </div>
    </el-form-item>

    <el-form-item label="Master列表">
      <template #label>
        <div>
          <span>Master列表</span>
          <el-tooltip raw-content content="单行格式: QQ:12345<br>也可添加群ID, 在指定的群中所有人均视为有Master权限">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <template v-if="config.diceMasters && config.diceMasters.length">
        <div v-for="k2, index in config.diceMasters" style="width: 100%; margin-bottom: .5rem;">
          <!-- 这里面是单条修改项 -->
          <div style="display: flex;">
            <div>
              <!-- :suffix-icon="Management" -->
              <el-input v-model="config.diceMasters[index]" :autosize="true"></el-input>
            </div>
            <div style="display: flex; align-items: center; width: 1.3rem; margin-left: 1rem;">
              <el-tooltip :content="index === 0 ? '点击添加项目' : '点击删除你不想要的项'" placement="bottom-start">
                <el-icon>
                  <circle-plus-filled v-if="index == 0" @click="config.diceMasters = addItem(config.diceMasters)" />
                  <circle-close v-else @click="removeItem(config.diceMasters, index)" />
                </el-icon>
              </el-tooltip>
            </div>
          </div>
        </div>
      </template>
      <template v-else>
        <el-icon>
          <circle-plus-filled @click="config.diceMasters = addItem(config.diceMasters)" />
        </el-icon>
      </template>
    </el-form-item>

    <el-form-item label="消息通知列表">
      <template #label>
        <div>
          <span>消息通知列表</span>
          <el-tooltip raw-content content="会对以下消息进行通知:<br>加群邀请、好友邀请、进入群组、被踢出群、被禁言、自动激活、指令退群<br>单行格式: QQ:12345 QQ-Group:12345 Mail:abc@foo.bar<br>通知列表中的QQ号在发件时会自动转换成对应邮箱">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>

      <template v-if="config.noticeIds && config.noticeIds.length">
        <div v-for="k2, index in config.noticeIds" style="width: 100%; margin-bottom: .5rem;">
          <!-- 这里面是单条修改项 -->
          <div style="display: flex;">
            <div>
              <!-- :suffix-icon="Management" -->
              <el-input v-model="config.noticeIds[index]" :autosize="true"></el-input>
            </div>
            <div style="display: flex; align-items: center; width: 1.3rem; margin-left: 1rem;">
              <el-tooltip :content="index === 0 ? '点击添加项目' : '点击删除你不想要的项'" placement="bottom-start">
                <el-icon>
                  <circle-plus-filled v-if="index == 0" @click="config.noticeIds = addItem(config.noticeIds)" />
                  <circle-close v-else @click="removeItem(config.noticeIds, index)" />
                </el-icon>
              </el-tooltip>
            </div>
          </div>
        </div>
      </template>
      <template v-else>
        <el-icon>
          <circle-plus-filled @click="config.noticeIds = addItem(config.noticeIds)" />
        </el-icon>
      </template>
    </el-form-item>

    <el-form-item label="邮箱通知">
        <el-tooltip raw-content content="为处理某些平台通知频繁引起的问题，启用后<b>只会通过邮件</b>进行通知">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
          <el-checkbox label="开启" v-model="config.mailEnable"/>
    </el-form-item>
    <el-form-item label="发件邮箱">
        <el-tooltip raw-content content="特别提醒：QQ号被冻结后对应账号的邮箱也无法使用">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
      <el-input v-model="config.mailFrom" style="width: 12rem"/>
    </el-form-item>
    <el-form-item label="邮箱密钥">
        <el-tooltip raw-content content="请注意不一定是密码，例如 QQ邮箱就需要在设置界面生成授权码">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
      <el-input v-model="config.mailPassword"  style="width: 12rem"/>
    </el-form-item>
    <el-form-item label="SMTP 服务地址">
        <el-tooltip raw-content content="例如 smtp.qq.com">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
      <el-input v-model="config.mailSmtp" style="width: 12rem"/>
    </el-form-item>
    <el-form-item>
      <template #label>
        <div>
          <el-text>发送测试邮件</el-text>
          <el-tooltip content="向通知列表发送测试邮件，便于检查邮件相关配置">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-button type="primary" @click="mailTest">发送</el-button>
    </el-form-item>


    <el-form-item>
      <template #label>
        <div>
          <span>私骰模式</span>
          <el-tooltip raw-content content="只允许信任用户拉入群聊">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-checkbox label="开启" v-model="config.trustOnlyMode"/>
    </el-form-item>

    <el-form-item>
      <template #label>
        <div>
          <span>允许自由开关</span>
          <el-tooltip raw-content content="默认开启。允许任何人执行bot on/off和ext on/off，否则只有邀请者、管理员和master进行操作">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-checkbox label="开启" v-model="config.botExtFreeSwitch"/>
    </el-form-item>
    <el-form-item>
      <template #label>
        <div>
          <span>启用戳一戳</span>
          <el-tooltip raw-content content="默认开启。仅QQ平台，关闭后不回应戳一戳。">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-checkbox label="开启" v-model="config.QQEnablePoke"/>
    </el-form-item>

    <el-form-item>
      <template #label>
        <div>
          <span>限制.text指令</span>
          <el-tooltip raw-content content="默认开启。开启后只有master和信任用户可使用.text指令。如果你不了解这个指令，一定要开启。">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-checkbox label="开启" v-model="config.textCmdTrustOnly"/>
    </el-form-item>

    <el-form-item>
      <template #label>
        <div>
          <span>忽略.bot裸指令</span>
          <el-tooltip raw-content content="默认关闭。开启后在群聊中只有@骰子才会响应.bot">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-checkbox label="开启" v-model="config.ignoreUnaddressedBotCmd"/>
    </el-form-item>

    <el-form-item>
      <template #label>
        <div>
          <span>存活确认(骰狗)</span>
          <el-tooltip raw-content content="定期向通知列表发送消息，以便于骰主知晓存活状态">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-checkbox label="开启" v-model="config.aliveNoticeEnable"/>
    </el-form-item>

    <el-form-item>
      <template #label>
        <div>
          <span>存活消息间隔</span>
          <el-tooltip raw-content content="间隔写法请参阅 <a href='https://pkg.go.dev/github.com/robfig/cron' target='_blank'>cron文档</a>。注意:重启骰子后重新计时。">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-input v-model="config.aliveNoticeValue" style="width: 12rem"></el-input>
    </el-form-item>

<!--    <el-form-item>-->
<!--      <template #label>-->
<!--        <div>-->
<!--          <span>.help 骰主</span>-->
<!--        </div>-->
<!--      </template>-->
<!--      <el-input v-model="config.helpMasterInfo" type="textarea" clearable style="width: 14rem;" />-->
<!--    </el-form-item>-->

<!--    <el-form-item>-->
<!--      <template #label>-->
<!--        <div>-->
<!--          <span>.help 协议</span>-->
<!--        </div>-->
<!--      </template>-->
<!--      <el-input v-model="config.helpMasterLicense" type="textarea" autosize clearable style="width: 14rem;" />-->
<!--    </el-form-item>-->

<!--    <el-form-item>-->
<!--      <template #label>-->
<!--        <div>-->
<!--          <span>.draw keys自定义</span>-->
<!--        </div>-->
<!--      </template>-->
<!--      <div>-->
<!--        <div>-->
<!--          <el-checkbox label="开启" v-model="config.customDrawKeysTextEnable"/>-->
<!--          <el-tooltip raw-content content="启用后替代原本的.draw keys牌组关键字文本">-->
<!--            <el-icon><question-filled /></el-icon>-->
<!--          </el-tooltip>-->
<!--        </div>-->
<!--        <el-input v-model="config.customDrawKeysText" type="textarea" autosize clearable style="width: 14rem;" />-->
<!--      </div>-->
<!--    </el-form-item>-->

    <el-form-item>
      <template #label>
        <div>
          <span>日志记录提示</span>
        </div>
      </template>
      <div>
        <div>
          <el-checkbox label="开启" v-model="config.logSizeNoticeEnable"/>
          <el-tooltip raw-content content="每记录N条文本后，主动发送一条提醒信息，避免忘记log off">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
        <el-input v-model="config.logSizeNoticeCount" type="number" autosize clearable style="width: 14rem;" />
      </div>
    </el-form-item>

<!--    <el-form-item>-->
<!--      <template #label>-->
<!--        <div>-->
<!--          <span>.bot 附加文本</span>-->
<!--        </div>-->
<!--      </template>-->
<!--      <el-input v-model="config.customBotExtraText" type="textarea" autosize clearable style="width: 14rem;" />-->
<!--    </el-form-item>-->

    <h2>刷屏警告</h2>
    <el-form-item>
      <template #label>
        <div>
          <span>刷屏警告开关</span>
          <el-tooltip raw-content content="默认关闭。开启后会对使用指令过快的用户和群组进行警告，警告后继续使用指令会增加怒气值，只对QQ平台有效">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-checkbox label="开启" v-model="config.rateLimitEnabled"/>
    </el-form-item>
    <el-form-item>
      <template #label>
        <div>
          <span>刷屏警告速率</span>
        </div>
      </template>
      个人速率 <el-tooltip raw-content content="支持以下格式: <br/>@every 3s 每3秒一个<br/>3 每秒3个">
        <el-icon><question-filled /></el-icon>
      </el-tooltip>
      <el-input v-model="config.personalReplenishRate" clearable style="width: 14rem;" /> <br/>
    </el-form-item>

    <el-form-item>
      个人上限 <el-input-number v-model="config.personalBurst" :min="1" :step="1" clearable/>
    </el-form-item>

    <el-form-item>
      群组速率 <el-tooltip raw-content content="支持以下格式: <br/>@every 3s 每3秒一个<br/>3 每秒3个">
        <el-icon><question-filled /></el-icon>
      </el-tooltip>
      <el-input v-model="config.groupReplenishRate" clearable style="width: 14rem;" /> <br/>
    </el-form-item>

    <el-form-item>
      群组上限 <el-input-number v-model="config.groupBurst" :min="1" :step="1" clearable/>
    </el-form-item>

    <el-form-item>
      <el-text>
        <div>
          刷屏警告工作原理如下：
          <ul>
            <li>每群每用户独立有一个装令牌的桶，桶最多能装「上限」枚令牌</li>
            <li>每次指令视作拿走一枚令牌</li>
            <li>当桶里没有令牌时，试图拿走令牌将被阻止（触发警告）</li>
            <li>桶以「速率」自动补充令牌</li>
          </ul>
          处置时，优先惩罚个人刷屏，其次是群组内许多人一起刷屏。
        </div>
        <div>如果您感觉难以理解，为了稳定性还是不要更改比较好！</div>
        <div>对速率限制的所有更改 <strong>重启后生效</strong> </div>
      </el-text>
    </el-form-item>

    <h2>访问控制</h2>
    <el-form-item label="UI界面地址">
      <template #label>
        <div>
          <span>UI界面地址</span>
          <el-tooltip raw-content content="0.0.0.0:3211 主要用于服务器，代表可以在公网中用你的手机和电脑访问 <br>127.0.0.1:3211 主要用于自己的电脑/手机，只能在当前设备上管理海豹<br>注意：重启骰子后生效！<br>另，想开多个海豹必须修改端口号！">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-autocomplete v-model="config.serveAddress" clearable name="setting" :fetch-suggestions="querySearch">
        <template #default="{ item }">
          <div class="value">{{ item.link }}</div>
        </template>
      </el-autocomplete>
    </el-form-item>

    <el-form-item label="UI界面密码">
      <template #label>
        <div>
          <span>UI界面密码</span>
          <el-tooltip content="公网用户一定要加，登录后会自动记住一段时间！">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>

      <el-input type="password" show-password v-model="config.uiPassword" style="width: auto;" />
    </el-form-item>

    <h2>QQ频道设置</h2>
    <el-form-item>
      <template #label>
        <div>
          <span>总开关</span>
          <el-tooltip raw-content content="如果关闭，将忽略任何频道消息">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-checkbox label="开启" v-model="config.workInQQChannel"/>
    </el-form-item>

    <el-form-item>
      <template #label>
        <div>
          <span>自动bot on</span>
          <el-tooltip raw-content content="如果开启，需要在每一个子频道手动bot off，推荐关闭">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-checkbox label="开启" v-model="config.QQChannelAutoOn"/>
    </el-form-item>

    <el-form-item>
      <template #label>
        <div>
          <span>记录消息日志</span>
          <el-tooltip raw-content content="是否记录频道消息到日志，如果频道较多，可能造成严重刷屏。<br>若关闭则仅在日志记录指令，推荐关闭">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-checkbox label="开启" v-model="config.QQChannelLogMessage"/>
    </el-form-item>

    <h2>游戏</h2>
    <el-form-item>
      <template #label>
        <div>
          <span>COC默认房规</span>
          <el-tooltip raw-content content="可设置为0-5，以及dg（DeltaGreen）">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-input v-model="config.defaultCocRuleIndex" clearable style="width: 14rem;" />
    </el-form-item>

    <el-form-item>
      <template #label>
        <div>
          <span>COC制卡上限</span>
          <el-tooltip raw-content content=".coc n中n的最大值，1-12之间，默认5">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-input v-model="config.maxCocCardGen" clearable style="width: 14rem;" />
    </el-form-item>

    <el-form-item>
      <template #label>
        <div>
          <span>骰点轮数上限</span>
          <el-tooltip raw-content content=".r n#中n的最大值，1-25之间，默认12">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-input v-model="config.maxExecuteTime" clearable style="width: 14rem;" />
    </el-form-item>

    <h2>海豹</h2>
    <el-form-item>
      <template #label>
        <span>保护开关</span>
      </template>
      <!-- <el-button v-if="!isShowUnlockCode" @click="isShowUnlockCode = true">查看</el-button> -->
      <el-checkbox label="我已阅读功能描述" v-model="isUploadEnable"/>
    </el-form-item>

    <el-form-item>
      <template #label>
        <div>
          <span>固件升级</span>
          <el-tooltip raw-content content="使用海豹安装包进行覆盖升级">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <!-- <el-button v-if="!isShowUnlockCode" @click="isShowUnlockCode = true">查看</el-button> -->
      <el-upload class="upload" action="" multiple :before-upload="beforeUpload" :file-list="fileList" :disabled="!isUploadEnable">
        <el-button type="primary" :icon="Upload" :disabled="!isUploadEnable">上传压缩包</el-button>
      </el-upload>
    </el-form-item>

    <el-form-item>
      <div>使用指定的压缩包对当前海豹进行覆盖，上传完成后会自动重启海豹。</div>
      <div>请注意尽量不要从高版本降低到低版本，数据库有可能不兼容。</div>
      <div>选择文件确定后请耐心等待反馈。</div>
    </el-form-item>

    <h2>其他</h2>
    <el-form-item label="加好友验证信息">
      <template #label>
        <div>
          <span>加好友验证</span>
          <el-tooltip raw-content content="加好友时必须输入正确的验证信息才能通过<br>注意：若使用“回答问题并由我确认”，只写问题答案，有多个答案用空格隔开：<br>问题1答案 问题2答案<br>注意问题答案中本身不能有空格">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>

      <el-input v-model="config.friendAddComment" type="text" clearable style="width: auto;" :placeholder="'为空则填写任意均可通过'" />
    </el-form-item>

    <el-form-item label="不活跃N天后自动退群">
      <template #label>
        <div>
          <span>自动退群阈值</span>
          <el-tooltip raw-content content="每天凌晨4时检查所有群聊的不活跃天数, 若超过设定天数则自动退群<br>设置为0以关闭该功能">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-input-number v-model="config.quitInactiveThreshold" type="number" style="width: auto;" :placeholder="'不活跃N天后自动退出'" />
    </el-form-item>

    <el-form-item label="退群批次大小">
      <el-input-number v-model="config.quitInactiveBatchSize" type="number" style="width: auto;" />
    </el-form-item>

    <el-form-item label="退群批次间隔(分)">
      <el-input-number v-model="config.quitInactiveBatchWait" type="number" style="width: auto;" />
    </el-form-item>

    <el-form-item label="QQ回复延迟(秒)">
      <el-input v-model="config.messageDelayRangeStart" type="number" style="width: 6rem;" />
      <span style="margin: 0 1rem">-</span>
      <el-input v-model="config.messageDelayRangeEnd" type="number" style="width: 6rem;" />
    </el-form-item>

    <el-form-item>
      <template #label>
        <div>
          <span>{{ `<玩家名>` }}外框</span>
          <el-tooltip raw-content content="默认开启。建议开启，这可以防止用户改名为指令(如.bot)，并利用播报去唤醒其他骰子造成刷屏。">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-checkbox label="开启" v-model="config.playerNameWrapEnable" @click="nameWrapUncheck"/>
    </el-form-item>

    <el-form-item label="日志仅记录指令">
      <el-checkbox label="在群聊中" v-model="config.onlyLogCommandInGroup"/>
      <el-checkbox label="在私聊中" v-model="config.onlyLogCommandInPrivate"/>
    </el-form-item>

    <el-form-item label="拒绝加入新群">
      <el-checkbox label="拒绝加群(仅在非强制拉入时起效)" v-model="config.refuseGroupInvite"/>
    </el-form-item>

    <el-form-item label="自动重登录">
      <template #label>
        <div>
          <span>自动重登录</span>
          <el-tooltip content="当5分钟内连续有两次风控信息，进行重登录(每5分钟最多一次)。现不建议使用">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>
      <el-checkbox label="遭遇风控时自动重登录" v-model="config.autoReloginEnable"/>
    </el-form-item>

    <el-form-item label="指令前缀">
      <template #label>
        <div>
          <span>指令前缀</span>
          <el-tooltip content="举例：添加!作为指令前缀，运行 !r 可以骰点">
            <el-icon><question-filled /></el-icon>
          </el-tooltip>
        </div>
      </template>

      <template v-if="config.commandPrefix && config.commandPrefix.length">
        <div v-for="k2, index in config.commandPrefix" style="width: 100%; margin-bottom: .5rem;">
          <!-- 这里面是单条修改项 -->
          <div style="display: flex;">
            <div>
              <!-- :suffix-icon="Management" -->
              <el-input v-model="config.commandPrefix[index]" :autosize="true"></el-input>
            </div>
            <div style="display: flex; align-items: center; width: 1.3rem; margin-left: 1rem;">
              <el-tooltip :content="index === 0 ? '点击添加项目' : '点击删除你不想要的项目'" placement="bottom-start">
                <el-icon>
                  <circle-plus-filled v-if="index == 0" @click="config.commandPrefix = addItem(config.commandPrefix)" />
                  <circle-close v-else @click="removeItem(config.commandPrefix, index)" />
                </el-icon>
              </el-tooltip>
            </div>
          </div>
        </div>
      </template>
      <template v-else>
        <el-icon>
          <circle-plus-filled @click="config.commandPrefix = addItem(config.commandPrefix)" />
        </el-icon>
      </template>
    </el-form-item>

    <h2>扩展及扩展指令</h2>
    <div style="padding-left: 1rem;">
      <div v-for="i in config.extDefaultSettings" style="margin-top: 1rem">
        <span>扩展: {{ i.name }}</span>
        <div>
          禁用指令
          <el-button style="margin-left: 0;" :type="v ? 'danger' : ''" size="small" v-for="v,k in i.disabledCommand" @click="i.disabledCommand[k]=!v">{{k}}</el-button>
        </div>
        <div><el-checkbox v-model="i.autoActive">入群自动开启</el-checkbox></div>
      </div>
      <!-- <el-input v-model="config.helpMasterLicense" type="textarea" autosize clearable style="width: auto;" /> -->
    </div>

    <el-form-item label="" style="margin-top: 3rem;" v-if="modified">
      <el-button type="danger" @click="submitGiveup">放弃改动</el-button>
      <el-button type="success" @click="submit">保存设置</el-button>
    </el-form-item>
  </el-form>

</template>

<script lang="ts" setup>
import {
  Location,
  Document,
  Menu as IconMenu,
  Setting,
  CirclePlusFilled,
  CircleClose,
  QuestionFilled,
  BrushFilled,
  Upload
} from '@element-plus/icons-vue'
import { cloneDeep, toNumber } from 'lodash-es';
import { computed, nextTick, onBeforeMount, onBeforeUnmount, ref, watch } from 'vue';
import { useStore } from '~/store';
import { objDiff, passwordHash } from '~/utils';
import { ElMessage, ElMessageBox } from 'element-plus'

const store = useStore()

const config = ref<any>({})
const fileList = ref<any[]>([])

const isShowUnlockCode = ref(false)
const modified = ref(false)
const isUploadEnable = ref(false)

const beforeUpload = async (file: any) => { // UploadRawFile
  let fd = new FormData()
  fd.append('files', file)
  try {
    const resp = await store.diceUploadToUpgrade({ form: fd });
    ElMessage.success('上传完成，程序即将离线');
  } catch (e: any) {
    ElMessage.error(e.toString());
    console.log(e);
  }
}

watch(() => config, (newValue, oldValue) => { //直接监听
  modified.value = true
}, {
  deep: true
});

const addItem = (k: any[]) => {
  if (!k) {
    k = []
  }
  k.push('')
  return k
}

const removeItem = (v: any[], index: number) => {
  v.splice(index, 1)
}

onBeforeMount(async () => {
  await store.diceConfigGet()
  const val = cloneDeep(store.curDice.config)
  if ((!val.diceMasters) || val.diceMasters.length === 0) {
    val.diceMasters = ['']
  }
  if ((!val.commandPrefix) || val.commandPrefix.length === 0) {
    val.commandPrefix = ['']
  }
  config.value = val
  nextTick(() => {
    modified.value = false
  })
})

const submitGiveup = async () => {
  config.value = cloneDeep(store.curDice.config)
  modified.value = false
  nextTick(() => {
    modified.value = false
  })
}

onBeforeUnmount(() => {
  // clearInterval(timerId)
})

const submit = async () => {
  const mod = objDiff(config.value, store.curDice.config)
  if (mod.uiPassword) {
    mod.uiPassword = await passwordHash(store.salt, mod.uiPassword)
  }

  if (mod.diceMasters) {
    mod.diceMasters = cloneDeep(config.value.diceMasters)
  }

  if (mod.commandPrefix) {
    mod.commandPrefix = cloneDeep(config.value.commandPrefix)
  }

  if (mod.noticeIds) {
    mod.noticeIds = cloneDeep(config.value.noticeIds)
  }

  if (mod.extDefaultSettings) {
    mod.extDefaultSettings = cloneDeep(config.value.extDefaultSettings)
  }

  if (mod.logSizeNoticeCount) {
    mod.logSizeNoticeCount = toNumber(mod.logSizeNoticeCount)
  }

  await store.diceConfigSet(mod)
  submitGiveup()
}

interface LinkItem {
  value: string
  link: string
}

const state = ref('')
const links = ref<LinkItem[]>([
  { link: '外网: 0.0.0.0:3211', value: '0.0.0.0:3211' },
  { link: '本机: 127.0.0.1:3211', value: '127.0.0.1:3211' },
])

const querySearch = (queryString: string, cb: any) => {
  const results = queryString
    ? links.value.filter(createFilter(queryString))
    : links.value
  // call callback function to return suggestion objects
  cb(results)
}
const createFilter = (queryString: string) => {
  return (restaurant: LinkItem) => {
    return (
      restaurant.value.toLowerCase().indexOf(queryString.toLowerCase()) === 0
    )
  }
}

const nameWrapUncheck = (v: boolean) => {
  if (!v) {
    // 取消
    ElMessageBox.confirm(
      '不推荐：用户可能会改名为.bot/.dismiss，并利用骰点播报让群内其他骰子刷屏，确定要取消吗？',
      '',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
      }
    ).catch(e => {
      config.value.playerNameWrapEnable = true;
    })
  }
}

const mailTest = async () => {
  const res = await store.diceMailTest()
  if (res.result) {
    ElMessage.success("已尝试发送测试邮件")
  } else {
    ElMessage.error("发送测试邮件失败！" + res.err)
  }
}

</script>

<style scoped lang="scss">
.about {
  background-color: #fff;
  padding: 2rem;
  line-height: 2rem;
  text-align: left;
  box-shadow: 0 2px 4px rgba(0, 0, 0, 0.12), 0 0 6px rgba(0, 0, 0, 0.04)
}

.top-item {
  flex: 1 0 50%;
  flex-grow: 0;
}
</style>
