'use strict';
// main.js (主进程)
const { Worker } = require('worker_threads');
const path = require('path');

// 游戏配置
let gameConfig = {};

// 创建Worker
const worker = new Worker(path.join(__dirname, 'config-worker.js'));

// 监听配置更新
worker.on('message', (newConfig) => {
  gameConfig = newConfig;
  console.log('配置已更新');
});

// 定时触发配置更新
setInterval(() => {
  worker.postMessage('reload');
}, 5 * 60 * 1000); // 每5分钟更新一次