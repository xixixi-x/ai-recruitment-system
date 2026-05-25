# ai-recruitment-system
=======
# AI Agent 智能招聘系统大作业

本项目按照题目要求拆分为四个独立工程：

```text
final_homework/
├── hr-frontend/          # HR 管理端前端页面
├── user-frontend/        # 候选人用户端前端页面
├── web-gin-service/      # Gin Web 网关服务，只处理 HTTP、JWT、参数校验和 gRPC 转发
├── logic-grpc-service/   # Logic 核心业务 gRPC 服务，处理 MySQL、OSS、Eino AI、权限业务逻辑
├── api.md                # 前后端接口说明
├── db.md                 # 数据库设计说明
├── .env.example          # 环境变量模板
└── go.work               # 本地 Go workspace
```

第七点拓展设计 `answer.md` 暂未实现，当前版本优先完成系统要求中的核心开发功能。

## 一、核心技术目标对应关系

### 1. 两层后端架构：Web + Logic

- `web-gin-service` 是 Gin 网关层，只接收前端 HTTP 请求。
- `logic-grpc-service` 是核心业务层，负责用户、岗位、投递、MySQL、OSS、AI 对话。
- Web 服务与 Logic 服务之间只通过 gRPC 远程调用，Web 服务不直接连接数据库，也不直接调用 AI。

### 2. JWT 双角色登录与权限隔离

- 支持 `hr` 与 `candidate` 两类账号。
- 支持多个 HR 账号。
- HR 只能查看自己创建岗位收到的投递。
- 候选人未登录时只能浏览公开岗位，不能投递。
- 候选人投递前必须完善结构化档案并上传合规简历。

### 3. 私有 OSS 简历上传与下载

- 候选人端通过 `POST /candidate/resume/sign-upload` 获取签名上传 URL。
- 前端使用签名 URL 直传 OSS。
- 上传完成后调用 `POST /candidate/resume/confirm` 将 objectKey 写入 MySQL。
- HR 查看候选人简历时调用 `GET /hr/applications/:id/resume-url` 获取签名下载 URL。
- 只允许 PDF、DOC、DOCX，拒绝图片、压缩包、TXT 等格式。

### 4. Eino AI 对话

- AI 功能封装在 Logic 服务中。
- 处理流程：HR 自然语言问题 → 查询 MySQL 真实业务数据 → 拼接业务上下文 → 调用 Eino ChatModel → 保存历史对话。
- 当前不做向量库、不做 RAG、不做简历智能匹配，符合题目要求。

## 二、运行前准备

### 1. 创建 MySQL 数据库

```sql
CREATE DATABASE ai_recruitment DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
```

### 2. 配置环境变量

复制 `.env.example` 为 `.env`，并按实际情况修改：

```bash
cp .env.example .env
```

必须配置：

```env
MYSQL_DSN=root:123456@tcp(127.0.0.1:3306)/ai_recruitment?charset=utf8mb4&parseTime=True&loc=Local
JWT_SECRET=change-this-secret
OSS_ENDPOINT=oss-cn-hangzhou.aliyuncs.com
OSS_ACCESS_KEY_ID=你的AccessKeyId
OSS_ACCESS_KEY_SECRET=你的AccessKeySecret
OSS_BUCKET=你的私有Bucket名称
AI_API_KEY=你的大模型APIKey
AI_BASE_URL=https://api.deepseek.com/v1
AI_MODEL=deepseek-chat
```

OSS Bucket 应保持私有读写，不要开启公共读。

## 三、启动服务

### 1. 启动 Logic gRPC 服务

```bash
cd logic-grpc-service
go mod tidy
go run ./cmd/server
```

默认监听：`127.0.0.1:9090`

### 2. 启动 Web Gin 网关服务

```bash
cd web-gin-service
go mod tidy
go run ./cmd/server
```

默认监听：`http://localhost:8080`

### 3. 启动 HR 前端

```bash
cd hr-frontend
npm install
npm run dev
```

访问：`http://localhost:5173`

### 4. 启动候选人前端

```bash
cd user-frontend
npm install
npm run dev
```

访问：`http://localhost:5174`

## 四、推荐演示流程

1. 打开 HR 端，注册 HR 账号并登录。
2. HR 发布一个岗位，例如「Go 后端开发工程师」。
3. 打开候选人端，以游客身份查看公开岗位列表。
4. 注册候选人账号并登录。
5. 完善姓名、电话、邮箱、学历、学校、经历、技能等结构化档案。
6. 上传 PDF/DOC/DOCX 简历，前端通过签名 URL 直传私有 OSS。
7. 点击岗位一键投递。
8. 回到 HR 端，查看投递候选人。
9. 点击「签名下载」获取私有 OSS 简历短期访问链接。
10. 打开 HR AI 对话，提问：`哪个岗位投递最多？` 或 `当前总共有多少份投递？`

## 五、注意事项

- 本项目不考核 Docker，因此未提供 Docker Compose。
- 前端只负责页面渲染、交互和接口请求，不包含核心业务逻辑。
- Web 服务只负责网关职责，不写核心业务，不直接连接 MySQL，不直接调用大模型。
- Logic 服务独立部署，所有业务能力通过 gRPC 对外提供。
- `.env` 中包含敏感信息，不要提交到仓库。

