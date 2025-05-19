"use strict";
const {Worker, isMainThread, parentPort, workerData} = require('worker_threads');

console.log("worker_threads test");

if (isMainThread) {
    console.log("worker_threads test111111");
    const worker = new Worker(__filename);
    worker.on("message", (data)=>{
        console.log("定时执行", data);
    });
    setInterval(()=>{
        worker.postMessage("updateConfig");
    }, 1000);
}else {
    console.log("worker_threads test2222222");
    parentPort.on("message", async (message)=>{
        if (message === "updateConfig") {
            parentPort.postMessage(11);
            console.log("更新配置");
        }
    });
}