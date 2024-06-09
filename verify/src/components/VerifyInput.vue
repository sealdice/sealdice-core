<script setup lang="ts">
import type { VerifyData, VerifyPayload, VerifyResult } from '@/types'
import { decode as base2048Decode } from '@/utils/base2048'
import { decode as unpack } from 'msgpack-lite'
import dayjs from 'dayjs'
import relativeTime from 'dayjs/plugin/relativeTime'
import 'dayjs/locale/zh-cn'
import { KJUR } from 'jsrsasign'
import publicKey from '../assets/seal_trusted.public.pem?raw'

dayjs.locale('zh-cn')
dayjs.extend(relativeTime)

const sealCode = ref<string>('')
const result = ref<VerifyResult>()
const time = computed(() => {
  return result.value?.success ? dayjs.unix(result.value.timestamp) : undefined
})

function toHexString(byteArray: Uint8Array) {
  return Array.from(byteArray, b => ('0' + (b & 0xFF).toString(16)).slice(-2)).join('')
}

const signVerify = (payload: Uint8Array, signData: Uint8Array): boolean => {
  const data = toHexString(payload)
  const sign = toHexString(signData)

  const sig = new KJUR.crypto.Signature({ alg: 'SHA1withECDSA' })
  sig.init(publicKey.trim())
  sig.updateHex(data)

  return sig.verify(sign)
}

const verify = async () => {
  if (!sealCode.value || sealCode.value.length === 0) {
    return
  }
  if (!sealCode.value.startsWith('SEAL#') && !sealCode.value.startsWith('SEAL%')) {
    result.value = {
      success: false,
      err: '不是可识别的海豹码'
    }
    return
  }

  // 解析
  const code = sealCode.value.slice('SEAL'.length)
  let data: VerifyData
  try {
    let rawData: Uint8Array
    if (code.startsWith('#')) {
      // base64 编码的海豹校验码
      rawData = Uint8Array.from(atob(code.slice(1)), c => c.charCodeAt(0))
    } else {
      // base2048 编码的海豹校验码
      rawData = base2048Decode(code.slice(1))
    }
    data = unpack(rawData)
  } catch (e) {
    console.error('parse error', e)
    result.value = {
      success: false,
      err: '无法解析海豹码，是否复制完全？'
    }
    return
  }

  console.log('parsed', data)
  // 校验
  if (!data.sign || data.sign.length === 0) {
    result.value = {
      success: false,
      err: '该海豹码不是官方发布的海豹生成的！'
    }
    return
  } else {
    const isValid = signVerify(data.payload, data.sign)
    if (!isValid) {
      result.value = {
        success: false,
        err: '该海豹码不是官方发布的海豹生成的！'
      }
      return
    }
  }

  let payload: VerifyPayload = unpack(data.payload)
  result.value = {
    success: true,
    ...payload
  }
}
</script>

<template>
  <div class="break-all">
    <n-input type="textarea" :autosize="{ minRows: 3 }"
             placeholder="请输入生成的神秘海豹码，以「SEAL%」或「SEAL#」开头" v-model:value="sealCode"
             @blur="verify" />
  </div>

  <n-flex v-if="result" vertical align="center"
          class="mt-8 text-base">
    <template v-if="result.success">
      <n-text type="success" class="my-4 text-xl">校验通过</n-text>

      <n-text>由 &lt;{{ result.username }}&gt;({{ result.uid }}) 于 {{ result.platform }} 生成</n-text>

      <n-flex justify="center" align="center">
        <n-text>
          生成时间：{{ time?.format('YYYY-MM-DD HH:mm:ss') }}
        </n-text>
        <n-tag :bordered="false" type="info">{{ time?.fromNow() }}</n-tag>
      </n-flex>

      <n-flex justify="center" align="center">
        <n-text>海豹版本为</n-text>
        <n-tag :bordered="false" type="info">{{ result.version }}</n-tag>
      </n-flex>
    </template>
    <template v-else>
      <n-text type="error" class="text-lg">校验失败！{{ result.err }}</n-text>
    </template>
  </n-flex>
</template>

<style scoped>

</style>