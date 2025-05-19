"use strict";
const { program } = require('commander');
const { exec } = require('child_process');
const fs = require('fs');
const path = require('path');

// 配置默认值
const DEFAULT_CONFIG = {
  platform: 'web',
  output: './build',
  compress: true
};

// 解析命令行参数
program
  .option('-p, --platform <platform>', '目标平台（web/android/ios）', DEFAULT_CONFIG.platform)
  .option('-o, --output <path>', '输出路径', DEFAULT_CONFIG.output)
  .option('-c, --compress [boolean]', '是否压缩资源', DEFAULT_CONFIG.compress)
  .parse(process.argv);

const args = program.opts();

// 读取自定义配置文件
const configPath = path.join(process.cwd(), 'pack.config.json');
let userConfig = {};
if (fs.existsSync(configPath)) {
  userConfig = JSON.parse(fs.readFileSync(configPath, 'utf-8'));
}

// 合并配置
const finalConfig = { ...DEFAULT_CONFIG, ...userConfig, ...args };

// 验证平台参数
const validPlatforms = ['web', 'android', 'ios'];
if (!validPlatforms.includes(finalConfig.platform)) {
  console.error(`错误：不支持的平台 ${finalConfig.platform}，请选择 ${validPlatforms.join('/')}`);
  process.exit(1);
}

// 构造Cocos打包命令
const cocosCommand = `cocos build --platform ${finalConfig.platform} --out ${finalConfig.output} ${finalConfig.compress ? '--compress' : ''}`;

console.log(`正在执行打包命令：${cocosCommand}`);

// 执行命令
exec(cocosCommand, (error, stdout, stderr) => {
  if (error) {
    console.error(`打包失败：${error.message}`);
    console.error(`错误输出：${stderr}`);
    process.exit(1);
  }
  console.log(`打包成功！输出路径：${finalConfig.output}`);
  console.log(`标准输出：${stdout}`);
});