#!/usr/bin/env bash
set -euo pipefail

cat <<'EOF'
NginxPulse npm 脚本总览（在仓库根目录执行）

基础开发
- npm run dev:local
  启动本地联调环境（后端 + 前端）
- npm run build:single
  打包单文件版本

文档相关
- npm run docs:sync
  同步 docs/wiki 到 docs/fumadocs/content/docs
- npm run docs:dev
  启动 fumadocs 本地开发服务
- npm run docs:build
  构建 fumadocs，并输出 dist/nginxpulse-docs 最新产物

发布相关
- npm run docker:publish -- [-v <version>] [-p <platforms>] [--no-push]
  发布正式 Docker 镜像（默认仓库: magiccoders/nginxpulse）
- npm run docker:publish:beta -- [-v <version>] [-p <platforms>] [--no-push]
  发布 Beta Docker 镜像（默认仓库: magiccoders/nginxpulse）
- npm run docker:verify-latest -- -v <version> [-l <latest_tag>]
  校验 latest（或指定 tag）是否指向目标版本（默认仓库: magiccoders/nginxpulse）
- npm run wiki:push
  推送 docs/wiki 到 GitHub Wiki

资源与工具
- npm run assets:crop-supporters -- <image1> [image2 ...]
  生成圆形头像 PNG
- npm run db:schema:render
  渲染 PG schema 文档
- npm run website:ids
  生成/查看网站 ID 工具

前端快捷
- npm run web:dev | web:build | web:preview
  Web 前端开发/构建/预览
- npm run mobile:dev | mobile:build | mobile:preview
  移动端前端开发/构建/预览
- npm run fumadocs:dev | fumadocs:build
  文档站开发/构建
EOF
