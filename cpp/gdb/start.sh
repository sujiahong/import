#!/bin/bash

### 下载
curl -o liboptimization_v2.d.so  http://127.0.0.1:38473/static/liboptimization_v2.d.so
### 软连接
ln -sf liboptimization_v2.d.so libggg.so
### 启动
./processing.l config.ini logs pbprocessing