#!/bin/bash

# Linux x86_64 Nginx 部署打包脚本
# 前端构建为静态文件 + 后端二进制文件

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 配置
PACKAGE_NAME="exam-nginx-linux-x64-$(date +%Y%m%d-%H%M%S)"
TEMP_DIR="build-temp"
DIST_DIR="$PACKAGE_NAME"

echo -e "${BLUE}=== 智能刷题平台 Nginx 部署打包工具 ===${NC}"
echo -e "${BLUE}开始打包项目（前端静态文件 + 后端二进制）...${NC}"

# 检查环境
echo -e "${BLUE}检查构建环境...${NC}"

# 检查 Node.js
if ! command -v node &> /dev/null; then
    echo -e "${RED}错误: 未找到 Node.js，请先安装 Node.js${NC}"
    exit 1
fi

# 检查 Go
if ! command -v go &> /dev/null; then
    echo -e "${RED}错误: 未找到 Go，请先安装 Go 语言环境${NC}"
    exit 1
fi

# 检查项目文件
if [ ! -f "package.json" ] || [ ! -d "go-server" ] || [ ! -d "src" ]; then
    echo -e "${RED}错误: 请在项目根目录运行此脚本${NC}"
    exit 1
fi

# 清理旧的构建文件
echo -e "${BLUE}清理旧的构建文件...${NC}"
rm -rf "$TEMP_DIR" "$DIST_DIR" "${PACKAGE_NAME}.tar.gz" dist

# 创建构建目录
mkdir -p "$TEMP_DIR"
mkdir -p "$DIST_DIR"

echo -e "${BLUE}构建前端静态文件...${NC}"
# 安装前端依赖
npm install

# 配置生产环境 API 地址
echo -e "${YELLOW}配置生产环境 API 地址...${NC}"
# 备份原文件
cp src/api/index.js src/api/index.js.backup

# 替换 API 地址为相对路径（通过 Nginx 代理）
sed -i.tmp "s|const API_BASE_URL = 'http://localhost:3005/api'|const API_BASE_URL = '/api'|g" src/api/index.js
rm -f src/api/index.js.tmp

echo -e "${GREEN}API 地址已配置为相对路径 '/api'${NC}"

# 构建前端
npm run build

# 恢复原文件
mv src/api/index.js.backup src/api/index.js

# 检查构建结果
if [ ! -d "dist" ]; then
    echo -e "${RED}错误: 前端构建失败，未找到 dist 目录${NC}"
    exit 1
fi

echo -e "${GREEN}前端构建完成${NC}"

echo -e "${BLUE}交叉编译 Go 后端到 Linux x86_64...${NC}"
cd go-server

# 设置 Go 交叉编译环境变量
export GOOS=linux
export GOARCH=amd64
export CGO_ENABLED=0

# 编译 Go 程序
go mod tidy
go build -ldflags="-s -w" -o ../exam-server-linux-x64 .

cd ..

# 检查编译结果
if [ ! -f "exam-server-linux-x64" ]; then
    echo -e "${RED}错误: Go 后端编译失败${NC}"
    exit 1
fi

echo -e "${GREEN}Go 后端编译完成${NC}"

echo -e "${BLUE}打包文件...${NC}"

# 创建目录结构
mkdir -p "$DIST_DIR/frontend"
mkdir -p "$DIST_DIR/backend"
mkdir -p "$DIST_DIR/nginx"

# 复制前端构建文件到 frontend 目录
cp -r dist/* "$DIST_DIR/frontend/"

# 复制后端二进制文件
cp exam-server-linux-x64 "$DIST_DIR/backend/"

# 复制数据库文件
if [ -f "go-server/exam.db" ]; then
    cp go-server/exam.db "$DIST_DIR/backend/"
fi

# 复制配置文件
if [ -f "go-server/.env" ]; then
    cp go-server/.env "$DIST_DIR/backend/"
fi

# 复制示例数据
if [ -f "sample-questions.json" ]; then
    cp sample-questions.json "$DIST_DIR/backend/"
fi

# 创建 Nginx 配置文件
cat > "$DIST_DIR/nginx/exam-practice.conf" << 'EOF'
server {
    listen 80;
    server_name localhost;  # 修改为你的域名
    
    # 前端静态文件
    location / {
        root /path/to/exam-practice/frontend;  # 修改为实际路径
        index index.html;
        try_files $uri $uri/ /index.html;
        
        # 静态资源缓存
        location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
            expires 1y;
            add_header Cache-Control "public, immutable";
        }
    }
    
    # 后端 API 代理
    location /api/ {
        proxy_pass http://127.0.0.1:3005;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        # WebSocket 支持（如果需要）
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
    }
    
    # 健康检查
    location /health {
        proxy_pass http://127.0.0.1:3005;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
    }
    
    # 安全配置
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header Referrer-Policy "no-referrer-when-downgrade" always;
    add_header Content-Security-Policy "default-src 'self' http: https: data: blob: 'unsafe-inline'" always;
    
    # 日志
    access_log /var/log/nginx/exam-practice.access.log;
    error_log /var/log/nginx/exam-practice.error.log;
}
EOF

# 创建后端启动脚本
cat > "$DIST_DIR/backend/start-backend.sh" << 'EOF'
#!/bin/bash

# 后端服务启动脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# 配置
PORT=3005
BINARY_NAME="exam-server-linux-x64"
PID_FILE=".exam-server.pid"
LOG_FILE="exam-server.log"

# 检查二进制文件
if [ ! -f "$BINARY_NAME" ]; then
    echo -e "${RED}错误: 未找到服务器程序 $BINARY_NAME${NC}"
    exit 1
fi

# 设置执行权限
chmod +x "$BINARY_NAME"

# 检查端口
if netstat -tuln 2>/dev/null | grep -q ":$PORT " || ss -tuln 2>/dev/null | grep -q ":$PORT "; then
    echo -e "${YELLOW}警告: 端口 $PORT 已被占用${NC}"
    echo "请检查是否有其他服务正在运行"
    
    # 显示占用端口的进程
    echo "占用端口的进程："
    netstat -tuln 2>/dev/null | grep ":$PORT " || ss -tuln 2>/dev/null | grep ":$PORT "
    exit 1
fi

echo -e "${BLUE}启动后端服务...${NC}"
echo -e "${BLUE}服务端口: $PORT${NC}"
echo -e "${BLUE}API地址: http://localhost:$PORT/api${NC}"
echo -e "${BLUE}健康检查: http://localhost:$PORT/health${NC}"
echo -e "${BLUE}日志文件: $LOG_FILE${NC}"
echo ""

# 启动服务器
nohup ./"$BINARY_NAME" > "$LOG_FILE" 2>&1 &
SERVER_PID=$!

# 保存 PID
echo $SERVER_PID > "$PID_FILE"

echo -e "${GREEN}后端服务已启动 (PID: $SERVER_PID)${NC}"
echo ""
echo "管理命令:"
echo "  查看日志: tail -f $LOG_FILE"
echo "  停止服务: kill $SERVER_PID 或 ./stop-backend.sh"
echo "  查看状态: ps -p $SERVER_PID"
echo ""
echo -e "${GREEN}后端服务启动完成！${NC}"

# 等待服务启动
sleep 2

# 检查服务是否正常启动
if ps -p $SERVER_PID > /dev/null 2>&1; then
    echo -e "${GREEN}✓ 服务运行正常${NC}"
    
    # 测试健康检查
    if command -v curl &> /dev/null; then
        echo "测试健康检查..."
        if curl -s http://localhost:$PORT/health > /dev/null; then
            echo -e "${GREEN}✓ 健康检查通过${NC}"
        else
            echo -e "${YELLOW}⚠ 健康检查失败，请检查日志${NC}"
        fi
    fi
else
    echo -e "${RED}✗ 服务启动失败，请检查日志${NC}"
    exit 1
fi
EOF

# 创建后端停止脚本
cat > "$DIST_DIR/backend/stop-backend.sh" << 'EOF'
#!/bin/bash

# 后端服务停止脚本

PID_FILE=".exam-server.pid"
PORT=3005

echo "停止后端服务..."

if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p $PID > /dev/null 2>&1; then
        echo "正在停止服务器 (PID: $PID)..."
        kill $PID
        
        # 等待进程结束
        sleep 2
        
        # 检查是否已经停止
        if ps -p $PID > /dev/null 2>&1; then
            echo "强制停止服务器..."
            kill -9 $PID
        fi
        
        rm -f "$PID_FILE"
        echo "服务器已停止"
    else
        echo "服务器进程不存在"
        rm -f "$PID_FILE"
    fi
else
    echo "未找到 PID 文件，尝试通过端口停止..."
    
    # 通过端口查找并停止进程
    PIDS=$(netstat -tuln 2>/dev/null | grep ":$PORT " | awk '{print $7}' | cut -d'/' -f1 2>/dev/null)
    if [ -n "$PIDS" ]; then
        echo "找到占用端口 $PORT 的进程: $PIDS"
        echo $PIDS | xargs kill -9 2>/dev/null
        echo "已停止占用端口的进程"
    else
        echo "未找到占用端口 $PORT 的进程"
    fi
fi

echo "后端服务停止完成"
EOF

# 创建后端状态检查脚本
cat > "$DIST_DIR/backend/status-backend.sh" << 'EOF'
#!/bin/bash

# 后端服务状态检查脚本

PID_FILE=".exam-server.pid"
PORT=3005
LOG_FILE="exam-server.log"

echo "=== 后端服务状态 ==="

# 检查进程
if [ -f "$PID_FILE" ]; then
    PID=$(cat "$PID_FILE")
    if ps -p $PID > /dev/null 2>&1; then
        echo "✓ 服务正在运行 (PID: $PID)"
        
        # 显示资源使用情况
        echo ""
        echo "资源使用情况:"
        ps -p $PID -o pid,ppid,pcpu,pmem,etime,cmd --no-headers
    else
        echo "✗ 服务进程不存在"
        rm -f "$PID_FILE"
    fi
else
    echo "○ 服务未运行"
fi

# 检查端口
echo ""
echo "端口状态:"
if netstat -tuln 2>/dev/null | grep -q ":$PORT " || ss -tuln 2>/dev/null | grep -q ":$PORT "; then
    echo "✓ 端口 $PORT 正在监听"
    
    # 测试健康检查
    if command -v curl &> /dev/null; then
        echo ""
        echo "健康检查:"
        if curl -s http://localhost:$PORT/health > /dev/null; then
            echo "✓ 健康检查通过"
        else
            echo "✗ 健康检查失败"
        fi
    fi
else
    echo "✗ 端口 $PORT 未在监听"
fi

# 显示日志
echo ""
echo "日志文件:"
if [ -f "$LOG_FILE" ]; then
    echo "  $LOG_FILE ($(wc -l < $LOG_FILE) 行)"
    echo "  最后 5 行日志:"
    tail -5 "$LOG_FILE" | sed 's/^/    /'
else
    echo "  无日志文件"
fi
EOF

# 创建部署脚本
cat > "$DIST_DIR/deploy.sh" << 'EOF'
#!/bin/bash

# 智能刷题平台部署脚本

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}=== 智能刷题平台部署脚本 ===${NC}"
echo ""

# 检查系统
echo -e "${BLUE}检查系统环境...${NC}"

# 检查架构
ARCH=$(uname -m)
if [ "$ARCH" != "x86_64" ]; then
    echo -e "${YELLOW}警告: 当前系统架构为 $ARCH，建议使用 x86_64 架构${NC}"
fi

# 检查 Nginx
if ! command -v nginx &> /dev/null; then
    echo -e "${YELLOW}警告: 未安装 Nginx${NC}"
    echo "安装方法："
    echo "  Ubuntu/Debian: sudo apt-get install nginx"
    echo "  CentOS/RHEL: sudo yum install nginx"
    echo ""
fi

# 获取当前路径
CURRENT_PATH=$(pwd)
FRONTEND_PATH="$CURRENT_PATH/frontend"
BACKEND_PATH="$CURRENT_PATH/backend"

echo -e "${BLUE}部署信息:${NC}"
echo "  前端路径: $FRONTEND_PATH"
echo "  后端路径: $BACKEND_PATH"
echo "  Nginx配置: $CURRENT_PATH/nginx/exam-practice.conf"
echo ""

# 设置脚本执行权限
echo -e "${BLUE}设置执行权限...${NC}"
chmod +x backend/*.sh

echo -e "${BLUE}部署步骤:${NC}"
echo ""
echo "1. 配置 Nginx:"
echo "   sudo cp nginx/exam-practice.conf /etc/nginx/sites-available/"
echo "   sudo ln -sf /etc/nginx/sites-available/exam-practice.conf /etc/nginx/sites-enabled/"
echo "   # 编辑配置文件，修改前端路径为: $FRONTEND_PATH"
echo "   sudo nginx -t"
echo "   sudo systemctl reload nginx"
echo ""
echo "2. 启动后端服务:"
echo "   cd backend"
echo "   ./start-backend.sh"
echo ""
echo "3. 访问应用:"
echo "   http://localhost (通过 Nginx)"
echo "   http://localhost:3005/health (后端健康检查)"
echo ""
echo "4. 管理服务:"
echo "   cd backend"
echo "   ./status-backend.sh  # 查看状态"
echo "   ./stop-backend.sh    # 停止服务"
echo ""

echo -e "${GREEN}部署准备完成！${NC}"
echo -e "${YELLOW}请按照上述步骤完成部署${NC}"
EOF

# 创建 README 文档
cat > "$DIST_DIR/README.md" << 'EOF'
# 智能刷题平台 - Nginx 部署版

## 架构说明

这是一个前后端分离的 Nginx 部署方案：
- **前端**: 静态文件，通过 Nginx 提供服务
- **后端**: Go 二进制文件，提供 API 服务
- **反向代理**: Nginx 将 API 请求代理到后端服务

## 目录结构

```
./
├── frontend/              # 前端静态文件
│   ├── index.html        # 入口页面
│   └── assets/           # 静态资源
├── backend/              # 后端服务
│   ├── exam-server-linux-x64  # Go 二进制文件
│   ├── exam.db          # SQLite 数据库
│   ├── start-backend.sh # 启动脚本
│   ├── stop-backend.sh  # 停止脚本
│   └── status-backend.sh # 状态检查
├── nginx/               # Nginx 配置
│   └── exam-practice.conf # 站点配置
├── deploy.sh           # 部署脚本
└── README.md           # 本文档
```

## 系统要求

- Linux x86_64 系统
- Nginx 1.14+
- 可用端口 80（前端）和 3005（后端）
- 至少 100MB 磁盘空间
- 至少 256MB 内存

## 快速部署

### 1. 解压文件
```bash
tar -xzf exam-nginx-linux-x64-*.tar.gz
cd exam-nginx-linux-x64-*
```

### 2. 运行部署脚本
```bash
./deploy.sh
```

### 3. 配置 Nginx
```bash
# 复制配置文件
sudo cp nginx/exam-practice.conf /etc/nginx/sites-available/

# 创建软链接
sudo ln -sf /etc/nginx/sites-available/exam-practice.conf /etc/nginx/sites-enabled/

# 编辑配置文件，修改前端路径
sudo nano /etc/nginx/sites-available/exam-practice.conf
# 将 /path/to/exam-practice/frontend 修改为实际的前端路径

# 测试配置
sudo nginx -t

# 重载 Nginx
sudo systemctl reload nginx
```

### 4. 启动后端服务
```bash
cd backend
./start-backend.sh
```

### 5. 访问应用
- 前端页面: http://localhost
- 后端API: http://localhost:3005
- 健康检查: http://localhost/health

## 服务管理

### 后端服务管理
```bash
cd backend

# 启动服务
./start-backend.sh

# 停止服务
./stop-backend.sh

# 查看状态
./status-backend.sh

# 查看日志
tail -f exam-server.log
```

### Nginx 管理
```bash
# 检查配置
sudo nginx -t

# 重载配置
sudo systemctl reload nginx

# 重启 Nginx
sudo systemctl restart nginx

# 查看状态
sudo systemctl status nginx

# 查看日志
sudo tail -f /var/log/nginx/exam-practice.access.log
sudo tail -f /var/log/nginx/exam-practice.error.log
```

## 环境安装

### Ubuntu/Debian
```bash
# 安装 Nginx
sudo apt-get update
sudo apt-get install nginx

# 启动 Nginx
sudo systemctl start nginx
sudo systemctl enable nginx
```

### CentOS/RHEL
```bash
# 安装 Nginx
sudo yum install nginx

# 启动 Nginx
sudo systemctl start nginx
sudo systemctl enable nginx
```

## 配置说明

### Nginx 配置要点
- 前端静态文件服务
- API 请求反向代理到后端
- 静态资源缓存优化
- 安全头设置
- 日志记录

### 后端配置
- 端口: 3005
- 数据库: SQLite (exam.db)
- 日志: exam-server.log

## 故障排除

### 前端无法访问
```bash
# 检查 Nginx 状态
sudo systemctl status nginx

# 检查配置文件
sudo nginx -t

# 检查前端文件路径
ls -la frontend/

# 查看 Nginx 错误日志
sudo tail -f /var/log/nginx/error.log
```

### 后端 API 错误
```bash
# 检查后端服务状态
cd backend && ./status-backend.sh

# 查看后端日志
cd backend && tail -f exam-server.log

# 检查端口占用
netstat -tuln | grep 3005
```

### 权限问题
```bash
# 设置文件权限
chmod +x backend/*.sh
chmod +x backend/exam-server-linux-x64

# 检查 Nginx 用户权限
sudo chown -R www-data:www-data frontend/  # Ubuntu/Debian
sudo chown -R nginx:nginx frontend/        # CentOS/RHEL
```

## 性能优化

### Nginx 优化
- 启用 gzip 压缩
- 设置静态文件缓存
- 调整 worker 进程数
- 配置连接池

### 后端优化
- 调整数据库连接池
- 启用 Go 程序的生产模式
- 配置日志轮转

## 安全建议

1. **防火墙配置**
   ```bash
   # 只开放必要端口
   sudo ufw allow 80
   sudo ufw allow 443
   sudo ufw deny 3005  # 后端端口不对外开放
   ```

2. **SSL 配置**
   - 配置 HTTPS 证书
   - 强制 HTTPS 重定向
   - 设置安全头

3. **访问控制**
   - 限制 API 访问频率
   - 配置 IP 白名单
   - 启用访问日志

## 监控和维护

### 日志监控
```bash
# 实时监控访问日志
sudo tail -f /var/log/nginx/exam-practice.access.log

# 实时监控后端日志
cd backend && tail -f exam-server.log
```

### 定期维护
- 定期备份数据库文件
- 清理旧日志文件
- 更新系统安全补丁
- 监控磁盘空间使用

## 技术支持

如有问题请：
1. 检查相关日志文件
2. 验证配置文件语法
3. 确认服务运行状态
4. 联系技术支持团队
EOF

# 设置脚本执行权限
chmod +x "$DIST_DIR"/*.sh
chmod +x "$DIST_DIR/backend"/*.sh

echo -e "${BLUE}创建压缩包...${NC}"
tar -czf "${PACKAGE_NAME}.tar.gz" "$DIST_DIR"

# 清理临时文件
rm -rf "$TEMP_DIR" "$DIST_DIR" exam-server-linux-x64

# 显示结果
echo ""
echo -e "${GREEN}=== Nginx 部署打包完成 ===${NC}"
echo -e "${GREEN}打包文件: ${PACKAGE_NAME}.tar.gz${NC}"
echo -e "${GREEN}文件大小: $(du -h "${PACKAGE_NAME}.tar.gz" | cut -f1)${NC}"
echo ""
echo -e "${BLUE}部署架构:${NC}"
echo "  前端: 静态文件 → Nginx (端口 80)"
echo "  后端: Go 二进制 → API 服务 (端口 3005)"
echo "  代理: Nginx → 后端 API"
echo ""
echo -e "${BLUE}部署到 Linux x86_64 系统的步骤:${NC}"
echo "1. 上传文件: scp ${PACKAGE_NAME}.tar.gz user@server:/path/"
echo "2. 解压文件: tar -xzf ${PACKAGE_NAME}.tar.gz"
echo "3. 进入目录: cd ${PACKAGE_NAME}"
echo "4. 运行部署: ./deploy.sh"
echo "5. 配置 Nginx: 按照提示修改配置文件路径"
echo "6. 启动后端: cd backend && ./start-backend.sh"
echo "7. 访问应用: http://your-server-ip"
echo ""
echo -e "${YELLOW}注意事项:${NC}"
echo "- 确保目标系统已安装 Nginx"
echo "- 确保端口 80 和 3005 未被占用"
echo "- 需要 root 权限配置 Nginx"
echo "- 建议配置 SSL 证书启用 HTTPS"
echo ""
echo -e "${GREEN}打包完成！${NC}"