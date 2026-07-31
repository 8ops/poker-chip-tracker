# 棋牌筹码记录器

Go 后端 + React 前端，单二进制部署（SQLite，无外部数据库）。

## 开发

### 后端

```powershell
$env:Path = "C:\Program Files\Go\bin;" + $env:Path
$env:GOPROXY = "https://goproxy.cn,direct"
go test ./...
go run .
```

### 前端（Vite 开发服务器，API 代理到 :8080）

```powershell
cd frontend
npm install
npm run dev
```

打开 http://localhost:5173

## 生产构建（嵌入 React 到 Go）

```powershell
cd frontend
npm install
npm run build
# 将产物复制到 Go embed 目录
Remove-Item -Recurse -Force ..\web\* -ErrorAction SilentlyContinue
Copy-Item -Recurse dist\* ..\web\
cd ..
go build -o poker-chip-tracker.exe .
.\poker-chip-tracker.exe
```

访问 http://localhost:8080

## Docker（一键构建前后端）

```powershell
docker compose up -d --build
```

## 测试

```powershell
go test ./... -v
```

覆盖：房间创建、成员管理、筹码平衡校验、关闭房间拒绝写入、历史统计、REST API 全流程。
