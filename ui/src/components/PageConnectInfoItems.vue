<template>
  <!-- <div style="position: relative;"> -->
  <Teleport to="#root">
    <div style="position: absolute; right: 40px; bottom: 60px; z-index: 10">
      <!--    <el-button type="primary" class="btn-add" :icon="Plus" circle @click="addOne"></el-button>-->
      <el-button type="primary" class="btn-add" :icon="Plus" circle @click="addOne"></el-button>
    </div>
  </Teleport>
  <!-- </div> -->

  <div v-if="(!store.curDice.conns) || (store.curDice.conns && store.curDice.conns.length === 0)">
    <span style="vertical-align: middle;">似乎还没有账号，</span>
    <el-link style="font-size: 16px; font-weight: bolder;" type="primary" @click="addOne">点我添加一个</el-link>
  </div>

  <div style="display: flex; flex-wrap: wrap;">
    <div v-for="(i, index) in reactive(store.curDice.conns)" style="min-width: 20rem; flex: 1 0 50%; flex-grow: 0;">
      <el-card class="box-card" shadow="hover" style="margin-right: 1rem; margin-bottom: 1rem; position: relative">
        <template #header>
          <div class="card-header">
            <span style="word-break: break-all;">{{ i.nickname || '<"未知">' }}({{ i.userId }})</span>
            <!-- <el-button class="button" type="text"  @click="doModify(i, index)">修改</el-button> -->
            <el-button size="small" type="danger" :icon="Delete" plain @click="doRemove(i)">删除</el-button>
          </div>
        </template>

        <div style="position: absolute; width: 17rem; height: 14rem; background: #fff; z-index: 1;"
          v-if="(i.adapter?.loginState === goCqHttpStateCode.InLoginQrCode) && store.curDice.qrcodes[i.id]">
          <div style="margin-left: 2rem">需要同账号的手机QQ扫码登录:</div>
          <img
            style="margin-left: -3rem; image-rendering: pixelated; width: 10rem; height:10rem; margin-left: 3.5rem; margin-top: 2rem;"
            :src="store.curDice.qrcodes[i.id]" />
        </div>

        <div style="position: absolute; width: 17rem; height: 14rem; background: #fff; z-index: 1;"
          v-if="(i.adapter?.loginState === goCqHttpStateCode.InLoginBar) && i.adapter?.goCqHttpLoginDeviceLockUrl">
          <!-- <div style="position: absolute; width: 17rem; height: 14rem; background: #fff; z-index: 1;"> -->

          <template v-if="i.id === curCaptchaIdSet">
            <div>已提交ticket，正在等待gocqhttp回应</div>
          </template>
          <template v-else>
            <div style="margin-left: 2rem">滑条验证码流程</div>
            <!-- <div><a style="line-break: anywhere;" :href="i.adapter?.goCqHttpLoginDeviceLockUrl" target="_blank">{{ i.adapter?.goCqHttpLoginDeviceLockUrl }}</a></div> -->
            <div><a @click="captchaUrlSet(i, i.adapter?.goCqHttpLoginDeviceLockUrl)" style="line-break: anywhere;"
                href="javascript:void(0)">{{ i.adapter?.goCqHttpLoginDeviceLockUrl }}</a></div>
          </template>
        </div>

        <div style="position: absolute; width: 17rem; height: 18rem; background: #fff; z-index: 1;"
          v-if="(i.adapter?.loginState === goCqHttpStateCode.InLoginVerifyCode)">
          <div style="margin-left: 2rem">短信验证码流程</div>
          <div style="margin-top: 4rem;">
            <el-form label-width="5rem">
              <el-form-item label="验证码">
                <el-input v-model="smsCode"></el-input>
              </el-form-item>
              <el-form-item label="">
                <el-button :disabled="smsCode == ''" type="primary" @click="submitSmsCode(i)">提交</el-button>
              </el-form-item>
            </el-form>
          </div>
        </div>

        <el-form ref="formRef" :model="i" label-width="100px">
          <el-alert v-if="i.platform === 'QQ' && i.protocolType === 'red'" type="error" :closable="false"
                    style="margin-bottom: 1rem;">
            新版 Chronocat（0.2.x 以上）不再提供 red 协议，故海豹将在未来移除该支持，请尽快迁移。
          </el-alert>
          <!-- <el-form-item label="帐号">
            <el-input v-model="i.account"></el-input>
            <div>123456789<el-tag size="small">{{i.platform}}</el-tag></div>
          </el-form-item>

          <el-form-item label="昵称">
            <div>阮鸫</div>
          </el-form-item> -->

          <el-form-item label="状态">
            <el-space>
              <div v-if="i.state === 0"><el-tag type="danger" disable-transitions>断开</el-tag></div>
              <div v-if="i.state === 2"><el-tag type="warning" disable-transitions>连接中</el-tag></div>
              <div v-if="i.state === 1"><el-tag type="success" disable-transitions>已连接</el-tag></div>
              <div v-if="i.state === 3"><el-tag type="danger" disable-transitions>失败</el-tag></div>
              <el-tooltip
                :content="`看到这个标签是因为最近20分钟内有风控警告，将在重新登录后临时消除。触发时间: ` + dayjs.unix(i.adapter?.inPackGoCqHttpLastRestricted).fromNow()"
                v-if="Math.round(new Date().getTime() / 1000) - i.adapter?.inPackGoCqHttpLastRestricted < 30 * 60">
                <el-tag type="warning">风控</el-tag>
              </el-tooltip>
            </el-space>
          </el-form-item>

          <el-form-item label="在线时长">
            <div>{{ i.onlineTotalTime }} 未实现</div>
          </el-form-item>

          <el-form-item label="群组数量">
            <div>{{ i.groupNum }}</div>
          </el-form-item>

          <el-form-item label="累计响应指令">
            <div>{{ i.cmdExecutedNum }}</div>
          </el-form-item>

          <el-form-item label="上次执行指令">
            <div v-if="i.cmdExecutedLastTime">{{ dayjs.unix(i.cmdExecutedLastTime).fromNow() }}</div>
            <div v-else>从未</div>
          </el-form-item>

          <el-form-item label="连接地址" v-if="i.adapter?.connectUrl">
            <!-- <el-input v-model="i.connectUrl"></el-input> -->
            <div>{{ i.adapter?.connectUrl }}</div>
          </el-form-item>

          <el-form-item label="服务地址" v-if="i.adapter?.isReverse">
            <!-- <el-input v-model="i.connectUrl"></el-input> -->
            <div>{{ i.adapter?.reverseAddr }}/ws</div>
          </el-form-item>

          <template v-if="i.platform === 'QQ' && (i.protocolType === 'onebot' || i.protocolType === 'walle-q')">
            <!-- <el-form-item label="忽略好友请求">
              <div>{{i.adapter?.ignoreFriendRequest ? '是' : '否'}}</div>
            </el-form-item> -->

            <el-form-item label="协议" v-if="i.adapter.useInPackGoCqhttp">
              <!-- <el-input v-model="i.connectUrl"></el-input> -->
              <div v-if="i.adapter?.inPackGoCqHttpProtocol === 0">Unset</div>
              <div v-if="i.adapter?.inPackGoCqHttpProtocol === 1">Android</div>
              <div v-if="i.adapter?.inPackGoCqHttpProtocol === 2">Android 手表</div>
              <div v-if="i.adapter?.inPackGoCqHttpProtocol === 3">MacOS</div>
              <div v-if="i.adapter?.inPackGoCqHttpProtocol === 5">iPad</div>
              <div v-if="i.adapter?.inPackGoCqHttpProtocol === 6">AndroidPad</div>
              <!-- <el-button type="primary" class="btn-add" :icon="Plus" circle @click="addOne"></el-button> -->
              <el-button size="small" type="primary" style="margin-left: 1rem" @click="askSetData(i)"
                :icon="Edit"></el-button>
            </el-form-item>
            <el-form-item label="协议版本" v-if="i.adapter.useInPackGoCqhttp">
              <div v-if="i.adapter?.inPackGoCqHttpAppVersion === ''">未指定</div>
              <div v-if="i.adapter?.inPackGoCqHttpAppVersion && i.adapter.inPackGoCqHttpAppVersion !== ''">{{
                i.adapter.inPackGoCqHttpAppVersion }}</div>
            </el-form-item>
            <el-form-item label="协议实现" v-if="i.adapter.useInPackGoCqhttp">
              <!-- <el-input v-model="i.connectUrl"></el-input> -->
              <div v-if="i.adapter?.implementation === 'gocq' || i.adapter?.implementation === ''">Go-Cqhttp</div>
              <div v-if="i.adapter?.implementation === 'walle-q'">Walle-q</div>
              <!-- <el-button type="primary" class="btn-add" :icon="Plus" circle @click="addOne"></el-button> -->
            </el-form-item>
            <el-form-item label="特殊" v-else-if="i.adapter?.isReverse">
              <div>反向WS</div>
            </el-form-item>
            <el-form-item label="特殊" v-else>
              <div>分离部署</div>
            </el-form-item>
          </template>

          <template v-if="i.platform === 'QQ' && i.protocolType === 'red'">
            <el-form-item label="协议">
              <div>[已弃用]Red</div>
            </el-form-item>
            <el-form-item label="协议版本">
              <div>{{ i.adapter?.redVersion || '未知' }}</div>
            </el-form-item>
            <el-form-item label="连接地址">
              <div>{{ i.adapter?.host + ":" + i.adapter?.port }}</div>
            </el-form-item>
          </template>

          <template v-if="i.platform === 'QQ' && i.protocolType === 'official'">
            <el-form-item label="协议">
              <div>[WIP]官方 QQ Bot</div>
            </el-form-item>
            <el-form-item label="AppID">
              <div>{{ i.adapter?.appID }}</div>
            </el-form-item>
          </template>

          <template v-if="i.platform === 'QQ' && i.protocolType === 'onebot'">
            <el-form-item label="其他">
              <el-tooltip content="导出gocq设置，用于转分离部署" placement="top-start">
                <el-button type="" @click="doGocqExport(i)">导出</el-button>
              </el-tooltip>
            </el-form-item>
          </template>

          <template v-if="i.protocolType === 'satori'">
            <el-form-item label="协议">
              <div>[WIP]Satori</div>
            </el-form-item>
            <el-form-item label="平台">
              <div>{{ i.platform }}</div>
            </el-form-item>
          </template>

          <!-- <el-form-item label="密码">
            <el-input type="password" v-model="i.password"></el-input>
          </el-form-item> -->

          <!-- <el-form-item label="启用">
            <el-switch v-model="i.enable"></el-switch>
          </el-form-item> -->

          <!-- <el-form-item label=""> -->
          <div style="display: flex;justify-content: center; margin-bottom: 1rem;">
            <el-button-group>
              <el-tooltip content="如果日志中出现帐号被风控，可以试试这个功能" placement="bottom-start">
                <el-button type="warning" @click="askGocqhttpReLogin(i)">重新登录</el-button>
              </el-tooltip>
              <el-tooltip content="离线/启用此账号，重启骰子后仍将保持离线/启用状态" placement="bottom-start">
                <el-button type="warning" @click="askSetEnable(i, false)" v-if="i.enable">禁用</el-button>
                <el-button type="warning" @click="askSetEnable(i, true)" v-else>启用</el-button>
              </el-tooltip>
            </el-button-group>
          </div>
          <!-- </el-form-item> -->

        </el-form>
      </el-card>
    </div>
  </div>

  <el-dialog v-model="dialogSetDataFormVisible" title="属性修改" :close-on-click-modal="false" :close-on-press-escape="false"
    :show-close="false" class="the-dialog">
    <el-form :model="form">
      <el-form-item label="类型" :label-width="formLabelWidth">
        <div>QQ账号</div>
      </el-form-item>

      <el-form-item label="忽略好友请求" :label-width="formLabelWidth">
        <el-checkbox v-model="form.ignoreFriendRequest">{{ form.ignoreFriendRequest ? '我会登录官方客户端处理好友请求' : '让海豹帮我按照预设答案处理'
        }}</el-checkbox>
      </el-form-item>

      <el-form-item label="协议" :label-width="formLabelWidth" required>
        <el-select v-model="form.protocol">
          <!-- <el-option label="iPad 协议" :value="0"></el-option> -->
          <el-option label="Android 协议 - 稳定协议，建议！" :value="1"></el-option>
          <el-option label="Android 手表协议 - 可共存,但不支持频道/戳一戳" :value="2"></el-option>
          <el-option label="MacOS" :value="3"></el-option>
          <el-option label="iPad" :value="5"></el-option>
          <el-option v-if="form.implementation === 'gocq' || form.implementation === ''" label="AndroidPad - 稳定协议，建议！"
            :value="6"></el-option>
          <!-- <el-option label="MacOS" :value="3"></el-option> -->
        </el-select>
      </el-form-item>

      <el-form-item :label-width="formLabelWidth"
        v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6)">
        <template #label>
          <div style="display: flex; align-items: center;">
            <span>版本</span>
            <el-tooltip content="只有需要升级协议版本时才指定。" style="">
              <el-icon>
                <QuestionFilled />
              </el-icon>
            </el-tooltip>
          </div>
        </template>
        <el-select v-model="form.appVersion">
          <el-option v-for="version of supportedQQVersions" :key="version" :value="version" :label="version || '不指定版本'" />
        </el-select>
      </el-form-item>

      <el-form-item v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6)"
        :label-width="formLabelWidth">
        <template #label>
          <div style="display: flex; align-items: center;">
            <span>签名服务</span>
            <el-tooltip content="如果不知道这是什么，请选择 不使用。允许填写签名服务相关信息。" style="">
              <el-icon>
                <QuestionFilled />
              </el-icon>
            </el-tooltip>
          </div>
        </template>
        <el-radio-group v-model="signConfigType" size="small" @change="signConfigTypeChange">
          <el-radio-button label="none">不使用</el-radio-button>
          <el-radio-button label="simple">简易配置</el-radio-button>
          <el-radio-button label="advanced">高级配置</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item
        v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'simple'"
        label="服务url" :label-width="formLabelWidth">
        <el-input v-model="form.signServerConfig.signServers[0].url" type="string" autocomplete="off"
          placeholder="http://127.0.0.1:13579"></el-input>
      </el-form-item>
      <el-form-item
        v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'simple'"
        label="服务key" :label-width="formLabelWidth">
        <el-input v-model="form.signServerConfig.signServers[0].key" type="string" autocomplete="off"
          placeholder="114514"></el-input>
      </el-form-item>
      <el-form-item
        v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'simple'"
        label="服务鉴权" :label-width="formLabelWidth">
        <el-input v-model="form.signServerConfig.signServers[0].authorization" type="string" autocomplete="off"
          placeholder="Bearer xxxx 未设置可不填"></el-input>
      </el-form-item>

      <el-form-item
        v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'">
        <el-alert type="warning" :closable="false">如果不理解以下配置项，请使用 <strong>简易配置</strong></el-alert>
      </el-form-item>
      <el-form-item
        v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'">
        <el-table :data="form.signServerConfig.signServers" table-layout="auto">
          <el-table-column prop="url" label="服务url">
            <template #default="scope">
              <el-input v-model="scope.row.url" placeholder="http://127.0.0.1:8080" />
            </template>
          </el-table-column>
          <el-table-column prop="key" label="服务key">
            <template #default="scope">
              <el-input v-model="scope.row.key" placeholder="114514" />
            </template>
          </el-table-column>
          <el-table-column prop="authorization" label="服务鉴权">
            <template #default="scope">
              <el-input v-model="scope.row.authorization" placeholder="Bearer xxxx" />
            </template>
          </el-table-column>
          <el-table-column align="right">
            <template #header="scope">
              <el-button size="small" type="primary" @click="handleSignServerAdd">新增一行</el-button>
            </template>
            <template #default="scope">
              <el-button size="small" type="danger" @click="handleSignServerDelete(scope.row.url)">删除</el-button>
            </template>
          </el-table-column>
        </el-table>
      </el-form-item>
      <el-form-item
        v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'"
        :label-width="formLabelWidth">
        <template #label>
          <div style="display: flex; align-items: center;">
            <span>自动切换规则</span>
            <el-tooltip style="">
              <template #content>
                判断签名服务不可用（需要切换）的额外规则<br />
                - 不设置 （此时仅在请求无法返回结果时判定为不可用）<br />
                - 在获取到的 sign 为空 （若选此建议关闭 auto-register，一般为实例未注册但是请求签名的情况）<br />
                - 在获取到的 sign 或 token 为空（若选此建议关闭 auto-refresh-token ）
              </template>
              <el-icon>
                <QuestionFilled />
              </el-icon>
            </el-tooltip>
          </div>
        </template>
        <el-radio-group v-model="form.signServerConfig.ruleChangeSignServer" size="small">
          <el-radio-button :label="0">不设置</el-radio-button>
          <el-radio-button :label="1">sign为空时切换</el-radio-button>
          <el-radio-button :label="2">sign/token为空时切换</el-radio-button>
        </el-radio-group>
      </el-form-item>
      <el-form-item
        v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'"
        :label-width="formLabelWidth">
        <template #label>
          <div style="display: flex; align-items: center;">
            <span>最大尝试次数</span>
            <el-tooltip style="">
              <template #content>
                连续寻找可用签名服务器最大尝试次数<br />
                为 0 时会在连续 3 次没有找到可用签名服务器后保持使用主签名服务器，不再尝试进行切换备用<br />
                否则会在达到指定次数后 <strong>退出</strong> 主程序
              </template>
              <el-icon>
                <QuestionFilled />
              </el-icon>
            </el-tooltip>
          </div>
        </template>
        <el-input-number v-model="form.signServerConfig.maxCheckCount" size="small" :precision="0" :min="0" />
      </el-form-item>
      <el-form-item
        v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'"
        :label-width="formLabelWidth">
        <template #label>
          <div style="display: flex; align-items: center;">
            <span>请求超时时间</span>
            <el-tooltip style="">
              <template #content>
                签名服务请求超时时间(s)
              </template>
              <el-icon>
                <QuestionFilled />
              </el-icon>
            </el-tooltip>
          </div>
        </template>
        <el-input-number v-model="form.signServerConfig.signServerTimeout" size="small" :precision="0" :min="0" />
        <span>&nbsp;秒</span>
      </el-form-item>
      <el-form-item
        v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'"
        :label-width="formLabelWidth">
        <template #label>
          <div style="display: flex; align-items: center;">
            <span>自动注册实例</span>
            <el-tooltip style="">
              <template #content>
                在实例可能丢失（获取到的签名为空）时是否尝试重新注册<br />
                为 true 时，在签名服务不可用时可能每次发消息都会尝试重新注册并签名。<br />
                为 false 时，将不会自动注册实例，在签名服务器重启或实例被销毁后需要重启 go-cqhttp 以获取实例<br />
                否则后续消息将不会正常签名。关闭此项后可以考虑开启签名服务器端 auto_register 避免需要重启<br />
                由于实现问题，当前建议关闭此项，推荐开启签名服务器的自动注册实例
              </template>
              <el-icon>
                <QuestionFilled />
              </el-icon>
            </el-tooltip>
          </div>
        </template>
        <el-switch v-model="form.signServerConfig.autoRegister" style="--el-switch-on-color: #67C23A;" />
      </el-form-item>
      <el-form-item
        v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'"
        :label-width="formLabelWidth">
        <template #label>
          <div style="display: flex; align-items: center;">
            <span>自动刷新token</span>
            <el-tooltip style="">
              <template #content>
                是否在 token 过期后立即自动刷新签名 token（在需要签名时才会检测到，主要防止 token 意外丢失）<br />
                独立于定时刷新
              </template>
              <el-icon>
                <QuestionFilled />
              </el-icon>
            </el-tooltip>
          </div>
        </template>
        <el-switch v-model="form.signServerConfig.autoRefreshToken" style="--el-switch-on-color: #67C23A;" />
      </el-form-item>
      <el-form-item
        v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'"
        :label-width="formLabelWidth">
        <template #label>
          <div style="display: flex; align-items: center;">
            <span>刷新间隔</span>
            <el-tooltip style="">
              <template #content>
                定时刷新 token 间隔时间，单位为分钟, 建议 30~40 分钟, 不可超过 60 分钟<br />
                目前丢失token也不会有太大影响，可设置为 0 以关闭，推荐开启
              </template>
              <el-icon>
                <QuestionFilled />
              </el-icon>
            </el-tooltip>
          </div>
        </template>
        <el-input-number v-model="form.signServerConfig.refreshInterval" size="small" :precision="0" :min="0" />
        <span>&nbsp;分钟</span>
      </el-form-item>

      <small>
        <div>提示: 切换协议后，需要点击重新登录，或.master reboot重启骰子以应用设置</div>
      </small>

    </el-form>

    <template #footer>
      <span class="dialog-footer">
        <el-button @click="dialogSetDataFormVisible = false">取消</el-button>
        <el-button type="primary" @click="doSetData">确定</el-button>
      </span>
    </template>
  </el-dialog>

  <el-dialog v-model="dialogFormVisible" title="帐号登录" :close-on-click-modal="false" :close-on-press-escape="false"
    :show-close="false" class="the-dialog">
    <el-button style="float: right; margin-top: -4rem;" @click="openSocks">辅助工具-13325端口</el-button>
    <template v-if="form.step === 1">
      <el-alert v-if="form.accountType === 7" type="error" :closable="false"
        style="margin-bottom: 1.5rem;">该支持功能不完善，所适配的目标 Chronocat 版本为 0.0.54，低于该版本不建议使用。<br />同时，新版 Chronocat（0.2.x 以上）不再提供 red 协议，海豹也将在未来移除该支持。</el-alert>
      <el-alert v-if="form.accountType === 10" type="warning" :closable="false"
        style="margin-bottom: 1.5rem;">该支持仍处于实验阶段，部分功能尚未完善。<br />同时，受到腾讯官方提供的 API 能力的限制，一些功能暂时无法实现。</el-alert>
      <el-alert v-if="form.accountType === 14" type="warning" :closable="false"
                style="margin-bottom: 1.5rem;">该支持仍处于实验阶段，部分功能尚未完善。<br />- QQ 平台适配目标版本 0.2.x 以上的 Chronocat。</el-alert>

      <el-form :model="form">
        <el-form-item label="账号类型" :label-width="formLabelWidth">
          <el-select v-model="form.accountType">
            <el-option label="QQ(内置gocq)" :value="0"></el-option>
            <el-option label="QQ(onebot11分离部署)" :value="6"></el-option>
            <el-option label="QQ(onebot11反向WS)" :value="11"></el-option>
            <el-option label="[WIP]QQ(官方bot)" :value="10"></el-option>
            <el-option label="[WIP]Satori" :value="14"></el-option>
            <el-option label="[WIP]SealChat" :value="13"></el-option>
            <el-option label="Discord" :value="1"></el-option>
            <el-option label="KOOK(开黑啦)" :value="2"></el-option>
            <el-option label="Telegram" :value="3"></el-option>
            <el-option label="Minecraft服务器(Paper)" :value="4"></el-option>
            <el-option label="Dodo语音" :value="5"></el-option>
            <el-option label="钉钉" :value="8"></el-option>
            <el-option label="Slack" :value="9"></el-option>
            <el-option label="[已弃用]QQ(red协议)" :value="7"></el-option>
          </el-select>
        </el-form-item>

        <el-form-item v-if="form.accountType === 0" label="设备" :label-width="formLabelWidth" required>
          <el-select v-model="form.protocol">
            <!-- <el-option label="Unset" :value="0"></el-option> -->
            <el-option label="Android 协议" :value="1"></el-option>
            <el-option label="Android 手表协议 - 可共存,但不支持频道/戳一戳" :value="2"></el-option>
            <el-option label="MacOS" :value="3"></el-option>
            <el-option label="iPad" :value="5"></el-option>
            <el-option v-if="form.implementation === 'gocq' || form.implementation === ''" label="AndroidPad - 可共存"
              :value="6"></el-option>
            <!-- <el-option label="MacOS" :value="3"></el-option> -->
          </el-select>
        </el-form-item>

        <el-form-item :label-width="formLabelWidth"
          v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6)">
          <template #label>
            <div style="display: flex; align-items: center;">
              <span>版本</span>
              <el-tooltip content="只有需要升级协议版本时才指定。" style="">
                <el-icon>
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>
          <el-select v-model="form.appVersion">
            <el-option v-for="version of supportedQQVersions" :key="version" :value="version"
              :label="version || '不指定版本'" />
          </el-select>
        </el-form-item>
        <!-- <el-form-item v-if="form.accountType === 0" label="协议实现" :label-width="formLabelWidth" required>
          <el-select v-model="form.implementation">
            <el-option label="Go-cqhttp" :value="'gocq'"></el-option>
            <el-option label="Walle-Q" :value="'walle-q'"></el-option>
          </el-select>
        </el-form-item> -->

        <el-form-item v-if="form.accountType === 0" label="账号" :label-width="formLabelWidth" required>
          <el-input v-model="form.account" type="number" autocomplete="off"></el-input>
        </el-form-item>

        <el-form-item v-if="form.accountType === 0" label="密码" :label-width="formLabelWidth">
          <el-input v-model="form.password" type="password" autocomplete="off"></el-input>
          <small>
            <div>提示: 新设备首次登录多半需要手机版扫码，建议先准备好</div>
            <div>能够进行扫码登录（不填写密码即可），但注意扫码登录不支持自动重连。</div>
            <div>如果出现“要求同一WIFI扫码”可以本地登录后备份，复制到服务器上。</div>
            <!-- v-if="form.protocol !== 2"  -->
            <div style="color: #aa4422;">提示: 首次登录时，建议先尝试AndroidPad，如失败，切换使用Android，再失败手表协议。</div>
            <!-- <div v-if="form.protocol !== 1" style="color: #aa4422;">提示: 首次登录时，iPad或者Android手表协议一般都会失败，建议用安卓登录后改协议。</div> -->
          </small>
        </el-form-item>

        <!-- <el-form-item label="附加参数" :label-width="formLabelWidth">
          <template #label>
            <div style="display: flex; align-items: center;">
              <span>附加参数</span>
              <el-tooltip content="默认参数的作用为让gocqhttp在启动时自动更新协议" style="">
                <el-icon>
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>
          <el-input v-model="form.extraArgs" type="string" autocomplete="off"></el-input>
        </el-form-item> -->

        <el-form-item v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6)"
          :label-width="formLabelWidth">
          <template #label>
            <div style="display: flex; align-items: center;">
              <span>签名服务</span>
              <el-tooltip content="如果不知道这是什么，请选择 不使用。允许填写签名服务相关信息。" style="">
                <el-icon>
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>
          <el-radio-group v-model="signConfigType" size="small" @change="signConfigTypeChange">
            <el-radio-button label="none">不使用</el-radio-button>
            <el-radio-button label="simple">简易配置</el-radio-button>
            <el-radio-button label="advanced">高级配置</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item
          v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'simple'"
          label="服务url" :label-width="formLabelWidth">
          <el-input v-model="form.signServerConfig.signServers[0].url" type="string" autocomplete="off"
            placeholder="http://127.0.0.1:8080"></el-input>
        </el-form-item>
        <el-form-item
          v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'simple'"
          label="服务key" :label-width="formLabelWidth">
          <el-input v-model="form.signServerConfig.signServers[0].key" type="string" autocomplete="off"
            placeholder="114514"></el-input>
        </el-form-item>
        <el-form-item
          v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'simple'"
          label="服务鉴权" :label-width="formLabelWidth">
          <el-input v-model="form.signServerConfig.signServers[0].authorization" type="string" autocomplete="off"
            placeholder="Bearer xxxx"></el-input>
        </el-form-item>

        <el-form-item
          v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'">
          <el-alert type="warning" :closable="false">如果不理解以下配置项，请使用 <strong>简易配置</strong></el-alert>
        </el-form-item>
        <el-form-item
          v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'">
          <el-table :data="form.signServerConfig.signServers" table-layout="auto">
            <el-table-column prop="url" label="服务url">
              <template #default="scope">
                <el-input v-model="scope.row.url" placeholder="http://127.0.0.1:8080" />
              </template>
            </el-table-column>
            <el-table-column prop="key" label="服务key">
              <template #default="scope">
                <el-input v-model="scope.row.key" placeholder="114514" />
              </template>
            </el-table-column>
            <el-table-column prop="authorization" label="服务鉴权">
              <template #default="scope">
                <el-input v-model="scope.row.authorization" placeholder="Bearer xxxx" />
              </template>
            </el-table-column>
            <el-table-column align="right">
              <template #header="scope">
                <el-button size="small" type="primary" @click="handleSignServerAdd">新增一行</el-button>
              </template>
              <template #default="scope">
                <el-button size="small" type="danger" @click="handleSignServerDelete(scope.row.url)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </el-form-item>
        <el-form-item
          v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'"
          :label-width="formLabelWidth">
          <template #label>
            <div style="display: flex; align-items: center;">
              <span>自动切换规则</span>
              <el-tooltip style="">
                <template #content>
                  判断签名服务不可用（需要切换）的额外规则<br />
                  - 不设置 （此时仅在请求无法返回结果时判定为不可用）<br />
                  - 在获取到的 sign 为空 （若选此建议关闭 auto-register，一般为实例未注册但是请求签名的情况）<br />
                  - 在获取到的 sign 或 token 为空（若选此建议关闭 auto-refresh-token ）
                </template>
                <el-icon>
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>
          <el-radio-group v-model="form.signServerConfig.ruleChangeSignServer" size="small">
            <el-radio-button :label="0">不设置</el-radio-button>
            <el-radio-button :label="1">sign为空时切换</el-radio-button>
            <el-radio-button :label="2">sign/token为空时切换</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item
          v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'"
          :label-width="formLabelWidth">
          <template #label>
            <div style="display: flex; align-items: center;">
              <span>最大尝试次数</span>
              <el-tooltip style="">
                <template #content>
                  连续寻找可用签名服务器最大尝试次数<br />
                  为 0 时会在连续 3 次没有找到可用签名服务器后保持使用主签名服务器，不再尝试进行切换备用<br />
                  否则会在达到指定次数后 <strong>退出</strong> 主程序
                </template>
                <el-icon>
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>
          <el-input-number v-model="form.signServerConfig.maxCheckCount" size="small" :precision="0" :min="0" />
        </el-form-item>
        <el-form-item
          v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'"
          :label-width="formLabelWidth">
          <template #label>
            <div style="display: flex; align-items: center;">
              <span>请求超时时间</span>
              <el-tooltip style="">
                <template #content>
                  签名服务请求超时时间(s)
                </template>
                <el-icon>
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>
          <el-input-number v-model="form.signServerConfig.signServerTimeout" size="small" :precision="0" :min="0" />
          <span>&nbsp;秒</span>
        </el-form-item>
        <el-form-item
          v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'"
          :label-width="formLabelWidth">
          <template #label>
            <div style="display: flex; align-items: center;">
              <span>自动注册实例</span>
              <el-tooltip style="">
                <template #content>
                  在实例可能丢失（获取到的签名为空）时是否尝试重新注册<br />
                  为 true 时，在签名服务不可用时可能每次发消息都会尝试重新注册并签名。<br />
                  为 false 时，将不会自动注册实例，在签名服务器重启或实例被销毁后需要重启 go-cqhttp 以获取实例<br />
                  否则后续消息将不会正常签名。关闭此项后可以考虑开启签名服务器端 auto_register 避免需要重启<br />
                  由于实现问题，当前建议关闭此项，推荐开启签名服务器的自动注册实例
                </template>
                <el-icon>
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>
          <el-switch v-model="form.signServerConfig.autoRegister" style="--el-switch-on-color: #67C23A;" />
        </el-form-item>
        <el-form-item
          v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'"
          :label-width="formLabelWidth">
          <template #label>
            <div style="display: flex; align-items: center;">
              <span>自动刷新token</span>
              <el-tooltip style="">
                <template #content>
                  是否在 token 过期后立即自动刷新签名 token（在需要签名时才会检测到，主要防止 token 意外丢失）<br />
                  独立于定时刷新
                </template>
                <el-icon>
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>
          <el-switch v-model="form.signServerConfig.autoRefreshToken" style="--el-switch-on-color: #67C23A;" />
        </el-form-item>
        <el-form-item
          v-if="form.accountType === 0 && (form.protocol === 1 || form.protocol === 6) && signConfigType === 'advanced'"
          :label-width="formLabelWidth">
          <template #label>
            <div style="display: flex; align-items: center;">
              <span>刷新间隔</span>
              <el-tooltip style="">
                <template #content>
                  定时刷新 token 间隔时间，单位为分钟, 建议 30~40 分钟, 不可超过 60 分钟<br />
                  目前丢失token也不会有太大影响，可设置为 0 以关闭，推荐开启
                </template>
                <el-icon>
                  <QuestionFilled />
                </el-icon>
              </el-tooltip>
            </div>
          </template>
          <el-input-number v-model="form.signServerConfig.refreshInterval" size="small" :precision="0" :min="0" />
          <span>&nbsp;分钟</span>
        </el-form-item>

        <el-form-item v-if="form.accountType === 6" label="账号" :label-width="formLabelWidth" required>
          <el-input v-model="form.account" type="number" autocomplete="off"></el-input>
        </el-form-item>
        <el-form-item v-if="form.accountType === 6" label="程序目录" :label-width="formLabelWidth">
          <el-input v-model="form.relWorkDir" type="text" autocomplete="off"
            placeholder="gocqhttp的程序目录，如 d:/my-gocqhttp"></el-input>
        </el-form-item>
        <el-form-item v-if="form.accountType === 6" label="连接地址" :label-width="formLabelWidth" required>
          <el-input v-model="form.connectUrl" placeholder="正向WS连接地址，如 ws://localhost:1234" type="text"
            autocomplete="off"></el-input>
        </el-form-item>
        <el-form-item v-if="form.accountType === 6" label="访问令牌" :label-width="formLabelWidth">
          <el-input v-model="form.accessToken" placeholder="gocqhttp配置的access token，没有不用填写" type="text"
            autocomplete="off"></el-input>
        </el-form-item>

        <el-form-item v-if="form.accountType === 11" label="账号" :label-width="formLabelWidth" required>
          <el-input v-model="form.account" type="number" autocomplete="off"></el-input>
        </el-form-item>
        <el-form-item v-if="form.accountType === 11" label="连接地址" :label-width="formLabelWidth" required>
          <el-input v-model="form.reverseAddr" placeholder="反向WS服务地址，如 0.0.0.0:4001 (允许全部IP连入，4001端口)" type="text"
            autocomplete="off"></el-input>
        </el-form-item>

        <el-form-item v-if="form.accountType === 13" label="连接地址" :label-width="formLabelWidth" required>
          <el-input v-model="form.url" placeholder="连接地址，如 ws://127.0.0.1:3212/ws/seal" type="text"
            autocomplete="off"></el-input>
        </el-form-item>
        <el-form-item v-if="form.accountType === 13" label="Token" :label-width="formLabelWidth" required>
          <el-input v-model="form.token" type="text" autocomplete="off" placeholder="填入平台管理界面中获取的token"></el-input>
        </el-form-item>

        <el-form-item v-if="form.accountType === 14" label="平台" :label-width="formLabelWidth" required>
          <el-radio-group v-model="form.platform">
            <el-radio-button label="QQ"/>
          </el-radio-group>
        </el-form-item>
        <el-form-item v-if="form.accountType === 14" label="主机" :label-width="formLabelWidth" required>
          <el-input v-model="form.host" placeholder="Satori 服务的地址，如 127.0.0.1" type="text" autocomplete="off"></el-input>
        </el-form-item>
        <el-form-item v-if="form.accountType === 14" label="端口" :label-width="formLabelWidth" required>
          <el-input-number v-model="form.port" placeholder="如 5500" autocomplete="off"></el-input-number>
        </el-form-item>
        <el-form-item v-if="form.accountType === 14" label="Token" :label-width="formLabelWidth">
          <el-input v-model="form.token" type="text" autocomplete="off" placeholder="填入鉴权 token，没有时无需填写"></el-input>
        </el-form-item>

        <el-form-item v-if="form.accountType === 7" label="主机" :label-width="formLabelWidth" required>
          <el-input v-model="form.host" placeholder="Red 服务的地址，如 127.0.0.1" type="text" autocomplete="off"></el-input>
        </el-form-item>
        <el-form-item v-if="form.accountType === 7" label="端口" :label-width="formLabelWidth" required>
          <el-input-number v-model="form.port" placeholder="如 16530" autocomplete="off"></el-input-number>
        </el-form-item>
        <el-form-item v-if="form.accountType === 7" label="令牌" :label-width="formLabelWidth" required>
          <el-input v-model="form.token" placeholder="Red 服务的 token" type="text" autocomplete="off"></el-input>
        </el-form-item>

        <el-form-item v-if="form.accountType === 10" label="机器人ID" :label-width="formLabelWidth" required>
          <el-input v-model="form.appID" placeholder="填写在开放平台获取的AppID" autocomplete="off" type="number"></el-input>
        </el-form-item>
        <el-form-item v-if="form.accountType === 10" label="机器人令牌" :label-width="formLabelWidth" required>
          <el-input v-model="form.token" placeholder="填写在开放平台获取的Token" type="text" autocomplete="off"></el-input>
        </el-form-item>
        <el-form-item v-if="form.accountType === 10" label="机器人密钥" :label-width="formLabelWidth" required>
          <el-input v-model="form.appSecret" placeholder="填写在开放平台获取的AppSecret" type="text" autocomplete="off"></el-input>
        </el-form-item>
        <el-form-item v-if="form.accountType === 10" label="只在频道使用" :label-width="formLabelWidth" required>
          <el-switch v-model="form.onlyQQGuild" />
        </el-form-item>

        <el-form-item v-if="form.accountType === 10" :label-width="formLabelWidth">
          <small>
            <div>提示: 进入腾讯开放平台创建一个机器人</div>
            <div>https://q.qq.com/#/app/bot</div>
            <div>创建之后进入机器人管理后台，切换到「开发-开发设置」页</div>
            <div>把机器人的相关信息复制并粘贴进来</div>
          </small>
        </el-form-item>

        <el-form-item v-if="form.accountType === 1" label="Token" :label-width="formLabelWidth" required>
          <el-input v-model="form.token" type="string" autocomplete="off"></el-input>
          <small>
            <div>提示: 首先去discord开发者平台创建一个新的Application</div>
            <div>https://discord.com/developers/applications</div>
            <div>点击New Application 创建之后进入应用，然后点bot，Add bot</div>
            <div>然后把Privileged Gateway Intents下面的三个开关打开</div>
            <div>最后把bot的token复制下来粘贴进来</div>
          </small>
        </el-form-item>
        <el-form-item v-if="form.accountType === 1" label="http 代理地址" :label-width="formLabelWidth">
          <el-input v-model="form.proxyURL" type="string" autocomplete="off" placeholder="http://127.0.0.1:7890" />
        </el-form-item>

        <el-form-item v-if="form.accountType === 2" label="Token" :label-width="formLabelWidth" required>
          <el-input v-model="form.token" type="string" autocomplete="off"></el-input>
          <small>
            <div>提示: 进入KOOK开发者平台创建一个新的应用</div>
            <div>https://developer.kookapp.cn/app/index</div>
            <div>点击新建应用 创建之后进入应用，然后点机器人</div>
            <div>把机器人的token复制下来粘贴进来</div>
          </small>
        </el-form-item>

        <el-form-item v-if="form.accountType === 3" label="Token" :label-width="formLabelWidth" required>
          <el-input v-model="form.token" type="string" autocomplete="off"></el-input>
          <small>
            <div>提示: 私聊BotFather(https://t.me/BotFather)</div>
            <div>使用/newbot申请一个新的机器人</div>
            <div>按照指示创建机器人之后,在Bot setting里面把Group privacy里面privacy mode关掉</div>
            <div>把机器人的token复制下来粘贴进来</div>

          </small>
        </el-form-item>
        <el-form-item v-if="form.accountType === 3" label="http 代理地址" :label-width="formLabelWidth">
          <el-input v-model="form.proxyURL" type="string" autocomplete="off" placeholder="http://127.0.0.1:7890" />
        </el-form-item>

        <el-form-item v-if="form.accountType === 4" label="Url" :label-width="formLabelWidth" required>
          <el-input v-model="form.url" type="string" autocomplete="off"></el-input>
          <small>
            <div>提示: 前往 https://github.com/sealdice/sealdice-minecraft/releases/latest </div>
            <div>下载最新的mc插件然后安装在mc服务器中</div>
            <div>按照 ip:端口 的格式写在框里，默认端口8887</div>
            <div>详细的使用说明请阅读Readme (https://github.com/sealdice/sealdice-minecraft#readme)</div>
          </small>
        </el-form-item>

        <el-form-item v-if="form.accountType === 5" label="ClientID" :label-width="formLabelWidth" required>
          <el-input v-model="form.clientID" type="string" autocomplete="off"></el-input>
        </el-form-item>

        <el-form-item v-if="form.accountType === 5" label="Token" :label-width="formLabelWidth" required>
          <el-input v-model="form.token" type="string" autocomplete="off"></el-input>
          <small>
            <div>提示: 前往 Dodo 开发者平台 https://doker.imdodo.com/bot-list </div>
            <div>如果需要提交审核可以写跑团机器人开发</div>
            <div>你的帐号过审后，点击创建应用</div>
            <div>创建完成之后将clientID和Token复制到这两个框中</div>
          </small>
        </el-form-item>

        <el-form-item v-if="form.accountType === 8" label="昵称" :label-width="formLabelWidth">
          <el-input v-model="form.nickname" type="string" autocomplete="off" placeholder="机器人的昵称"></el-input>
        </el-form-item>
        <el-form-item v-if="form.accountType === 8" label="ClientID" :label-width="formLabelWidth" required>
          <el-input v-model="form.clientID" type="string" autocomplete="off"></el-input>
        </el-form-item>
        <el-form-item v-if="form.accountType === 8" label="RobotCode" :label-width="formLabelWidth" required>
          <el-input v-model="form.robotCode" type="string" autocomplete="off"></el-input>
        </el-form-item>

        <el-form-item v-if="form.accountType === 8" label="Token" :label-width="formLabelWidth" required>
          <el-input v-model="form.token" type="string" autocomplete="off"></el-input>
          <small>
            <div>提示: 前往钉钉开发者平台 https://open-dev.dingtalk.com/fe/app </div>
            <div>点击创建应用</div>
            <div>点击 基础信息 - 应用信息</div>
            <div>把 AppKey 复制到 ClientID 内</div>
            <div>把 AppSecret 复制到 Token 内</div>
            <div>创建完成之后点击 应用功能 - 机器人与消息推送 并将机器人配置的开关打开</div>
            <div>请务必确保 推送方式/消息接受模式 都为 Stream 模式</div>
            <div>点击发布后 复制 RobotCode 到 RobotCode 内</div>
          </small>
        </el-form-item>

        <el-form-item v-if="form.accountType === 9" label="AppToken" :label-width="formLabelWidth" required>
          <el-input v-model="form.appToken" type="string" autocomplete="off"></el-input>
        </el-form-item>
        <el-form-item v-if="form.accountType === 9" label="BotToken" :label-width="formLabelWidth" required>
          <el-input v-model="form.botToken" type="string" autocomplete="off"></el-input>
          <small>
            <div>提示: 前往 Slack 开发者平台 https://api.slack.com/apps </div>
            <div>点击 Create an app 选择 From scratch</div>
            <div>按照要求创建 APP 后，点击OAuth & Permissions</div>
            <div>在下方的 Scopes 中，为机器人添加 channels:write 和 im:write</div>
            <div>点击 Install App to Workspace</div>
            <div>随后将 Bot User OAuth Token 复制并粘贴在 Bot Token 内</div>
            <div>点击 Socket Mode</div>
            <div>把 Enable Socket Mode 打开</div>
            <div>点击 Event Subscriptions</div>
            <div>在 Subscribe to bot events 中，添加 app_mention message.groups 和 message.im</div>
            <div>如果要求你 reinstall 按照提示照做</div>
            <div>点击 Basic Information</div>
            <div>在 App-Level Tokens 一栏，点击 Generate Token and Scopes</div>
            <div>弹出的窗口添加 connections:write 命名随意</div>
            <div>随后将生成的 Token 复制到 App Token 内</div>
          </small>
        </el-form-item>
      </el-form>
    </template>
    <template v-else-if="form.step === 2">
      <el-timeline style="min-height: 260px;">
        <el-timeline-item v-for="(activity, index) in activities" :key="index" :type="(activity.type as any)"
          :color="activity.color" :size="(activity.size as any)" :hollow="activity.hollow">
          <div>{{ activity.content }}</div>
          <div v-if="index === 2 && isTestMode">
            <div>欢迎体验海豹骰点核心，展示模式下不提供登录功能，请谅解。</div>
            <div>如需测试指令，请移步“指令测试”界面。</div>
            <div>此外，数据会定期自动重置。</div>
            <div>展示版本未必是最新版，建议您下载体验。</div>
            <el-button style="margin-top: 1rem;" @click="formClose">再会</el-button>
          </div>
          <div v-else-if="index === 2 && curConn.adapter?.loginState === goCqHttpStateCode.InLoginQrCode">
            <div>登录需要扫码验证, 请使用登录此账号的手机QQ扫描二维码以继续登录:</div>
            <img :src="store.curDice.qrcodes[curConn.id]"
              style="width: 20rem; height: 20rem; image-rendering: pixelated;" />
          </div>

          <div
            v-else-if="index === 2 && curConn.adapter?.loginState === goCqHttpStateCode.InLoginDeviceLock && curConn.adapter?.goCqHttpLoginDeviceLockUrl">

            <template v-if="curConn.id === curCaptchaIdSet">
              <div>已提交ticket，正在等待gocqhttp回应</div>
            </template>
            <template v-else>
              <div>账号已开启设备锁，请访问此链接进行验证：</div>
              <div style="line-break: anywhere;">
                <el-link :href="curConn.adapter?.goCqHttpLoginDeviceLockUrl" target="_blank">{{
                  curConn.adapter?.goCqHttpLoginDeviceLockUrl }}</el-link>
              </div>
            </template>

            <div>
              <div>确认验证完成后，点击此按钮：</div>
              <div>
                <!-- :disabled="duringRelogin" -->
                <el-button type="warning" @click="gocqhttpReLogin(curConn)">下一步</el-button>
              </div>
            </div>
          </div>

          <div
            v-else-if="index === 2 && curConn.adapter?.loginState === goCqHttpStateCode.InLoginBar && curConn.adapter?.goCqHttpLoginDeviceLockUrl">

            <template v-if="curConn.id === curCaptchaIdSet">
              <div>已提交ticket，正在等待gocqhttp回应</div>
            </template>
            <template v-else>
              <div>滑条验证码流程，访问以下链接操作:</div>
              <div style="line-break: anywhere;">
                <div><a @click="captchaUrlSet(curConn, curConn.adapter?.goCqHttpLoginDeviceLockUrl)"
                    style="line-break: anywhere;" href="javascript:void(0)">{{ curConn.adapter?.goCqHttpLoginDeviceLockUrl
                    }}</a></div>
                <!-- <el-link :href="curConn.adapter?.goCqHttpLoginDeviceLockUrl" target="_blank">{{curConn.adapter?.goCqHttpLoginDeviceLockUrl}}</el-link> -->
              </div>
            </template>
          </div>

          <div v-else-if="index === 2 && curConn.adapter?.loginState === goCqHttpStateCode.InLoginVerifyCode">
            <!-- <div v-else-if="1"> -->
            <div>短信验证码流程:</div>
            <div style="line-break: anywhere;">
              <el-form label-width="5rem">
                <el-form-item label="手机号">
                  <div>{{ curConn.adapter?.goCqHttpSmsNumberTip }}</div>
                </el-form-item>
                <el-form-item label="验证码">
                  <el-input v-model="smsCode"></el-input>
                </el-form-item>
                <el-form-item label="">
                  <el-button :disabled="smsCode == ''" type="primary" @click="submitSmsCode(curConn)">提交</el-button>
                </el-form-item>
              </el-form>
            </div>
          </div>

          <div v-else-if="index === 2 && (curConn.adapter?.loginState === goCqHttpStateCode.LoginFailed)">
            <div>
              <div>登录失败!可能是以下原因：</div>
              <ul>
                <li>密码错误(注意检查大小写)</li>
                <li>二维码获取失败(日志中会有“二维码错误”)</li>
                <li>扫二维码超时过期(约2分钟)</li>
                <li>海豹发生了故障</li>
              </ul>
              <div>具体请参见日志。如果不出现“确定”按钮，请直接刷新。</div>
              <div>若删除账号重复尝试无果，请回报bug给开发者。</div>
              <el-link href="javascript:window.location.reload()" type="primary">点我前往日志界面</el-link>
            </div>
          </div>
        </el-timeline-item>
      </el-timeline>
    </template>
    <template v-else-if="form.step === 3">
      <el-result icon="success" title="成功啦!" sub-title="现在账号状态应该是“已连接”了，去试一试骰子吧！">
        <!-- <template #extra></template> -->
      </el-result>
    </template>
    <template v-else-if="form.step === 4">
      <el-result icon="success" title="成功啦!" sub-title="操作完成，现在可以进行下一步了">
        <!-- <template #extra></template> -->
      </el-result>
    </template>

    <template #footer>
      <span class="dialog-footer">
        <template v-if="form.step === 1">
          <el-button @click="dialogFormVisible = false">取消</el-button>
          <el-button type="primary" @click="goStepTwo" :disabled="form.accountType === 0 && form.account === '' ||
            (form.accountType === 1 || form.accountType === 2 || form.accountType === 3) && form.token === '' ||
            form.accountType === 4 && form.url === '' ||
            form.accountType === 5 && (form.clientID === '' || form.token === '') ||
            form.accountType === 8 && (form.clientID === '' || form.token === '' || form.robotCode === '') ||
            form.accountType === 6 && (form.account === '' || form.connectUrl === '') ||
            form.accountType === 7 && (form.host === '' || form.port === '' || form.token === '') ||
            form.accountType === 9 && (form.botToken === '' || form.appToken === '') ||
            form.accountType === 11 && (form.account === '' || form.reverseAddr === '') ||
            form.accountType === 13 && (form.token === '' || form.url === '')">
            下一步</el-button>
        </template>
        <template v-if="form.isEnd">
          <el-button @click="formClose">确定</el-button>
        </template>
      </span>
    </template>
  </el-dialog>

  <!-- 滑条验证，需要3000 z-index的原因是 element 的overlay是2012，其移动端页面上是2017，我不知道是不是累加的，所以给一个很大的值 -->
  <div v-show="dialogSlideVisible"
    style="position: fixed; top:0; left: 0; width: 100%; height: 100%; background: rgba(1,1,1,0.7); z-index: 3000;"
    id="slide">
    <iframe id="slideIframe" ref="slideIframe" src="about:blank" rel="noreferrer"
      style="width: 100%; height: 100%;"></iframe>
    <div v-show="slideBottomShow"
      style="position: absolute; bottom: 0; width: 100%; height: 100px; z-index: 10; display: flex; justify-content: center; flex-direction: column; align-items: center;">
      <div style=" margin-bottom: .5rem;"><a style="line-break: anywhere; font-size: .5rem;" :href="slideLink"
          target="_blank">方式2:新页面打开(如无法验证)</a></div>
      <el-button type="" @click="closeCaptchaFrame">关闭，滑条完成后点击</el-button>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { h, reactive, onBeforeMount, onBeforeUnmount, onMounted, ref, nextTick, Ref, computed } from 'vue';
import { useStore, goCqHttpStateCode } from '~/store';
import type { DiceConnection } from '~/store';
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, Edit, QuestionFilled, Delete } from '@element-plus/icons-vue'
import { sleep } from '~/utils'
import * as dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import { urlBase } from '~/backend';

dayjs.extend(relativeTime)

const fullActivities = [
  {
    content: '正在生成虚拟设备信息',
    size: 'large',
    type: 'primary',
    hollow: true,
  },
  {
    content: '正在生成登录配置文件',
    size: 'large',
    color: '#0bbd87',
    hollow: true,
  },
  {
    content: '进行登录……',
    size: 'large',
    flag: true
  },
  {
    content: '完成!',
    type: 'primary',
    hollow: true,
  },

  {
    content: '进行重新登录流程',
    type: 'large',
    hollow: true,
  },
  {
    content: '如果卡在这里不出二维码，可以尝试1分钟后刷新页面，再次点击登录。如果还不行请删除此账号重新添加',
    type: 'large',
    hollow: true,
  },
]
const activities = ref([] as typeof fullActivities)

const store = useStore()
const curCaptchaIdSet = ref(''); // 当前设置了ticket的id

const isRecentLogin = ref(false)
const duringRelogin = ref(false)
const dialogFormVisible = ref(false)
const dialogSetDataFormVisible = ref(false)
const dialogSlideVisible = ref(false)
const formLabelWidth = '120px'
const isTestMode = ref(false)

const slideIframe = ref(null)
const slideLink = ref('')
const slideBottomShow = ref(false)

const curConn = ref({} as DiceConnection);
const curConnId = ref('');
const smsCode = ref('');

let captchaTimer = null as any
const captchaUrlSet = (i: DiceConnection, url: string) => {
  if (slideIframe.value) {
    dialogSlideVisible.value = true
    const el: HTMLIFrameElement = slideIframe.value;
    slideLink.value = url;
    el.src = url;

    const x = new URL(url);
    const key = x.searchParams.get('cap_cd')
    clearTimeout(captchaTimer);

    // window.addEventListener("message", (e) => {
    //   const key = e.data.code;
    let requestURL = `${urlBase}/sd-api/utils/get_token?key=${key}`;
    console.log('code', key);
    document.cookie = "b=" + key + "; path=/;";

    const ticketCheck = async () => {
      const resp = await fetch(requestURL, { method: 'GET', timeout: 240000 } as any);
      const text = await resp.text();
      if (text) {
        console.log('ticket', text);
        if (text === 'FAIL') {
          captchaTimer = setTimeout(ticketCheck, 2000);
          return;
        }
        curCaptchaIdSet.value = i.id;

        submitCaptchaCode(i, text);
        ElMessage({
          type: 'success',
          message: '已自动读取 ticket:' + text,
          duration: 8000,
        })
        setTimeout(() => {
          dialogSlideVisible.value = false;
        }, 500);
        clearTimeout(captchaTimer);
        captchaTimer = null;
        return;
      }
      captchaTimer = setTimeout(ticketCheck, 2000);
    }
    captchaTimer = setTimeout(ticketCheck, 5000);
    // });

    slideBottomShow.value = false;
    setTimeout(() => {
      // 等一小会再出来，防止误触
      slideBottomShow.value = true;
    }, 3000);
  }
}

onMounted(() => {
})

const closeCaptchaFrame = () => {
  clearTimeout(captchaTimer);
  dialogSlideVisible.value = false;
}

const submitCaptchaCode = async (i: DiceConnection, code: string) => {
  store.ImConnectionsCaptchaSet(i, code)
}

const submitSmsCode = async (i: DiceConnection) => {
  console.log(smsCode.value);
  if (!smsCode.value) return;
  const code = smsCode.value;
  smsCode.value = '';
  store.ImConnectionsSmsCodeSet(i, code)
}

const setRecentLogin = () => {
  isRecentLogin.value = true
  // 无用
  // curConn.value.adapter.inPackGoCqHttpRunning = false;
  // curConn.value.adapter.inPackGoCqHttpLoginDeviceLockUrl = '';
  setTimeout(() => {
    isRecentLogin.value = false
  }, 3000)
}

const openSocks = async () => {
  const ret = await store.toolOnebot()
  if (ret.ok) {
    const msg = h('p', null, [
      h('div', null, '将在服务器上开启临时socks5服务，端口13325'),
      h('div', null, '默认持续时长为20分钟'),
      h('div', null, [h('span', null, `可能的公网IP: `), h('span', { style: 'color: teal' }, `${ret.ip}`)]),
      h('div', null, '注: ip不一定对仅供参考'),
      h('div', { style: 'min-height: 1rem' }, ''),
      h('div', null, '请于服务器管理面板放行13325端口，协议TCP'),
      h('div', null, '如果为Windows Server系统，请再额外关闭系统防火墙或设置放行规则.')
    ]);
    ElMessageBox.alert(msg, '开启辅助工具')
  } else {
    const msg = h('p', null, [
      h('div', null, '启动服务失败，或已经启动'),
      h('div', null, [h('span', null, `报错信息: `), h('span', { style: 'color: #9b0d0d' }, `${ret.errText}`)]),
      h('div', null, [h('span', null, `可能的公网IP: `), h('span', { style: 'color: teal' }, `${ret.ip}`)]),
      h('div', null, '注: ip不一定对仅供参考'),
      h('div', { style: 'min-height: 1rem' }, ''),
      h('div', null, '请于服务器管理面板放行13325端口，协议TCP'),
      h('div', null, '如果为Windows Server系统，请再额外关闭系统防火墙或设置放行规则。')
    ]);
    ElMessageBox.alert(msg, '开启辅助工具')
  }
}

const goStepTwo = async () => {
  form.step = 2
  curConnId.value = '';
  setRecentLogin()
  duringRelogin.value = false;

  store.addImConnection(form as any).then((conn) => {
    if ((conn as any).testMode) {
      isTestMode.value = true
    } else {
      curConnId.value = conn.id;
    }
  }).catch(e => {
    console.log(e)
    ElMessageBox.alert('似乎已经添加了这个账号！', '添加失败')
    formClose()
  })
  if (form.accountType > 0) {
    dialogFormVisible.value = false
    form.step = 1
    return
  }
  activities.value = []
  await sleep(500)
  activities.value.push(fullActivities[0])
  await sleep(1000)
  activities.value.push(fullActivities[1])
  await sleep(1000)
  activities.value.push(fullActivities[2])
}

const formClose = async () => {
  curConnId.value = ''
  dialogFormVisible.value = false;
  form.step = 1;
  form.isEnd = false;
}

const setEnable = async (i: DiceConnection, val: boolean) => {
  const ret = await store.getImConnectionsSetEnable(i, val)
  i.enable = ret.enable
  curCaptchaIdSet.value = '';
  ElMessage.success('状态修改完成')
  if (val) {
    setRecentLogin()
    // 若是启用骰子，走登录流程
    curConnId.value = '' // 先改掉这个，以免和当前连接一致，导致被瞬间重置
    nextTick(() => {
      curConnId.value = i.id
    })
    // store.gocqhttpReloginImConnection(i).then(theConn => {
    //   curConnId.value = i.id;
    // })

    // 重复登录时，也打开这个窗口
    activities.value = []
    dialogFormVisible.value = true


    if (i.adapter.useInPackGoCqhttp) {
      form.step = 2
      activities.value.push(fullActivities[4])
      activities.value.push(fullActivities[5])
      activities.value.push(fullActivities[2])
    } else {
      form.step = 4
      form.isEnd = true
    }
  }
}

const askSetData = async (i: DiceConnection) => {
  form.protocol = i.adapter?.inPackGoCqHttpProtocol;
  form.appVersion = i.adapter?.inPackGoCqHttpAppVersion;
  form.ignoreFriendRequest = i.adapter?.ignoreFriendRequest;
  form.useSignServer = i.adapter?.useSignServer;
  form.signServerConfig = i.adapter?.signServerConfig;
  dialogSetDataFormVisible.value = true;
  form.endpoint = i;

  signConfigType.value = i.adapter?.useSignServer === true ? 'simple' : 'none'
}

const doSetData = async () => {
  let param = {
    protocol: form.protocol,
    ignoreFriendRequest: form.ignoreFriendRequest,
  } as {
    protocol: number,
    appVersion: string,
    ignoreFriendRequest: boolean,
    useSignServer?: boolean,
    signServerConfig?: any
  }
  if (form.protocol === 1 || form.protocol === 6) {
    param = {
      ...param,
      appVersion: form.appVersion,
      useSignServer: form.useSignServer,
      signServerConfig: form.signServerConfig,
    }
  }
  const ret = await store.getImConnectionsSetData(form.endpoint, param);
  if (form.endpoint.adapter) {
    form.endpoint.adapter.inPackGoCqHttpProtocol = form.protocol;
  }
  ElMessage.success('修改完成，请手动重新登录');
  dialogSetDataFormVisible.value = false;
}


const askSetEnable = async (i: DiceConnection, val: boolean) => {
  ElMessageBox.confirm(
    '确认修改此账号的在线状态吗？',
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    setEnable(i, val)
  })
}

const askGocqhttpReLogin = async (i: DiceConnection) => {
  duringRelogin.value = false;
  ElMessageBox.confirm(
    '重新登录吗？有可能要再次扫描二维码',
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    gocqhttpReLogin(i)
  })
}

const doGocqExport = async (i: DiceConnection) => {
  duringRelogin.value = false;
  ElMessageBox.confirm(
    '即将下载gocq配置，是否继续？',
    '提示',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    // http://localhost:3211/sd-api/im_connections/gocq_config_download.zip?id=10f576a4-5237-43f6-9086-269a9f9aace5&token=J4JAofWluYsc0YTgUtDuw3eBnVbZyW%232gTG0agA%40aAVRRIFmrTT0w4tEMbVxGdXn%3A0000000063a8664f
    location.href = `${urlBase}/sd-api/im_connections/gocq_config_download.zip?token=${encodeURIComponent(store.token)}&id=${encodeURIComponent(i.id)}`
  })
}

const gocqhttpReLogin = async (i: DiceConnection) => {
  curCaptchaIdSet.value = '';
  setRecentLogin()
  duringRelogin.value = true;
  curConnId.value = ''; // 先改掉这个，以免和当前连接一致，导致被瞬间重置
  if (curConn.value && curConn.value.adapter) {
    curConn.value.adapter.loginState = goCqHttpStateCode.Init;
  }
  store.gocqhttpReloginImConnection(i).then(theConn => {
    curConnId.value = i.id;
  }).finally(() => {
    form.isEnd = true
  })
  // 重复登录时，也打开这个窗口
  activities.value = []
  dialogFormVisible.value = true

  if (i.adapter.useInPackGoCqhttp) {
    form.step = 2
    activities.value.push(fullActivities[4])
    activities.value.push(fullActivities[5])
    activities.value.push(fullActivities[2])
  } else {
    form.step = 4
  }
}

const signConfigType: Ref<'none' | 'simple' | 'advanced'> = ref('none')
const signConfigTypeChange = (value: any) => {
  switch (value) {
    case 'simple':
      form.useSignServer = true
      // 恢复其他配置项的默认值
      form.signServerConfig = {
        signServers: [form?.signServerConfig?.signServers?.[0] ?? { url: '', key: '', authorization: '' }],
        ruleChangeSignServer: 1,
        maxCheckCount: 0,
        signServerTimeout: 60,
        autoRegister: false,
        autoRefreshToken: false,
        refreshInterval: 40
      }
      break
    case 'advanced':
      form.useSignServer = true
      form.signServerConfig = {
        signServers: form.signServerConfig?.signServers ?? [{ url: '', key: '', authorization: '' }],
        ruleChangeSignServer: 1,
        maxCheckCount: 0,
        signServerTimeout: 60,
        autoRegister: false,
        autoRefreshToken: false,
        refreshInterval: 40
      }
      break
    case 'none':
    default:
      form.useSignServer = false
      form.signServerConfig = { signServers: [{ url: '', key: '', authorization: '' }] } as any
      break
  }
}

const handleSignServerAdd = () => {
  form.signServerConfig?.signServers?.push({ url: '', key: '', authorization: '' })
}

const handleSignServerDelete = (url: string) => {
  if (form.signServerConfig?.signServers) {
    form.signServerConfig.signServers = form.signServerConfig.signServers.filter((server) => { return server.url != url })
  }
}

const supportedQQVersions = ref<string[]>([])

const form = reactive({
  accountType: 0,
  step: 1,
  isEnd: false,
  account: '',
  nickname: '',
  password: '',
  protocol: 1,
  appVersion: '',
  implementation: '',
  id: '',
  token: '',
  botToken: '',
  appToken: '',
  proxyURL: '',
  url: '',
  clientID: '',
  robotCode: '',
  ignoreFriendRequest: false,
  extraArgs: '',
  endpoint: null as any as DiceConnection,

  relWorkDir: '',
  accessToken: '',
  connectUrl: '',

  host: '',
  port: undefined,

  appID: undefined,
  appSecret: '',
  onlyQQGuild: true,

  useSignServer: false,
  signServerConfig: {
    signServers: [
      {
        url: '',
        key: '',
        authorization: ''
      }
    ],
    ruleChangeSignServer: 1,
    maxCheckCount: 0,
    signServerTimeout: 60,
    autoRegister: false,
    autoRefreshToken: false,
    refreshInterval: 40
  },
  signServerUrl: '',
  signServerKey: '',

  reverseAddr: ':4001',
  platform: 'QQ',
})

export type addImConnectionForm = typeof form

const addOne = () => {
  dialogFormVisible.value = true
  form.protocol = 6
  form.implementation = 'gocq'
}

let timerId: number

onBeforeMount(async () => {
  await store.getImConnections()
  for (let i of store.curDice.conns || []) {
    delete store.curDice.qrcodes[i.id]
  }

  const versionsRes = await store.getSupportedQQVersions();
  if (versionsRes.result) {
    supportedQQVersions.value = ['', ...versionsRes.versions]
  }

  const lastIndex = {}
  timerId = setInterval(async () => {
    console.log('refresh')
    await store.getImConnections()

    for (let i of store.curDice.conns || []) {
      // 下一轮登录检查，移除二维码
      // if (!lastIndex[i.id]) lastIndex[i.id] = i.adapter?.curLoginIndex;
      // else {
      //   if (lastIndex[i.id] != i.adapter?.curLoginIndex) {
      //     ;
      //   }
      // }

      // 获取二维码
      if (i.adapter?.loginState === goCqHttpStateCode.InLoginQrCode) {
        store.curDice.qrcodes[i.id] = (await store.getImConnectionsQrCode(i)).img
      }

      if (i.id === curConnId.value) {
        curConn.value = i;

        // 登录失败
        if (i.state !== 1 && i.adapter?.loginState === goCqHttpStateCode.LoginFailed) {
          form.isEnd = true;
        }

        // 登录成功
        if (i.state === 1 && i.adapter?.loginState === goCqHttpStateCode.LoginSuccessed) {
          activities.value.push(fullActivities[3])
          await sleep(1000)
          form.step = 3
          form.isEnd = true
        }

        break;
      }
    }

  }, 3000) as any
})

onBeforeUnmount(() => {
  clearInterval(timerId)
})

const doRemove = async (i: DiceConnection) => {
  ElMessageBox.confirm(
    '删除此项帐号，确定吗？（注：删除账号不会影响人物卡和logs等数据）',
    '警告',
    {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    }
  ).then(async () => {
    await store.removeImConnection(i)
    await store.getImConnections()
    ElMessage({
      type: 'success',
      message: '删除成功!',
    })
  })
}

const doModify = () => {
  ElMessage.success('此功能尚未实现……')
}
</script>

<style scoped>
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.btn-add {
  width: 3rem !important;
  height: 3rem !important;
  font-size: 2rem;
  font-weight: bold;
}
</style>

<style>
.the-dialog {
  min-width: 370px;
}
</style>
