# LiveClass -- 一款新时代的实时课堂(并非)



[TOC]

------



### 概要：

**liveclass集成了许多技术，运用hertz+kitex的分布式架构构成了一个实时在线直播课堂，同时框架一致，扩展性强，API统一管理，RPC服务文件结构格式化，参数传递使用统一接口，学习成本较低，容易扩展开发。包含用户，直播，实时答题，ai智能助教等部分，未来将不断完善(也许**



- Nginx

- Etcd

- Docker

- Docker-compose

- Mysql

- Redis(存储常修改kv对，热点)

- Redis-stack(向量数据库)

- LUA(保证redis IO操作原子性)

- Hertz

- Kitex

- Eino

- EINO DEV可视化开发

- GORM

- JWT

- WebRTC(仅写出example，没前端配合www)

- RAG

- MCP

- REACT AGENT

- Viper

- RESTFUL API

- WebSocket

- Livego

  

今后仍有等等等等……敬请期待~

------

### 架构图:

暂无





------

### 各模块介绍:

##### 用户模块(userservice):

因为都是较平常的CRUD，暂时只实现了注册(register)、登录(login)、获取用户消息





##### 直播模块(liveservice):

前文提到，一开始了解了WebRTC发现需要前后端配合用WebSocket作为信令服务器来发送sdp和ice候选交换，于是暂且弃用(

直播模块实现较多，会在下面api接口中一一列举

由于设想的直播为后端程序访问livego获取secret key，前端(暂且用obs)推流，使用对应协议观看

所以对于创建直播，关闭直播等具体操作较少，**主要关注在MySQL中直播间与其对应KEY的CRUD和Redis于当前直播间实时参数的原子性操作**



##### 实时答题模块(quizservice):

使用两个部分:http和WebSocket

教师端(http)给具体课程创建问题(选择或者判断)，学生端(WebSocket)实时推送

问题在MySQL中由type:json存储(经由实现的StringArray接口与Scan、Value方法)



##### AI智能助教模块模块(agentservice):

使用eino-examples中eino_assistants的两个json生成(在intialize文件夹中)，再经由结构与具体逻辑的大幅修改而成



(由于自身在AI方面较为薄弱，所以在写此方面的时候调用大量Eino框架内接口以及参考现成实现，，，其实，好像AI编程就是这样hhhh调接口也足够啦)



RAG:使用redis-stack作为向量数据库，参考现有实现而成，目前只实现了较为简单、eino框架内置的markdown分词，由"#"(即一级标题)分成多个切片，再由EmbeddingModel向量化推送到redis-stack，agent使用时使用retriever节点来将其拉取下来再进行相似度查询



MCP:简易使用了SSE以及自身实现的如hash、加减乘除计算器、获取时间等等，以及自带内置duckduckgo搜索引擎，之后可能会用json的格式扩展支持studio



记忆持久化:参考官方实现(因为我自己实现居然多了一百行！天呐ww，之后我要改改写重复逻辑的毛病了……)，使用jsonl和指定窗口读取大小进行IO



第一次进行系统ai编程总结:**Eino还是太高度集成了(，什么都有，导致很多地方写的不算难，也就调调接口，为了较为彻底搞懂此模块啃了半天文档和examples**



工作室中AI巨佬学长的看法:

![IMG_20250523_012143](https://github.com/J1407B-K/liveclass/blob/master/home/kq/GolandProjects/liveclass/images/IMG_20250523_012143.png)