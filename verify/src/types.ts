export type VerifyResult = VerifySuccess | VerifyError

export interface VerifySuccess extends VerifyPayload {
  success: true
}

export interface VerifyError {
  success: false
  err: string
}

export interface VerifyData {
  payload: Uint8Array
  sign: Uint8Array
}

export interface VerifyPayload {
  version: string
  timestamp: number
  platform: string
  uid: string
  username: string
}