#!/bin/bash

# Let's Encrypt SSL 证书申请脚本
# 域名: examtest.top

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

DOMAIN="examtest.top"
EMAIL="admin@examtest.top"  # 修改为你的邮箱地址

# 检测 Nginx 配置文件路径（优先使用宝塔面板路径）
BT_NGINX_DIR="/www/server/panel/vhost/nginx"
BT_NGINX_CONF="${BT_NGINX_DIR}/html_www.examtest.top.conf"  # 宝塔面板配置文件格式
BT_NGINX_CONF_ALT="${BT_NGINX_DIR}/exam-practice.conf"  # 备用名称
TRADITIONAL_NGINX_CONF="/etc/nginx/sites-available/exam-practice.conf"

# 检测 Nginx 主配置文件路径（宝塔面板 vs 传统方式）
BT_NGINX_MAIN_CONF="/www/server/nginx/conf/nginx.conf"  # 宝塔面板主配置
TRADITIONAL_NGINX_MAIN_CONF="/etc/nginx/nginx.conf"  # 传统主配置
NGINX_MAIN_CONF=""

if [ -f "$BT_NGINX_MAIN_CONF" ]; then
    NGINX_MAIN_CONF="$BT_NGINX_MAIN_CONF"
    echo -e "${BLUE}检测到宝塔面板 Nginx 主配置: ${NGINX_MAIN_CONF}${NC}"
elif [ -f "$TRADITIONAL_NGINX_MAIN_CONF" ]; then
    NGINX_MAIN_CONF="$TRADITIONAL_NGINX_MAIN_CONF"
    echo -e "${BLUE}使用传统 Nginx 主配置: ${NGINX_MAIN_CONF}${NC}"
else
    # 如果都找不到，尝试自动检测
    NGINX_MAIN_CONF="$BT_NGINX_MAIN_CONF"  # 默认尝试宝塔面板路径
    echo -e "${YELLOW}未找到 Nginx 主配置文件，将使用默认路径${NC}"
fi

if [ -f "$BT_NGINX_CONF" ]; then
    NGINX_CONF="$BT_NGINX_CONF"
    echo -e "${BLUE}检测到宝塔面板 Nginx 配置: ${NGINX_CONF}${NC}"
elif [ -f "$BT_NGINX_CONF_ALT" ]; then
    NGINX_CONF="$BT_NGINX_CONF_ALT"
    echo -e "${BLUE}检测到宝塔面板 Nginx 配置（备用）: ${NGINX_CONF}${NC}"
elif [ -f "$TRADITIONAL_NGINX_CONF" ]; then
    NGINX_CONF="$TRADITIONAL_NGINX_CONF"
    echo -e "${BLUE}使用传统 Nginx 配置: ${NGINX_CONF}${NC}"
else
    NGINX_CONF="$BT_NGINX_CONF"  # 默认尝试宝塔面板路径
fi

CERT_PATH="/etc/letsencrypt/live/${DOMAIN}"

echo -e "${BLUE}=== Let's Encrypt SSL 证书申请工具 ===${NC}"
echo -e "${BLUE}域名: ${DOMAIN}${NC}"
echo ""

# 检查是否为 root 用户
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}错误: 请使用 root 权限运行此脚本${NC}"
    echo "使用: sudo $0"
    exit 1
fi

# 检查域名解析
echo -e "${BLUE}检查域名解析...${NC}"
SERVER_IP=$(curl -s ifconfig.me || curl -s ip.sb || echo "")
DOMAIN_IP=$(dig +short ${DOMAIN} @8.8.8.8 | tail -1)

if [ -z "$DOMAIN_IP" ]; then
    echo -e "${RED}错误: 无法解析域名 ${DOMAIN}${NC}"
    echo "请确保域名已正确配置 DNS 解析"
    exit 1
fi

echo -e "${GREEN}域名 ${DOMAIN} 解析到: ${DOMAIN_IP}${NC}"
if [ -n "$SERVER_IP" ]; then
    if [ "$DOMAIN_IP" != "$SERVER_IP" ]; then
        echo -e "${YELLOW}警告: 域名解析的 IP (${DOMAIN_IP}) 与当前服务器 IP (${SERVER_IP}) 不一致${NC}"
        echo -e "${YELLOW}请确保域名已正确解析到当前服务器${NC}"
        read -p "是否继续? (y/n): " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    else
        echo -e "${GREEN}域名解析正确${NC}"
    fi
fi

# 检查 Nginx 是否安装
echo -e "${BLUE}检查 Nginx...${NC}"
if ! command -v nginx &> /dev/null; then
    echo -e "${RED}错误: 未安装 Nginx${NC}"
    echo "安装方法:"
    echo "  Ubuntu/Debian: sudo apt-get install nginx"
    echo "  CentOS/RHEL: sudo yum install nginx"
    exit 1
fi
echo -e "${GREEN}Nginx 已安装${NC}"

# 检查 Nginx 配置
if [ ! -f "$NGINX_CONF" ]; then
    echo -e "${YELLOW}警告: 未找到 Nginx 配置文件: ${NGINX_CONF}${NC}"
    echo ""
    echo "尝试查找其他位置的配置文件..."
    
    # 尝试查找宝塔面板配置
    if [ -f "$BT_NGINX_CONF" ]; then
        NGINX_CONF="$BT_NGINX_CONF"
        echo -e "${GREEN}找到宝塔面板配置: ${NGINX_CONF}${NC}"
    elif [ -f "$BT_NGINX_CONF_ALT" ]; then
        NGINX_CONF="$BT_NGINX_CONF_ALT"
        echo -e "${GREEN}找到宝塔面板配置（备用）: ${NGINX_CONF}${NC}"
    # 尝试查找传统配置
    elif [ -f "$TRADITIONAL_NGINX_CONF" ]; then
        NGINX_CONF="$TRADITIONAL_NGINX_CONF"
        echo -e "${GREEN}找到传统配置: ${NGINX_CONF}${NC}"
    # 尝试查找其他可能的配置位置
    elif [ -f "/etc/nginx/conf.d/exam-practice.conf" ]; then
        NGINX_CONF="/etc/nginx/conf.d/exam-practice.conf"
        echo -e "${GREEN}找到配置: ${NGINX_CONF}${NC}"
    else
        echo -e "${RED}错误: 未找到 Nginx 配置文件${NC}"
        echo "请检查以下位置:"
        echo "  1. ${BT_NGINX_CONF} (宝塔面板 - 主)"
        echo "  2. ${BT_NGINX_CONF_ALT} (宝塔面板 - 备用)"
        echo "  3. ${TRADITIONAL_NGINX_CONF} (传统方式)"
        echo "  4. /etc/nginx/conf.d/exam-practice.conf"
        echo ""
        echo "请先部署应用并配置 Nginx"
        exit 1
    fi
fi

echo -e "${GREEN}使用 Nginx 配置文件: ${NGINX_CONF}${NC}"

# 检查 certbot 是否安装
echo -e "${BLUE}检查 certbot...${NC}"
if ! command -v certbot &> /dev/null; then
    echo -e "${YELLOW}未安装 certbot，正在安装...${NC}"
    
    # 检测操作系统
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
    else
        echo -e "${RED}无法检测操作系统类型${NC}"
        exit 1
    fi
    
    case $OS in
        ubuntu|debian)
            apt-get update
            apt-get install -y certbot python3-certbot-nginx
            ;;
        centos|rhel|fedora)
            yum install -y certbot python3-certbot-nginx || {
                # 如果 yum 失败，尝试 dnf
                dnf install -y certbot python3-certbot-nginx
            }
            ;;
        *)
            echo -e "${RED}不支持的操作系统: $OS${NC}"
            echo "请手动安装 certbot:"
            echo "  https://certbot.eff.org/"
            exit 1
            ;;
    esac
    
    echo -e "${GREEN}certbot 安装完成${NC}"
else
    echo -e "${GREEN}certbot 已安装${NC}"
fi

# 检查 Nginx 是否正在运行
echo -e "${BLUE}检查 Nginx 运行状态...${NC}"
NGINX_RUNNING=false

# 方法1: 检查 systemd 服务状态
if systemctl is-active --quiet nginx 2>/dev/null; then
    NGINX_RUNNING=true
    echo -e "${GREEN}Nginx 正在运行 (systemd)${NC}"
# 方法2: 检查进程
elif pgrep -x nginx > /dev/null 2>&1; then
    NGINX_RUNNING=true
    echo -e "${GREEN}Nginx 正在运行 (进程检查)${NC}"
# 方法3: 检查端口 80
elif netstat -tuln 2>/dev/null | grep -q ":80 " || ss -tuln 2>/dev/null | grep -q ":80 "; then
    NGINX_RUNNING=true
    echo -e "${GREEN}Nginx 正在运行 (端口检查)${NC}"
fi

# 如果 Nginx 未运行，尝试启动
if [ "$NGINX_RUNNING" = false ]; then
    echo -e "${YELLOW}Nginx 未运行，尝试启动...${NC}"
    if systemctl start nginx 2>/dev/null || service nginx start 2>/dev/null; then
        sleep 2
        # 再次检查是否启动成功
        if systemctl is-active --quiet nginx 2>/dev/null || pgrep -x nginx > /dev/null 2>&1; then
            echo -e "${GREEN}Nginx 启动成功${NC}"
            NGINX_RUNNING=true
        else
            echo -e "${YELLOW}警告: Nginx 启动可能失败，但继续尝试申请证书${NC}"
            echo -e "${YELLOW}如果证书申请失败，请手动启动 Nginx: sudo systemctl start nginx${NC}"
        fi
    else
        echo -e "${YELLOW}警告: 无法启动 Nginx，但继续尝试申请证书${NC}"
        echo -e "${YELLOW}如果证书申请失败，请手动启动 Nginx: sudo systemctl start nginx${NC}"
    fi
fi

# 检查防火墙
echo -e "${BLUE}检查防火墙...${NC}"
if command -v ufw &> /dev/null; then
    if ufw status | grep -q "Status: active"; then
        echo -e "${YELLOW}检测到 UFW 防火墙，确保端口 80 和 443 已开放${NC}"
        ufw allow 80/tcp
        ufw allow 443/tcp
    fi
elif command -v firewall-cmd &> /dev/null; then
    if firewall-cmd --state &>/dev/null; then
        echo -e "${YELLOW}检测到 firewalld，确保端口 80 和 443 已开放${NC}"
        firewall-cmd --permanent --add-service=http
        firewall-cmd --permanent --add-service=https
        firewall-cmd --reload
    fi
fi

# 申请证书
echo ""
echo -e "${BLUE}=== 开始申请 SSL 证书 ===${NC}"
echo -e "${BLUE}域名: ${DOMAIN}${NC}"
echo -e "${BLUE}邮箱: ${EMAIL}${NC}"
echo ""

# 使用 certbot 申请证书（自动配置 Nginx）
# 设置环境变量，确保 certbot 能找到正确的 nginx 配置
if [ -n "$NGINX_MAIN_CONF" ] && [ -f "$NGINX_MAIN_CONF" ]; then
    export NGINX_CONF="$NGINX_MAIN_CONF"
    echo -e "${BLUE}使用 Nginx 主配置文件: ${NGINX_MAIN_CONF}${NC}"
fi

# 如果检测到宝塔面板，设置 nginx 路径
CERTBOT_NGINX_ARGS=""
if [ -f "/www/server/nginx/sbin/nginx" ]; then
    export PATH="/www/server/nginx/sbin:$PATH"
    echo -e "${BLUE}检测到宝塔面板 Nginx，已添加到 PATH${NC}"
    # 为宝塔面板指定 nginx 配置目录
    CERTBOT_NGINX_ARGS="--nginx-server-root /www/server/nginx/conf"
    echo -e "${BLUE}使用宝塔面板 Nginx 配置目录: /www/server/nginx/conf${NC}"
fi

if certbot --nginx $CERTBOT_NGINX_ARGS -d ${DOMAIN} --non-interactive --agree-tos --email ${EMAIL} --redirect; then
    echo ""
    echo -e "${GREEN}=== SSL 证书申请成功！ ===${NC}"
    echo ""
    echo -e "${GREEN}证书路径:${NC}"
    echo "  证书文件: ${CERT_PATH}/fullchain.pem"
    echo "  私钥文件: ${CERT_PATH}/privkey.pem"
    echo ""
    echo -e "${BLUE}证书信息:${NC}"
    certbot certificates
    echo ""
    echo -e "${BLUE}测试证书配置:${NC}"
    # 使用正确的 nginx 配置文件路径
    if [ -n "$NGINX_MAIN_CONF" ] && [ -f "$NGINX_MAIN_CONF" ]; then
        nginx -c "$NGINX_MAIN_CONF" -t
    else
        # 如果找不到主配置，尝试默认路径或让 nginx 自动查找
        nginx -t 2>/dev/null || {
            echo -e "${YELLOW}警告: nginx -t 测试失败，尝试使用宝塔面板配置路径...${NC}"
            if [ -f "/www/server/nginx/conf/nginx.conf" ]; then
                nginx -c /www/server/nginx/conf/nginx.conf -t
            else
                echo -e "${YELLOW}跳过配置测试，直接重载 Nginx${NC}"
            fi
        }
    fi
    echo ""
    echo -e "${BLUE}重载 Nginx 配置:${NC}"
    systemctl reload nginx 2>/dev/null || service nginx reload 2>/dev/null || {
        echo -e "${YELLOW}警告: 无法通过 systemctl 重载，尝试使用宝塔面板命令...${NC}"
        # 宝塔面板可能使用不同的重载方式
        /www/server/nginx/sbin/nginx -s reload 2>/dev/null || echo -e "${YELLOW}请手动重载 Nginx 配置${NC}"
    }
    echo ""
    echo -e "${GREEN}证书已自动配置到 Nginx${NC}"
    echo -e "${GREEN}现在可以通过 https://${DOMAIN} 访问你的网站${NC}"
    echo ""
    echo -e "${BLUE}证书自动续期:${NC}"
    echo "Let's Encrypt 证书有效期为 90 天，certbot 会自动续期"
    echo "测试续期: sudo certbot renew --dry-run"
    echo ""
    echo -e "${GREEN}完成！${NC}"
else
    echo ""
    echo -e "${RED}=== SSL 证书申请失败 ===${NC}"
    echo ""
    echo "可能的原因:"
    echo "1. 域名未正确解析到当前服务器"
    echo "2. 端口 80 被防火墙阻止"
    echo "3. Nginx 配置有误"
    echo "4. 域名已申请过证书（需要先删除旧证书）"
    echo ""
    echo "手动申请命令:"
    echo "  sudo certbot --nginx -d ${DOMAIN} --email ${EMAIL}"
    echo ""
    echo "查看详细错误:"
    echo "  sudo certbot certificates"
    echo "  sudo tail -f /var/log/letsencrypt/letsencrypt.log"
    exit 1
fi

