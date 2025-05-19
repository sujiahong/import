"use strict";

// config-worker.js (Worker线程)
const { parentPort } = require('worker_threads');
const XLSX = require('xlsx');

// 处理配置加载请求
parentPort.on('message', async (message) => {
  if (message === 'reload') {
    try {
      const config = await loadExcelConfig('game_config.xlsx');
      parentPort.postMessage(config);
    } catch (error) {
      console.error('配置加载失败:', error);
      parentPort.postMessage(null); // 发送错误通知
    }
  }
});

// 加载Excel配置
async function loadExcelConfig(filePath) {
  const workbook = await XLSX.readFile(filePath);
  const worksheet = workbook.Sheets[workbook.SheetNames[0]];
  return XLSX.utils.sheet_to_json(worksheet);
}