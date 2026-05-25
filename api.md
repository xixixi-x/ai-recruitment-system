# API 接口说明

基础地址：`http://localhost:8080/api`

返回格式：

```json
{
  "code": 0,
  "message": "ok",
  "data": {}
}
```

除公开岗位列表与登录注册外，其他接口需要请求头：

```http
Authorization: Bearer <JWT_TOKEN>
```

## 一、认证接口

### 1. 注册

`POST /auth/register`

```json
{
  "username": "hr01",
  "password": "123456",
  "role": "hr"
}
```

`role` 可取：`hr`、`candidate`。

### 2. 登录

`POST /auth/login`

```json
{
  "username": "hr01",
  "password": "123456",
  "role": "hr"
}
```

返回：

```json
{
  "token": "JWT...",
  "user": {"id": 1, "username": "hr01", "role": "hr"}
}
```

## 二、公开岗位接口

### 1. 游客查看岗位

`GET /jobs/public?page=1&pageSize=20&keyword=go`

游客无需登录即可查看全平台公开岗位。

## 三、HR 管理端接口

### 1. 新增岗位

`POST /hr/jobs`

```json
{
  "title": "Go 后端开发工程师",
  "description": "负责招聘系统后端开发",
  "requirements": "熟悉 Gin、gRPC、MySQL",
  "salary": "15k-25k",
  "location": "武汉"
}
```

### 2. 查看当前 HR 创建的岗位

`GET /hr/jobs`

### 3. 查看当前 HR 岗位收到的投递

`GET /hr/applications`

只返回当前 HR 自己创建岗位下的候选人投递。

### 4. 查看单条投递详情

`GET /hr/applications/:id`

### 5. 获取简历签名下载 URL

`GET /hr/applications/:id/resume-url`

返回私有 OSS 签名下载链接，链接短期有效，后端不会暴露 Bucket 公共读权限。

### 6. AI 对话

`POST /hr/ai/chat`

```json
{
  "question": "哪个岗位投递最多？"
}
```

处理流程：HR 输入自然语言问题 → Logic 服务查询 MySQL 真实业务统计 → 使用 Eino ChatModel 生成自然语言回答 → 对话历史写入 MySQL。

### 7. AI 历史对话

`GET /hr/ai/history`

## 四、候选人用户端接口

### 1. 获取个人档案

`GET /candidate/profile`

### 2. 保存结构化档案

`PUT /candidate/profile`

```json
{
  "name": "张三",
  "phone": "13800000000",
  "email": "zhangsan@example.com",
  "education": "本科",
  "school": "武汉科技大学",
  "experience": "完成过 Go + React 项目",
  "skills": "Go, Gin, MySQL, React"
}
```

### 3. 获取简历签名上传 URL

`POST /candidate/resume/sign-upload`

```json
{
  "filename": "resume.pdf",
  "contentType": "application/pdf",
  "size": 102400
}
```

后端仅允许 `.pdf`、`.doc`、`.docx`，拒绝图片、压缩包、TXT 等非法格式。

### 4. 确认简历上传完成

`POST /candidate/resume/confirm`

```json
{
  "objectKey": "resumes/candidate_1/xxx_resume.pdf",
  "filename": "resume.pdf"
}
```

### 5. 一键投递岗位

`POST /candidate/jobs/:id/apply`

投递前强制校验：候选人必须登录、必须完善姓名/电话/邮箱、必须上传合规简历。

### 6. 查看我的投递

`GET /candidate/applications`
