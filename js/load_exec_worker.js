"use strict";
const { Worker, isMainThread, parentPort } = require('worker_threads');
const XLSX = require('xlsx');

if (isMainThread) {
  // 主进程：创建Worker并监听消息
  const worker = new Worker(__filename);
  
  worker.on('message', (data) => {
    // 更新游戏配置（需考虑线程安全）
    gameConfig = data;
    console.log('配置更新完成');
  });
  
  // 定时触发Worker
  setInterval(() => {
    worker.postMessage('updateConfig');
  }, 60 * 1000); // 每分钟更新一次
  
} else {
  // Worker线程：读取Excel文件
  parentPort.on('message', async (message) => {
    if (message === 'updateConfig') {
      try {
        const workbook = await XLSX.readFile('config.xlsx', { async: true });
        const worksheet = workbook.Sheets[workbook.SheetNames[0]];
        const jsonData = XLSX.utils.sheet_to_json(worksheet);
        
        parentPort.postMessage(jsonData);
      } catch (error) {
        console.error('读取配置文件失败:', error);
      }
    }
  });
}