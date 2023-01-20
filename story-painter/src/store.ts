
import { defineStore } from 'pinia'
import { EditorView } from '@codemirror/view';
import axios from 'axios';
import { TextInfo } from './logManager/importers/_logImpoter';

export interface CharItem {
  name: string,
  IMUserId: number | string,
  role: '主持人' | '角色' | '骰子' | '隐藏',
  color: string
}

export interface LogItem {
  id: number;
  nickname: string;
  IMUserId: number | string;
  time: number;
  timeText?: string;
  message: string;
  isDice: boolean;
  commandId: number;
  color?: string;
  role?: string;
  commandInfo?: any;

  // 如果为真，那么只有message有意义，且当作纯文本处理
  isRaw?: boolean;
  index?: number;
}

export const useStore = defineStore('main', {
  state: () => {
    return {
      index: 0,
      editor: null as any as EditorView,
      pcList: [] as CharItem[],
      palette: ['#cb4d68', '#f99252', '#f48cb6', '#9278b9', '#3e80cc', '#84a59d', '#5b5e71'],
      paletteStack: [] as string[],
      items: [] as LogItem[],
      doEditorHighlight: false,

      trgIsAddVoiceMark: false,

      previewElement: HTMLElement,
      _reloadEditor: null as any as (highlight: boolean) => void,

      exportOptions: {
        commandHide: false,
        imageHide: false,
        offTopicHide: false,
        timeHide: false,
        userIdHide: true,
        yearHide: true
      }
    }
  },
  getters: {
    pcMap() {
      let m = new Map<string, CharItem>();
      for (let i of this.pcList) {
        m.set(`${i.name}-${i.IMUserId}`, i);
      }
      return m;
    }
  },
  actions: {
    reloadEditor () {
      this._reloadEditor(this.doEditorHighlight)
    },

    getColor(): string {
      if (this.paletteStack.length === 0) {
        this.paletteStack = [...this.palette]
      }
      return this.paletteStack.shift() as string
    },

    async tryFetchLog(key: string, password: string) {
      const resp = await axios.get('https://weizaima.com/dice/api/log', {
        params: { key, password }
      })
      return resp.data
    },

    /** 移除不使用的pc名字 */
    async pcNameRefresh() {
      const names = new Set();
      const namesAll = new Set();
      const namesToDelete = new Set();
    
      for (let i of this.pcList) {
        namesAll.add(i.name)
      }
    
      for (let i of this.items) {
        names.add(i.nickname)
      }
    
      for (let i of namesAll) {
        if (!names.has(i)) {
          namesToDelete.add(i)
        }
      }
    
      for (let i of namesToDelete) {
        this.tryRemovePC(i as any)
      }
    },

    /** 更新pc列表 */
    async updatePcList(ti: TextInfo) {
      const exists = new Set();
      for (let i of this.pcList) {
        exists.add(i.name);
      }
    
      for (let [k, v] of ti.nicknames) {
        if (!exists.has(k)) {
          this.pcList.push({
            name: k,
            role: '角色',
            color: this.getColor(),
            IMUserId: v,
          });
          exists.add(k);
        }
      }
    },

    async tryRemovePC(name: string) {
      let index = 0
      for (let i of this.pcList) {
        if (i.name === name) {
          this.pcList.splice(index, 1)
          break
        }
        index += 1
      }
    },
  }
})
