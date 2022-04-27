
import { defineStore } from 'pinia'
import { EditorView } from '@codemirror/view';
import axios from 'axios';

export interface CharItem {
  name: string,
  role: '主持人' | '角色' | '骰子' | '隐藏',
  color: string
}

export interface LogItem {
  id: number;
  nickname: string;
  IMUserId: number;
  time: number;
  message: string;
  isDice: boolean;
  commandId: number;
  color?: string;
  commandInfo?: any;
}

export const useStore = defineStore('main', {
  state: () => {
    return {
      index: 0,
      editor: null as any as EditorView,
      pcList: [] as CharItem[],
      palette: ['#cb4d68', '#f99252', '#f48cb6', '#9278b9', '#3e80cc', '#84a59d', '#5b5e71'],
      paletteStack: [] as string[],
      itemById: {} as {[key: string]: LogItem},

      reloadEditor: null as any as () => void,
      reloadEditor2: null as any as () => void,
      exportOptions: {
        commandHide: false,
        imageHide: false,
        offSiteHide: false,
        timeHide: false,
        userIdHide: true,
        yearHide: true
      }
    }
  },
  getters: {
  },
  actions: {
    getColor(): string {
      if (this.paletteStack.length === 0) {
        this.paletteStack = [...this.palette]
      }
      return this.paletteStack.shift() as string
    },
    async customTextSave(category: string) {
    },
    async tryFetchLog(key: string, password: string) {
      const resp = await axios.get('https://weizaima.com/dice/api/log', {
        params: { key, password }
      })
      return resp.data
    },
    async tryAddPcList2(name: string) {
      let isExists = false
      for (let i of this.pcList) {
        if (i.name === name) {
          isExists = true
          break
        }
      }
      if (!isExists) {
        let role = '角色'
        if (name.toLowerCase().startsWith('ob')) {
          role = '隐藏'
        }
        this.pcList.push({
          name: name,
          role: role as any,
          color: this.getColor() // '#000000'
        })
      }
      return !isExists // changed
    },
    async tryAddPcList(item: LogItem) {
      let isExists = false
      for (let i of this.pcList) {
        if (i.name === item.nickname) {
          isExists = true
          break
        }
      }
      if (!isExists) {
        let role = '角色'
        if (item.nickname.toLowerCase().startsWith('ob')) {
          role = '隐藏'
        }
        if (item.isDice) {
          role = '骰子'
        }

        this.pcList.push({
          name: item.nickname,
          role: role as any,
          color: this.getColor()
        })
      }
      return !isExists // changed
    }
  }
})
