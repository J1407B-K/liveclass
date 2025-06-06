# LiveClass -- 一款新时代的实时课堂(并非)

### 概要：

**liveclass集成了许多技术，运用hertz+kitex的分布式架构构成了一个实时在线直播课堂，同时框架一致，扩展性强，API统一管理，RPC服务文件结构格式化，参数传递使用统一接口，学习成本较低，容易扩展开发。包含用户，直播，实时答题，ai智能助教等部分，未来将不断完善(也许，目前Test代码都没写呢hhh**



- Nginx

- Etcd

- Docker

- Docker-compose

- Mysql

- Redis(存储常修改kv对，热点)

- Redis-stack(向量数据库)

- LUA(保证redis IO操作原子性)

- Hertz

- Kitex(采用分布式架构)

- Eino

- EINO DEV可视化开发

- CozeLoop(扣子罗盘)

- GORM

- JWT

- WebRTC

- RAG

- MCP

- REACT AGENT

- Viper

- RESTFUL API

- WebSocket

- Livego

- Kafka

- MongoDB

- 腾讯云COS存储

- Jaeger(链路追踪:localhost:16686)

  ![1](images/1.png)
  
  
  
- Prometheus(指标测量:localhost:9090)

  ![2](images/2.png)

- Grafana(默认username,password均为admin)，可以查看Prometheus数据

![3](images/3.png)

今后仍有等等等等……敬请期待~

------

### 功能实现:(提到的基本都“丐版”实现了嘿嘿)

**基础要求**:

- 采用分布式架构
- 直播功能(livego/**webrtc**)
- 课堂互动: 在线答题
  - 实时题目推送。
  - 学生端提交答案，教师端实时统计结果。
  - 支持选择题、判断题等题型。

**进阶要求**:

- AI Agent智能助教:RAG、MCP
- 业务功能完善:
  - 签到(支持定时关闭)
  - 随机点名
  - 举手提问+师生聊天(本地似乎验证不太了，不过我看流确实是推上去了hhh)
  - 实时文字聊天（敏感词屏蔽）
  - 支持屏幕共享（与直播共享一个逻辑，只不过前端推的流从摄像头变成了屏幕捕捉）
  - 白板协作（后端其实写了对应json存储和获取的路由，本来打算通过最简单的轮询来同步，只不过ai跑的前端有点小问题...）
  - 课堂录制与回放(采用腾讯COS实现，其实本质上不与直播耦合，采用前端直接捕捉整个视频然后传给后端)
  - 课堂数据统计（采用redis，前端调用接口来实现用户的+1/-1 atomically）



------

### 快速开始(以根目录为相对):

```bash
#请先启用docker-compose
cd ./docker-compose 
docker-compose up -d

#api网关
cd ./internal/cmd
go run main.go

#userservice
cd ./internal
go run ./rpc/user(可选--db自动建表)

#liveservice
cd ./internal
go run ./rpc/live(可选--db自动建表)

#quizservice
cd ./internal
go run ./rpc/quiz(可选--db自动建表)

#agentservice
cd ./internal/rpc/agent
go run .

#chatservice
cd ./internal
go run ./rpc/chat

#webrtc_live
cd ./internal
go run ./rpc/webrtc_live(可选--db自动建表)
```



indexer:(更新.md后加入向量数据库)

```bash
cd ./internal/rpc/agent/cmd/indexer/
go run main.go

#md文件放在liveclass/internal/rpc/agent/cmd/indexer/data/memory中
```



------

### 架构图:

![liveclass简易架构图](images/liveclass简易架构图.png)





------

### 各模块介绍:

##### 用户模块(userservice):

因为都是较平常的CRUD，暂时只实现了注册(register)、登录(login)、获取用户消息





##### livego直播模块(liveservice):

前文提到，一开始了解了WebRTC发现需要前后端配合用WebSocket作为信令服务器来发送sdp和ice候选交换，于是暂且弃用(

直播模块实现较多，会在下面api接口中一一列举

由于设想的直播为后端程序访问livego获取secret key，前端(暂且用obs)推流，使用对应协议观看

所以对于创建直播，关闭直播等具体操作较少，**主要关注在MySQL中直播间与其对应KEY的CRUD和Redis于当前直播间实时参数的原子性操作**



##### **webrtc直播板块(webrtc_liveservice):**

使用**webrtc**来开播关播，师生连麦，其余实现同livego



##### 实时答题模块(quizservice):

使用两个部分:http和WebSocket

教师端(http)给具体课程创建问题(选择或者判断)，学生端(WebSocket)实时推送

问题在MySQL中由type:json存储(经由实现的StringArray接口与Scan、Value方法)



##### AI智能助教模块模块(agentservice):

使用eino-examples中eino_assistants的两个json生成(在intialize文件夹中)，再经由结构与具体逻辑的修改而成



(由于自身在AI基础方面较为薄弱，所以在写此方面的时候调用大量Eino框架内接口以及参考现成实现，，，其实，好像AI编程就是这样hhhh调接口也足够啦)



RAG:使用redis-stack作为向量数据库，参考现有实现而成，目前只实现了较为简单、eino框架内置的markdown分词，由"#"(即一级标题)分成多个切片，再由EmbeddingModel向量化推送到redis-stack，agent使用时使用retriever节点来将其拉取下来再进行相似度查询



MCP:简易使用了SSE(支持studio)以及自身实现的如hash、加减乘除计算器、获取时间等等，以及自带内置duckduckgo搜索引擎



记忆持久化:参考官方实现(因为我自己实现居然多了一百行！天呐ww，之后我要改改写重复逻辑的毛病了……)，使用jsonl和指定窗口读取大小进行IO



第一次进行系统ai编程总结:**Eino还是太高度集成了(，什么都有，如果不引用外部东西就不会报错出神秘小bug，导致很多地方写的不算难，也就调调接口，为了较为彻底搞懂此模块啃了半天文档和examples**



AI巨佬学长的看法:

![IMG_20250523_012143](images/IMG_20250523_012143.png)



##### **实时聊天板块**（chatservice）:

通过websocket作为全双工实时通讯，kafka作为消息队列来处理更多并发请求，同时具备指定关键词屏蔽



------

### API文档：

##### Register:

简介:用户注册

- **接口地址**: `POST :80/register`

- **功能说明**: 用户注册接口，接收 JSON 格式的用户名和密码。接口会先在 Redis 与 MySQL 中检查用户是否已存在，密码会使用 bcrypt 加密后存储。

- 参数解释：

  | 参数名称 | 解释                           |
  | -------- | ------------------------------ |
  | username | 用户名称                       |
  | password | 用户密码                       |
  | auth     | 身份，只能是Student或者Teacher |

  

- 请求示例:

  - Headers:

    - `Content-Type: application/json`

  - Body:

    ```
    {
      "username": "alice",
      "password": "123456",
      "auth":"Student"
    }
    ```

- 返回示例:

  - 成功:

    ```
    {
        "resp": {
            "code": 0,
            "msg": "ok",
            "data": "kq3"
        }
    }
    ```

    

  - 错误 (如用户已存在):

    ```
    {
        "resp": {
            "code": 5000,
            "msg": "remote or network error[remote]: biz error: Error 1062 (23000): Duplicate entry 'kq2' for key 'users.uni_users_username'rpc服务错误",
            "data": "nil"
        }
    }
    ```

##### Login

简介：用户登录

- **接口地址**：`POST /login`

- **功能说明**：接收 JSON 格式的用户名和密码，验证通过后返回 JWT 令牌。

- 参数解释：

  | 参数名称 | 解释   |
  | -------- | ------ |
  | username | 用户名 |
  | password | 密码   |

  

- 请求示例：

  - Headers：

    - `Content-Type: application/json`

  - Body：

    ```
    {
      "username": "alice",
      "password": "123456"
    }
    ```

- 返回示例：

  - 成功：

    ```
    {
        "code": 200,
        "expire": "2025-06-01T01:00:22+08:00",
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJleHAiOjE3NDg3MTA4MjIsIm9yaWdfaWF0IjoxNzQ4NjI0NDIyLCJ1c2VyaWQiOiIyIn0.kP5pZNuuCuzVg0IZqkOcVc3AYi5_4-evwYrFDGSDj-k"
    }
    ```

  - 错误（认证失败）：

    ```
    {
        "resp": {
            "code": 5000,
            "msg": "remote or network error[remote]: biz error: crypto/bcrypt: hashedPassword is not the hash of the given passwordrpc服务错误",
            "data": "nil"
        }
    }{
        "code": 401,
        "message": "鉴权失败"
    }
    ```

------

##### Get User Info

简介：获取当前用户信息

- **接口地址**：`GET /userinfo`

- **功能说明**：通过username，获取info

- 参数解释：

  | 参数名称 | 解释     |
  | -------- | -------- |
  | username | 用户名称 |

  

- 请求示例:

  - Headers:
    - `Content-Type: Content-Type: multipart/form-data`
  - Body:
    - username: kq1

- 返回示例：

  - 成功：

    ```
    {
        "resp": {
            "code": 0,
            "msg": "ok",
            "data": "2/kq1/Teacher"（id/username/auth）
        }
    }
    ```

  - 错误：

    ```
    {
        "resp": {
            "code": 5002,
            "msg": "remote or network error[remote]: biz error: record not founduserRpc error",
            "data": ""
        }
    }
    ```

------

##### Is Student in Lesson

简介：判断学生是否已加入指定课堂

- **接口地址**：`GET /is_stu_in_lesson`

- **功能说明**：根据 学生ID和课程 ID，判断当前学生是否已加入该课堂。

- 参数解释：

  | 参数       | 解释   |
  | ---------- | ------ |
  | student_id | 学生id |
  | lesson_id  | 课程id |

  

- 请求示例：

  - Headers:
    - `Content-Type:multipart/form-data`
  - Body:
    - student_id:1
    - lesson_id:1

- 返回示例：

  - 成功：

    ```
    {
        "resp": {
            "code": 0,
            "msg": "ok",
            "data": "in"
        }
    }
    ```

  - 错误：

    ```
    {
        "resp": {
            "code": 0,
            "msg": "ok",
            "data": "not_in"
        }
    }
    ```

------

##### Create LiveGo 直播

简介：创建一个新的 LiveGo 直播

- **接口地址**：`POST /create_live`

- **功能说明**：向 LiveGo 服务提交创建直播请求，返回推流地址等信息。

- 参数解释：

  | 参数名称    | 解释     |
  | ----------- | -------- |
  | lesson_name | 课程 ID  |
  | description | 课程描述 |

  

- 请求示例：

  - Headers：

    - `Content-Type:multipart/form-data`
    - `Authorization: Bearer <jwt_token>`

  - Body：

    lesson_name:class1

    description:"for 2025"

- 返回示例：

  - 成功：

    ```
    {
        "resp": {
            "code": 0,
            "msg": "ok",
            "data": "OeLEhdw3KPG965gZ8SAIE1BYWi7BQeLh7uwKKQ4Hq63t1hjJ$rtmp play:rtmp://localhost:1935/live/class1_kq1$flv play:http://127.0.0.1:7001/live/class1_kq1.flv$hls play:http://127.0.0.1:7002/live/class1_kq1.m3u8"
        }
    }
    ```

  - 错误：

    ```
    {
        "code": 401,
        "message": "auth header is empty"
    }
    ```

------



##### Close LiveGo 直播

简介：关闭指定的 LiveGo 直播

- **接口地址**：`DELETE /close_live`

- **功能说明**：删除mysql中课程信息

- 参数解释：

  | 参数名称 | 解释     |
  | -------- | -------- |
  | livename | 课程名字 |

  

- 请求示例：

  - Headers：
    - `Content-Type:multipart/form-data`
    - `Authorization: Bearer <jwt_token>`

  - Body：

    - livename
  
- 返回示例：

  - 成功：

    ```
    {
        "resp": {
            "code": 0,
            "msg": "ok",
            "data": "success"
        }
    }
    ```
    
  - 错误：

    ```
    {
        "resp": {
            "code": 5002,
            "msg": "remote or network error[remote]: biz error: 权限不够！你不是当前课程老师",
            "data": "nil"
        }
    }
    ```

------

##### Change User In Live

简介：学生/教师进入或退出直播间

- **接口地址**：`PUT /change_user_in_live`

- **功能说明**：更新用户在 LiveGo 直播间的状态（进入/退出）。

- 参数解释：

  | 参数名称  | 解释                     |
  | --------- | ------------------------ |
  | lesson_id | 课程 ID                  |
  | Options   | enum,"add"or"del"(+1/-1) |
  
  
  
- 请求示例：

  - Headers：

    - `Content-Type: multipart/form-data`
    - `Authorization: Bearer <jwt_token>`

  - Body：

    lesson_id : xxx
    
    options: enum(add,del)
  
- 返回示例：

  - 成功：

    ```
    {
        "resp": {
            "code": 0,
            "msg": "ok",
            "data": "success"
        }
    }
    ```
    
  - 错误：
  
    ```
    {
        "resp": {
            "code": 5001,
            "msg": "参数错误",
            "data": ""
        }
    }
    ```

------

##### Change User To Lesson 

简介：将学生加入课程中

- **接口地址**：`PUT /change_user_to_lesson`

- **功能说明**：将用户加入或移出某门课程表中。

- 参数解释：

  | 参数名称    | 解释          |
  | ----------- | ------------- |
  | lesson_name | 课程名字      |
  | teacher     | 老师名字      |
  | option      | enum(add,del) |
  | student_id  | 学生ID        |

  

- 请求示例：

  - Headers：

    - `Content-Type: multipart/form-data`
    - `Authorization: Bearer <jwt_token>`

  - Body：

    lesson_name:xxx
    
    teacher:xxx,
    
    option:enum(add,del)
  
- 返回示例：

  - 成功:
  
  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "success"
      }
  }
  ```
  
  - 错误:
  
    ```
    {
        "resp": {
            "code": 5002,
            "msg": "remote or network error[remote]: biz error: invalid options",
            "data": "nil"
        }
    }
    ```
  
    ​	

------



##### Select Lesson Info 

简介：查询直播间在线人数信息

- **接口地址**：`GET /select_lesson`

- **功能说明**：返回当前直播间的在线用户统计。

- 参数解释：

  | 参数名称  | 解释   |
  | --------- | ------ |
  | lesson_id | 课程id |
  | teacher   | 老师   |

  

- 请求示例：

  - Headers：
    - `Content-Type: multipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
    - teacher:xxx,
  
- 返回示例：

  - 成功

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "count:1///live member:kq1$Teacher  "
      }
  }
  ```
  
  - 失败或不存在:
  
    ```
    {
        "resp": {
            "code": 0,
            "msg": "ok",
            "data": "count:0///live member:"
        }
    }
    ```
  
    

##### Get Lesson Info 

简介：查询课程详情

- **接口地址**：`GET /get_lesson`

- **功能说明**：返回指定课程的基本信息。

- 参数解释：

  | 参数名称    | 解释     |
  | ----------- | -------- |
  | lesson_name | 课程名称 |
  | teacher     | 老师     |

  

- 请求示例：

  - Headers：
    - `Content-Type: multipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  - Body:
    - lesson_name:xxx,
    - teacher:xxx,
  
- 返回示例：

  - 成功:
  
  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "3$class1$kq1$for 2025$a0UgPUNSBnLZbvgpaRJd5E3A7K0bah44iyDVd34dpFAlDl3H$2/10/"
      }
  }
  ```
  
  - 失败:
  
    ```
    {
        "resp": {
            "code": 5002,
            "msg": "remote or network error[remote]: biz error: record not found",
            "data": "nil"
        }
    }
    ```
  
    

------



##### Record Lesson 

简介：录制并保存直播流信息

- **接口地址**：`POST /record_lesson`

- **功能说明**：接收录制文件路径或元数据，将直播录制结果保存至数据库并上传至存储。

- 参数解释：

  | 参数名称   | 解释                    |
  | ---------- | ----------------------- |
  | stream_url | rmtp地址                |
  | lesson_id  | 课程id                  |
  | duration   | 录制时间段(0为从头到尾) |

  

- 请求示例：

  - Headers：

    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`

  - Body：

    stream_url:rtmp://localhost:1935/live/class1_kq1
    
    lesson_id:3
    
    duration:0
  
- 返回示例：

  - 成功
  
    ```
    {
        "resp": {
            "code": 0,
            "msg": "ok",
            "data": "success:https://console.cloud.tencent.com/cos/bucket?bucket=lanshan-1338048877&region=ap-chongqing"
        }
    }
    ```
  
    
  
  - 失败
  
  ```
  {
      "resp": {
          "code": 5002,
          "msg": "remote or network error[remote]: biz error: 你不是当前课程的学生或老师！！！",
          "data": "nil"
      }
  }
  ```

------

##### Create Sign-In 

简介：创建一次签到活动

- **接口地址**：`POST /create_signin`

- **功能说明**：发起新的签到，包括开始和结束时间。

- 参数解释：

  | 参数名称  | 解释    |
  | --------- | ------- |
  | lesson_id | 课程 ID |
  
  

- 请求示例：

  - Headers：
  
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body：
  
    lesson_id:xxx,
  
- 返回示例：

  - 成功
  
    ```
    {
        "resp": {
            "code": 0,
            "msg": "ok",
            "data": "success"
        }
    }
    ```
  
    
  
  - 失败
  
  ```
  {
      "resp": {
          "code": 5002,
          "msg": "remote or network error[remote]: biz error: 你已创建过签到",
          "data": "nil"
      }
  }
  ```

------

##### Sign In 

简介：学生签到

- **接口地址**：`PUT /signin`

- **功能说明**：学生提交签到请求。

- 参数解释：

  | 参数名称  | 解释   |
  | --------- | ------ |
  | lesson_id | 课程ID |
  
  
  
- 请求示例：

  - Headers：
  
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body：
  
    lesson_id:xxx,
  
- 返回示例：

  - 成功
  
  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "success"
      }
  }
  ```
  
  - 失败
  
    ```
    {
        "resp": {
            "code": 5002,
            "msg": "remote or network error[remote]: biz error: 不是此课程学生",
            "data": "nil"
        }
    }
    ```
  
    



------



##### Select Sign-In 

简介：查询签到结果

- **接口地址**：`GET /select_signin`

- **功能说明**：获取某次签到的所有记录列表。

- 参数解释：

  | 参数名称  | 解释   |
  | --------- | ------ |
  | lesson_id | 课程ID |

  

- 请求示例：

  - Headers：
  
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body：
  
    lesson_id:xxx,

- 返回示例：

  - 成功:
  
    ```
    {
        "resp": {
            "code": 0,
            "msg": "ok",
            "data": "已签到为, 未签到为2"
        }
    }
    ```
  
    
  
  - 失败:
  
  ```
  {
      "resp": {
          "code": 5002,
          "msg": "remote or network error[remote]: biz error: 权限不够！！！你不是当前课程老师",
          "data": "nil"
      }
  }
  ```

------

##### Delete Sign-In 

简介：删除一次签到

- **接口地址**：`DELETE /del_signin`

- **功能说明**：移除某次签到及其所有记录。

- 参数解释：

  | 参数名称  | 解释   |
  | --------- | ------ |
  | lesson_id | 课程id |

  

- 请求示例：

  - Headers：
  
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body：
  
    lesson_id:xxx,
  
- 返回示例：

  - 成功
  
    ```
    {
        "resp": {
            "code": 0,
            "msg": "ok",
            "data": "success"
        }
    }
    ```
  
    
  
  - 失败
  
  ```
  {
      "resp": {
          "code": 5002,
          "msg": "remote or network error[remote]: biz error: 权限不够！！！你不是当前课程老师",
          "data": "nil"
      }
  }
  ```

------

##### Roll Call 

简介：随机点名一次

- **接口地址**：`GET /roll_call`

- **功能说明**：从已加入课程的学生列表中随机抽取若干名学生。

- 参数解释：

  | 参数名称  | 解释    |
  | --------- | ------- |
  | lesson_id | 课程 ID |

  

- 请求示例：

  - Headers：
  
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body：
  
    lesson_id:xxx,

- 返回示例：

  - 成功
  
    ```
    {
        "resp": {
            "code": 0,
            "msg": "ok",
            "data": "kq1"
        }
    }
    ```
  
    
  
  - 失败
  
  ```
  {
      "resp": {
          "code": 5002,
          "msg": "remote or network error[remote]: biz error: 权限不够！！！你不是当前课程老师",
          "data": "nil"
      }
  }
  ```

------

##### Broadcast(WebRTC) 

简介：教师发起 WebRTC 播流

- **接口地址**：`POST /broadcast`

- **功能说明**：接收 SDP offer，调用后端 RPC 创建 WebRTC 连接，返回 SDP answer。

- 参数解释：

  | 参数名称  | 解释                    |
  | --------- | ----------------------- |
  | lesson_id | 课程 ID                 |
  | b64offer  | Base64 编码的 SDP offer |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
    - b64offer:xxx,
  
- 返回示例：

- 成功:

  ```
  {
    "resp": {
      "code": 0,
      "msg": "ok",
      "data": （b64answer）
    }
  }
  ```

------

##### View (WebRTC)

简介：学生接收 WebRTC 播流

- **接口地址**：`POST /view`

- **功能说明**：接收 Base64 编码的 SDP answer，完成学生端 WebRTC 连接。

- 参数解释：

  | 参数名称  | 解释                    |
  | --------- | ----------------------- |
  | lesson_id | 课程 ID                 |
  | b64offer  | Base64 编码的 SDP offer |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
    - b64offer:xxx,
  
  
  
- 返回示例：

  ```
  {
    "resp": {
      "code": 0,
      "msg": "ok",
      "data": (b64answer)
    }
  }
  ```

------

##### Create Lesson (WebRTC)

简介：创建 WebRTC 课程实例

- **接口地址**：`POST /create_lesson_webrtc`

- **功能说明**：初始化直播源信息

- 参数解释：

  | 参数名称   | 解释     |
  | ---------- | -------- |
  | lessonname | 课程名字 |
  | desc       | 概述     |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`

  - Bodys:
    - lessonname: xxx

    - desc:xxx




- 返回示例：

  ```
  {
    "resp": {
      "code": 0,
      "msg": "ok",
      "data": "success",
    }
  }
  ```

------

##### Delete Lesson (WebRTC)

简介：删除 WebRTC 课程实例

- **接口地址**：`DELETE /del_lesson_webrtc`

- **功能说明**：移除房间及相关资源源信息。

- 参数解释：

  | 参数名称  | 解释    |
  | --------- | ------- |
  | lesson_id | 课程 ID |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  - Body:
    - lesson_id:xxx
  
  
  
- 返回示例：

  ```
  {
    "resp": {
      "code": 0,
      "msg": "ok",
      "data": "success"
    }
  }
  ```

------

##### Change User To Lesson (WebRTC)

简介：学生/教师切换到 WebRTC 课程

- **接口地址**：`PUT /change_user_to_lesson_webrtc`

- **功能说明**：在 WebRTC 环境中加入／移出课程房间。

- 参数解释：

  | 参数名称  | 解释          |
  | --------- | ------------- |
  | lesson_id | 课程 ID       |
  | options   | enum(add,del) |
  
  
  
- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  - Body:
    - lesson_id:xxx
    - options:add
  



- 返回示例：

  ```
  {
    "resp": {
      "code": 0,
      "msg": "ok",
      "data": "success"
    }
  }
  ```

------

##### Change User In Live (WebRTC)

简介：WebRTC 模式下用户进入/退出直播间

- **接口地址**：`GET /change_user_in_live_webrtc`

- **功能说明**：更新 WebRTC 下用户在直播间的状态。

- 参数解释：

  | 参数名称  | 解释          |
  | --------- | ------------- |
  | lesson_id | 课程 ID       |
  | options   | enum(add,del) |
  
  
  
- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  - Body:
    - lesson_id:xxx
    - options:add
  
- 返回示例：

  ```
  {
    "resp": {
      "code": 0,
      "msg": "ok",
      "data": "success"
    }
  }
  ```

------

##### Select Lesson Info (WebRTC)

简介：查询 WebRTC 直播间在线人数

- **接口地址**：`GET /select_lesson_webrtc`

- **功能说明**：返回 WebRTC 当前直播间在线用户数。

- 参数解释：

  | 参数名称  | 解释    |
  | --------- | ------- |
  | lesson_id | 课程 ID |

  

- 请求示例：	

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  - Body:
    - lesson_id:xxx
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "count:1///live member:kq1$Teacher  "
      }
  }
  ```

------

##### Get Lesson Info (WebRTC)

简介：查询 WebRTC 课程详情

- **接口地址**：`GET /get_lesson_webrtc`

- **功能说明**：返回 WebRTC 课程的元信息。

- 参数解释：

  | 参数名称    | 解释     |
  | ----------- | -------- |
  | lesson_name | 课程名称 |
  | teacher     | 老师     |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`

  - Body:
    - lesson_name:xxx,
    - teacher:xxx,




- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "3$class1$kq1$for 2025$a0UgPUNSBnLZbvgpaRJd5E3A7K0bah44iyDVd34dpFAlDl3H$2/10/"
      }
  }
  ```

------

##### Create Sign-In (WebRTC)

简介：在 WebRTC 环境中创建签到

- **接口地址**：`POST /create_signin_webrtc`

- **功能说明**：发起一次 WebRTC 签到活动。

- 参数解释：

  | 参数名称  | 解释    |
  | --------- | ------- |
  | lesson_id | 课程 ID |
  
  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "success"
      }
  }
  ```

------

##### Sign In (WebRTC)

简介：WebRTC 学生签到

- **接口地址**：`PUT /signin_webrtc`

- **功能说明**：提交 WebRTC 签到请求。

- 参数解释：

  | 参数名称  | 解释   |
  | --------- | ------ |
  | lesson_id | 课程id |
  
  
  
- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "success"
      }
  }
  ```

------

##### Select Sign-In (WebRTC)

简介：查询 WebRTC 签到结果

- **接口地址**：`GET /select_signin_webrtc`

- **功能说明**：获取某次 WebRTC 签到的记录列表。

- 参数解释：

  | 参数名称  | 解释   |
  | --------- | ------ |
  | lesson_id | 课程ID |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "已签到为1, 未签到为2"
      }
  }
  ```

------

##### Delete Sign-In (WebRTC)

简介：删除一次 WebRTC 签到

- **接口地址**：`DELETE /del_signin_webrtc`

- **功能说明**：人工移除一次 WebRTC 签到及其记录。

- 参数解释：

  | 参数名称  | 解释 |
  | --------- | ---- |
  | lesson_id | 课程 |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "success"
      }
  }
  ```

------

##### Roll Call (WebRTC)

简介：WebRTC 随机点名

- **接口地址**：`GET /roll_call_webrtc`

- **功能说明**：随机抽取 WebRTC 课程中的学生。

- 参数解释：

  | 参数名称  | 解释    |
  | --------- | ------- |
  | lesson_id | 课程 ID |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "kq1"
      }
  }
  ```

------

##### Record Lesson (WebRTC)

简介：录制并保存 WebRTC 流

- **接口地址**：`POST /record_lesson_webrtc`

- **功能说明**：接收录制文件地址或元数据，将 WebRTC 流录制结果保存并上传。

- 参数解释：

  | 参数名称  | 解释     |
  | --------- | -------- |
  | lesson_id | 课程 ID  |
  | file      | 视频文件 |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
    - file:(前端发送的整个文件)
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": (COS中filename)
      }
  }
  ```

------

##### Save Whiteboard

简介：保存白板状态

- **接口地址**：`POST /save_whiteboard`

- **功能说明**：接收前端传来的 Excalidraw JSON 并保存在后端。

- 参数解释：

  | 参数名称  | 解释                 |
  | --------- | -------------------- |
  | lesson_id | 课程 ID              |
  | file      | Excalidraw 导出 JSON |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
    - file:(前端发送的整个excalidraw的json)
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "success"
      }
  }
  ```

------

##### Get Whiteboard

简介：获取白板状态

- **接口地址**：`GET /get_whiteboard`

- **功能说明**：返回指定课程的最新白板 JSON。

- 参数解释：

  | 参数名称  | 解释    |
  | --------- | ------- |
  | lesson_id | 课程 ID |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  - Body:
    - lesson_id:xxx,
  
- 返回示例：

  ```
  {
    "resp": {
      "code": 0,
      "msg": "ok",
      "data": (mysql中存储的excalidraw的json)
    }
  }
  ```

------

##### Raise Hand

简介：学生举手

- **接口地址**：`PUT /raise_hand`

- **功能说明**：记录学生举手请求。

- 参数解释：

  | 参数名称  | 解释    |
  | --------- | ------- |
  | lesson_id | 课程 ID |
  
  
  
- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "success"
      }
  }
  ```

------

##### Get Raise Hands

简介：查询所有举手请求

- **接口地址**：`GET /get_raise_hand`

- **功能说明**：返回当前课程的所有未处理举手记录。

- 参数解释：

  | 参数名称  | 解释    |
  | --------- | ------- |
  | lesson_id | 课程 ID |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
  
- 返回示例：

  ```
  {
    "resp": {
      "code": 0,
      "msg": "ok",
      "data": “1/3/5”
    }
  }
  ```

------

##### Approve Hand

简介：教师批准举手

- **接口地址**：`PUT /approve_hand`

- **功能说明**：教师批准某次举手请求，将该学生放入连麦队列。

- 参数解释：

  | 参数名称  | 解释    |
  | --------- | ------- |
  | lesson_id | 课程 ID |
  | stuid     | 学生 ID |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
    - stuid:xxx,
  
- 返回示例：

  ```
  {
    "resp": {
      "code": 0,
      "msg": "ok",
      "data": “success”
    }
  }
  ```

------

##### Publish Mic

简介：学生/教师发布麦克风流

- **接口地址**：`POST /publish_mic`

- **功能说明**：接收 Base64 编码的 SDP offer，开启音频连麦流。

- 参数解释：

  | 参数名称  | 解释                    |
  | --------- | ----------------------- |
  | lesson_id | 课程 ID                 |
  | b64offer  | Base64 编码的 SDP offer |

  

- 请求示例：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
    - b64offer:(offer)
  
- 返回示例：

  ```
  {
    "resp": {
      "code": 0,
      "msg": "ok",
      "data": (answer)
    }
  }
  ```

------

##### View Mic

简介：学生/教师接收麦克风流

- **接口地址**：`POST /view_mic`

- **功能说明**：接收 Base64 编码的 SDP answer，建立音频连麦。

- 参数解释：

  | 参数名称  | 解释                    |
  | --------- | ----------------------- |
  | lesson_id | 课程 ID                 |
  | b64offer  | Base64 编码的 SDP offer |

  

- 请求示例：

  - Headers:
    - `Content-Type:application/json`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx,
    - b64offer:(offer)
  
  
  
- 返回示例：

  ```
  {
    "resp": {
      "code": 0,
      "msg": "ok",
      "data": (answer)
    }
  }
  ```

------

##### Create Question

简介：创建一道题目

- **接口地址**：`POST /create_question`

- **功能说明**：在题库中新增一道题目。

- 参数解释：

  - Headers:
    - `Content-Type:mutipart/form-data`
    - `Authorization: Bearer <jwt_token>`

  - Body:
    - lesson_id:xxx,
    - b64offer:(offer)

  | 参数名称    | 解释      |
  | ----------- | --------- |
  | lesson_id   | 课程 ID   |
  | content     | 题目 内容 |
  | option_nums | 选项个数  |
  | options     | 选项内容  |
  | answer      | 答案      |

  

- 请求示例：

  ```
  {
      "lesson_id":"9",
      "content":"math",
      "option_nums":4,
      "options":["kq","hyx","azh","jzj"],
      "answer":"A"
  }
  ```

- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "1"(question的id)
      }
  }
  ```

------

##### Delete Question

简介：删除一道题目

- **接口地址**：`DELETE /del_question`

- **功能说明**：从题库中移除指定题目。

- 参数解释：

  | 参数名称    | 解释    |
  | ----------- | ------- |
  | question_id | 题目 ID |

  

- 请求示例：

  - Headers:
    - `Content-Type:application/json`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - question_id:xxx
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "success"
      }
  }
  ```

------

##### Chat with Agent

简介：与智能助教对话

- **接口地址**：`POST /chat_agent`

- **功能说明**：提交用户问题，返回 AI 助教的回答。

- 参数解释：

  | 参数名称 | 解释     |
  | -------- | -------- |
  | message  | 用户消息 |

  

- 请求示例：

  - Headers:
    - `Content-Type:application/json`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - message:请写一篇静夜思的教学大纲
  
  
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "\n# 《静夜思》教案大纲\n\n## 一、教学目标\n### （一）知识与技能\n1. 能正确认读“静、夜、思”等生字，会写“床、前、光”等重点字词。\n2. 正确、流利、有感情地朗读古诗，背诵古诗。\n3. 理解诗句的意思，感受古诗的语言美和意境美。\n\n### （二）过程与方法\n1. 通过反复朗读、小组合作探究，理解古诗的内容和情感。\n2. 结合生活经验，想象诗中描绘的画面，培养学生的想象力和语言表达能力。\n\n### （三）情感态度与价值观\n体会诗人对故乡的思念之情，激发学生对家乡、对亲人的热爱之情。\n\n## 二、教学重难点\n### （一）重点\n1. 正确朗读、背诵古诗，理解诗句的意思。\n2. 掌握重点字词的写法和意思。\n\n### （二）难点\n体会诗人借景抒情的写作手法，感受诗中蕴含的思乡之情。\n\n## 三、教学方法\n朗读法、情境教学法、小组合作法、讲授法\n\n## 四、教学准备\n1. 多媒体课件（包含古诗配乐、插图、作者简介等）\n2. 生字卡片、田字格黑板贴\n3. 学生提前预习古诗，收集关于李白的资料\n\n## 五、教学过程\n### （一）导入新课（5分钟）\n1. 播放中秋月圆、家人团聚的视频片段，提问学生：“看到这样的画面，你有什么感受？如果出门在外，不能和家人团聚，你又会有什么感觉？”\n2. 引出课题《静夜思》，介绍诗人李白的背景（结合学生收集的资料，简单补充李白生平，如他是唐代著名的浪漫主义诗人，有“诗仙”之称，一生游历四方，写下了许多脍炙人口的诗篇）。\n\n### （二）初读古诗，识字解词（15分钟）\n1. 教师范读古诗，要求学生注意听准字音、停顿和语气。（播放配乐范读，营造静谧的氛围）\n2. 学生自由朗读古诗，圈出不认识的生字。\n3. 出示生字卡片（静、夜、思、床、前、光、疑、是、地、上、霜、举、头、望、明、月、低、头、思、故、乡），采用“小老师领读”“开火车读”“分组比赛读”等方式，检查学生认读情况，重点强调“疑”“霜”“举”等易错字的读音。\n4. 指导书写重点字词（以“床、前、光”为例）：\n    - 观察田字格中生字的结构、笔顺，教师在黑板上示范书写，强调关键笔画（如“床”是半包围结构，先写广字头，再写里面的“木”；“前”的上半部分是“丷”和“月”，下半部分是“刖”）。\n    - 学生在练习本上书写，教师巡视指导，及时纠正书写姿势和笔顺错误。\n5. 结合注释和生活经验，理解重点词语的意思：\n    - “静夜思”：静静的夜晚所引起的思念。\n    - “疑”：好像。\n    - “霜”：指白色的霜。\n    - “举头”：抬头。\n    - “低头”：低下头。\n    - “故乡”：家乡。\n\n### （三）精读古诗，想象画面（20分钟）\n1. 学生再次朗读古诗，边读边思考：诗中描绘了哪些景物？（明月光、地上霜、明月）\n2. 小组合作探究，讨论诗句的意思：\n    - “床前明月光，疑是地上霜”：明亮的月光洒在床前的地面上，诗人好像以为那是地上的霜。（引导学生想象：在一个安静的夜晚，诗人独自躺在床上，看到床前一片银白色的月光，一开始可能有些恍惚，以为是地上结了霜）\n    - “举头望明月，低头思故乡”：诗人抬头望着天上的明月，不由得低下头思念起自己的家乡。（提问学生：诗人为什么会望着明月思念家乡？引导学生联系生活经验，如中秋节时，人们看到月亮就会想起家人，诗人远离家乡，看到明月，自然也会思念家乡的亲人）\n3. 播放古诗插图，让学生根据诗句内容，用自己的话描述诗中描绘的画面，教师及时评价和补充，帮助学生形成生动的画面感。\n4. 指导学生有感情地朗读古诗：\n    - 第一句“床前明月光，疑是地上霜”读得轻柔、缓慢，表现出夜晚的静谧和诗人的恍惚。\n    - 第二句“举头望明月，低头思故乡”读得稍慢，语气中带有一丝忧伤和思念，“望明月”时声音稍高，“思故乡”时声音低沉，表现出诗人对家乡的深切思念。\n    - 学生练习朗读，教师个别指导，然后请学生上台展示朗读，其他学生评价，最后全班齐读。\n\n### （四）拓展延伸（10分钟）\n1. 交流讨论：除了《静夜思》，你还知道哪些表达思乡之情的古诗？（如王维的《九月九日忆山东兄弟》、王安石的《泊船瓜洲》等，可让学生简单背诵其中的名句）\n2. 开展“我是小诗人”活动：仿照《静夜思》的结构，结合自己的生活经验，写一写自己在某个夜晚的感受（可以是思念家人、想念朋友等），然后分享给全班同学。\n\n### （五）总结作业（5分钟）\n1. 总结本节课内容：通过学习《静夜思》，我们不仅学会了朗读和背诵古诗，理解了诗句的意思，还体会到了诗人对故乡的思念之情。希望同学们在今后的学习和生活中，能够更加热爱我们的家乡和亲人。\n2. 布置作业：\n    - 必做：背诵并默写《静夜思》，给家人朗诵古诗，分享诗中的故事和自己的感受。\n    - 选做：收集3首以上表达思乡之情的古诗，下节课分享。\n\n## 六、板书设计\n```\n          静夜思\n          李白\n床前明月光，疑是地上霜。\n举头望明月，低头思故乡。\n\n生字：静、夜、思、床、前、光……\n情感：思乡之情\n```"
      }
  }
  ```

------

##### List All User Convs

简介：列出所有用户会话

- **接口地址**：`GET /list_user_conv`

- **功能说明**：返回当前用户与 AI 的全部对话 ID 列表；

- 参数解释:无

- 请求示例：

  - Headers:
    - `Content-Type:application/json`
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "{\"role\":\"user\",\"content\":\"写一篇静夜思的教案大纲\"}\n{\"role\":\"assistant\",\"content\":\"\\n# 《静夜思》教案大纲\\n\\n## 一、教学目标\\n### （一）知识与技能\\n1. 能正确认读“静、夜、思”等生字，会写“床、前、光”等重点字词。\\n2. 正确、流利、有感情地朗读古诗，背诵古诗。\\n3. 理解诗句的意思，感受古诗的语言美和意境美。\\n\\n### （二）过程与方法\\n1. 通过反复朗读、小组合作探究，理解古诗的内容和情感。\\n2. 结合生活经验，想象诗中描绘的画面，培养学生的想象力和语言表达能力。\\n\\n### （三）情感态度与价值观\\n体会诗人对故乡的思念之情，激发学生对家乡、对亲人的热爱之情。\\n\\n## 二、教学重难点\\n### （一）重点\\n1. 正确朗读、背诵古诗，理解诗句的意思。\\n2. 掌握重点字词的写法和意思。\\n\\n### （二）难点\\n体会诗人借景抒情的写作手法，感受诗中蕴含的思乡之情。\\n\\n## 三、教学方法\\n朗读法、情境教学法、小组合作法、讲授法\\n\\n## 四、教学准备\\n1. 多媒体课件（包含古诗配乐、插图、作者简介等）\\n2. 生字卡片、田字格黑板贴\\n3. 学生提前预习古诗，收集关于李白的资料\\n\\n## 五、教学过程\\n### （一）导入新课（5分钟）\\n1. 播放中秋月圆、家人团聚的视频片段，提问学生：“看到这样的画面，你有什么感受？如果出门在外，不能和家人团聚，你又会有什么感觉？”\\n2. 引出课题《静夜思》，介绍诗人李白的背景（结合学生收集的资料，简单补充李白生平，如他是唐代著名的浪漫主义诗人，有“诗仙”之称，一生游历四方，写下了许多脍炙人口的诗篇）。\\n\\n### （二）初读古诗，识字解词（15分钟）\\n1. 教师范读古诗，要求学生注意听准字音、停顿和语气。（播放配乐范读，营造静谧的氛围）\\n2. 学生自由朗读古诗，圈出不认识的生字。\\n3. 出示生字卡片（静、夜、思、床、前、光、疑、是、地、上、霜、举、头、望、明、月、低、头、思、故、乡），采用“小老师领读”“开火车读”“分组比赛读”等方式，检查学生认读情况，重点强调“疑”“霜”“举”等易错字的读音。\\n4. 指导书写重点字词（以“床、前、光”为例）：\\n    - 观察田字格中生字的结构、笔顺，教师在黑板上示范书写，强调关键笔画（如“床”是半包围结构，先写广字头，再写里面的“木”；“前”的上半部分是“丷”和“月”，下半部分是“刖”）。\\n    - 学生在练习本上书写，教师巡视指导，及时纠正书写姿势和笔顺错误。\\n5. 结合注释和生活经验，理解重点词语的意思：\\n    - “静夜思”：静静的夜晚所引起的思念。\\n    - “疑”：好像。\\n    - “霜”：指白色的霜。\\n    - “举头”：抬头。\\n    - “低头”：低下头。\\n    - “故乡”：家乡。\\n\\n### （三）精读古诗，想象画面（20分钟）\\n1. 学生再次朗读古诗，边读边思考：诗中描绘了哪些景物？（明月光、地上霜、明月）\\n2. 小组合作探究，讨论诗句的意思：\\n    - “床前明月光，疑是地上霜”：明亮的月光洒在床前的地面上，诗人好像以为那是地上的霜。（引导学生想象：在一个安静的夜晚，诗人独自躺在床上，看到床前一片银白色的月光，一开始可能有些恍惚，以为是地上结了霜）\\n    - “举头望明月，低头思故乡”：诗人抬头望着天上的明月，不由得低下头思念起自己的家乡。（提问学生：诗人为什么会望着明月思念家乡？引导学生联系生活经验，如中秋节时，人们看到月亮就会想起家人，诗人远离家乡，看到明月，自然也会思念家乡的亲人）\\n3. 播放古诗插图，让学生根据诗句内容，用自己的话描述诗中描绘的画面，教师及时评价和补充，帮助学生形成生动的画面感。\\n4. 指导学生有感情地朗读古诗：\\n    - 第一句“床前明月光，疑是地上霜”读得轻柔、缓慢，表现出夜晚的静谧和诗人的恍惚。\\n    - 第二句“举头望明月，低头思故乡”读得稍慢，语气中带有一丝忧伤和思念，“望明月”时声音稍高，“思故乡”时声音低沉，表现出诗人对家乡的深切思念。\\n    - 学生练习朗读，教师个别指导，然后请学生上台展示朗读，其他学生评价，最后全班齐读。\\n\\n### （四）拓展延伸（10分钟）\\n1. 交流讨论：除了《静夜思》，你还知道哪些表达思乡之情的古诗？（如王维的《九月九日忆山东兄弟》、王安石的《泊船瓜洲》等，可让学生简单背诵其中的名句）\\n2. 开展“我是小诗人”活动：仿照《静夜思》的结构，结合自己的生活经验，写一写自己在某个夜晚的感受（可以是思念家人、想念朋友等），然后分享给全班同学。\\n\\n### （五）总结作业（5分钟）\\n1. 总结本节课内容：通过学习《静夜思》，我们不仅学会了朗读和背诵古诗，理解了诗句的意思，还体会到了诗人对故乡的思念之情。希望同学们在今后的学习和生活中，能够更加热爱我们的家乡和亲人。\\n2. 布置作业：\\n    - 必做：背诵并默写《静夜思》，给家人朗诵古诗，分享诗中的故事和自己的感受。\\n    - 选做：收集3首以上表达思乡之情的古诗，下节课分享。\\n\\n## 六、板书设计\\n```\\n          静夜思\\n          李白\\n床前明月光，疑是地上霜。\\n举头望明月，低头思故乡。\\n\\n生字：静、夜、思、床、前、光……\\n情感：思乡之情\\n```\",\"response_meta\":{\"finish_reason\":\"stop\",\"usage\":{\"prompt_tokens\":831,\"completion_tokens\":1583,\"total_tokens\":2414}},\"extra\":{\"ark-reasoning-content\":\"用户需求是写一篇《静夜思》的教案大纲，当前提供了《静夜思》等古诗的相关材料，需要结合这些材料以及教学相关知识来完成。现有工具中，calculate用于算术运算，current_time获取当前时间，random_number生成随机数，sha256计算哈希值，uuid4生成UUID，duckduckgo_search用于网络搜索。用户需求是编写教案大纲，不需要这些工具的功能，因此不需要使用工具，直接根据教学知识和提供的《静夜思》内容来组织教案大纲。\\n\",\"ark-request-id\":\"021749051877194a5b2811eb616610ebc3f86df97ca433b7455e9\"}}\n"
      }
  }
  ```

------

##### Delete All User Convs

简介：删除指定用户的所有会话

- **接口地址**：`DELETE /del_user_conv`

- **功能说明**：清空当前用户所有与 AI 的对话记录。

- 参数解释：无

- 请求示例：

  - Headers:
    - `Content-Type:application/json`
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "success"
      }
  }
  ```

------

##### Get History

简介：获取指定对话历史

- **接口地址**：`GET /get_his`

- **功能说明**：返回课程会话的全部消息列表。

- 参数解释：

  | 参数名称  | 解释    |
  | --------- | ------- |
  | lesson_id | 课程 ID |

  

- 请求示例：

  - Headers:
    - `Content-Type:application/json`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "[{\"_id\":\"68406bc0651682d726dbdca6\",\"content\":\"i love u\",\"lesson_id\":\"3\",\"sender\":\"2\",\"timestamp\":\"2025-06-04T15:52:32.86Z\"}]"
      }
  }
  ```

------

##### Delete History

简介：删除指定对话历史

- **接口地址**：`DELETE /del_his`

- **功能说明**：移除某次会话的全部消息记录。

- 参数解释：

  | 参数名称  | 解释    |
  | --------- | ------- |
  | lesson_id | 课程 ID |

  

- 请求示例：

  - Headers:
    - `Content-Type:application/json`
    - `Authorization: Bearer <jwt_token>`
  
  - Body:
    - lesson_id:xxx
  
- 返回示例：

  ```
  {
      "resp": {
          "code": 0,
          "msg": "ok",
          "data": "success"
      }
  }
  ```





##### Live_Chat(WebSocket)

简介:课程中聊天

- **接口地址**：`GET ws/live_chat`

- **功能说明**:课程实时聊天

- 参数解释:

  | 参数名称  | 解释      |
  | --------- | --------- |
  | lesson_id | 课程id    |
  | token     | JWT token |

  

- 请求示例（链接）：

  `QueryString`

  - Body:
    - lesson_id:xxx
    - token:xxx

- 请求示例(发送):

  `Content-Type:application/json`

  - Body:

    ```
    {
        "lesson_id":"3",
        "sender":"kq1",
        "content":"i love u"
    }
    ```

    ​		

  

##### Quiz(WebSocket)

简介:实时回答

- **接口地址**：`GET ws/quiz`

- **功能说明**: 教师实时接受学生回答，学生实时回答

- 参数解释:

  | 参数名称  | 解释      |
  | --------- | --------- |
  | lesson_id | 课程id    |
  | token     | JWT token |

  

- 请求示例(链接)：

  `QueryString`

  - Body:
    - lesson_id:xxx
    - token:xxx
  
- 请求示例(作答):

  `Content-Type:application/json`

  Body:
  
  ```
  {
  	"question_id":"xxx",
  	"answer":"xxx"
  }
  ```
  
  教师端就能实时接收