# 数据库设计说明

本项目使用 MySQL 作为唯一业务数据库，Logic gRPC 服务通过 GORM 自动建表。Web Gin 网关不直接连接 MySQL，所有数据读写均通过 gRPC 调用 Logic 服务完成。

## 1. users

双角色账号表，支持多个 HR 账号、多个候选人账号。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uint | 主键 |
| username | varchar(64) | 登录账号，唯一 |
| password_hash | varchar(255) | bcrypt 密码哈希 |
| role | varchar(20) | `hr` 或 `candidate` |
| created_at / updated_at | datetime | 时间戳 |

## 2. jobs

岗位表，每个岗位只属于创建它的 HR。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uint | 主键 |
| hr_id | uint | 创建岗位的 HR 用户 ID |
| title | varchar(120) | 岗位名称 |
| description | text | 岗位描述 |
| requirements | text | 岗位要求 |
| salary | varchar(120) | 薪资描述 |
| location | varchar(120) | 工作地点 |
| status | varchar(20) | 默认 `open` |
| created_at / updated_at | datetime | 时间戳 |

## 3. candidate_profiles

候选人结构化档案表。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uint | 主键 |
| user_id | uint | 候选人账号 ID，唯一 |
| name | varchar(80) | 姓名 |
| phone | varchar(40) | 电话 |
| email | varchar(120) | 邮箱 |
| education | varchar(120) | 最高学历 |
| school | varchar(120) | 毕业院校 |
| experience | text | 工作/项目经历 |
| skills | text | 核心技能标签 |
| resume_object_key | varchar(255) | OSS 私有对象路径 |
| resume_file_name | varchar(255) | 原始文件名 |
| created_at / updated_at | datetime | 时间戳 |

## 4. applications

岗位投递关联表。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uint | 主键 |
| job_id | uint | 岗位 ID |
| candidate_id | uint | 候选人用户 ID |
| status | varchar(30) | 默认 `submitted` |
| created_at / updated_at | datetime | 时间戳 |

唯一约束：`job_id + candidate_id`，防止同一候选人重复投递同一岗位。

## 5. ai_chat_messages

HR AI 对话历史表。

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| id | uint | 主键 |
| hr_id | uint | HR 用户 ID |
| role | varchar(20) | `user` 或 `assistant` |
| content | text | 消息内容 |
| created_at | datetime | 创建时间 |

AI 问答不使用向量库、不使用 RAG。系统先从 MySQL 汇总岗位数、投递数、岗位热度等真实业务数据，再通过 Eino ChatModel 生成自然语言回答。
