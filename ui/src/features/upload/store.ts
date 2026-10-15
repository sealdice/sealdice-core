import { defineStore } from 'pinia';
import { createUppyEngine } from './engine';
import { createUploadQueue } from './queue';
import { hashFile } from './fileHash';

// 全应用唯一的上传队列 store（沿用项目 defineStore + appPinia 约定）。
export const useUploadStore = defineStore('upload', () =>
  createUploadQueue({ engine: createUppyEngine(), hashFile })
);
