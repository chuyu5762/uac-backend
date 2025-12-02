# 统一认证中心后端 (UAC Backend)

企业级身份认证和访问管理（IAM）系统后端服务。

## 技术栈

- Go 1.25.0+
- Gin 1.10.0+
- GORM 1.25.0+
- PostgreSQL 16+ / MySQL 5.7+
- Redis 7.0+

## 项目结构

```
├── cmd/
│   └── server/          # 主程序入口
├── internal/
│   ├── config/          # 配置管理
│   ├── handler/         # HTTP 处理器
│   ├── middleware/      # 中间件
│   ├── model/           # 数据模型
│   ├── repository/      # 数据访问层
│   └── service/         # 业务逻辑层
├── pkg/
│   ├── response/        # 响应封装
│   ├── jwt/             # JWT 工具
│   └── crypto/          # 加密工具
├── configs/             # 配置文件
└── migrations/          # 数据库迁移
```

## 快速开始

### 1. 安装依赖

```bash
go mod download
```

### 2. 配置数据库

复制配置文件并修改数据库连接信息：

```bash
cp configs/config.yaml configs/config.local.yaml
```

### 3. 运行服务

```bash
go run cmd/server/main.go
```

### 4. 访问健康检查

```bash
curl http://localhost:8080/health
```

## API 响应格式

所有 API 响应遵循统一格式：

```json
{
  "code": 0,
  "msg": "操作成功",
  "data": {}
}
```

## 开发规范

- 错误信息使用中文
- 注释使用中文
- 每个文件不超过 1000 行
- 遵循面向失败开发原则

## 许可证

MIT
