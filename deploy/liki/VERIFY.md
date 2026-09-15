# 部署验收清单（skill 规则引擎工具化）

本清单在 liki.hk 发布（`docker compose -f deploy/liki/docker-compose.yml up -d --build`）后逐项验证。

## 1. 容器内 python3 可用（Dockerfile 已加 `apk add python3`）

```bash
docker compose -f deploy/liki/docker-compose.yml exec web python3 -c \
  "import json,os,urllib.request,typing,dataclasses,collections,csv; print('标准库 OK')"
```
期望：`标准库 OK`（纯标准库，无第三方依赖）。

## 2. skill 工具文件已分发

```bash
docker compose -f deploy/liki/docker-compose.yml exec web ls /web/skills/liki/bazi/tools/
```
期望包含：`agent_cli.py` `skill-tools.json` `paipan.py` `duanyu.py` `atoms.py` `engine.py` `aggregate.py` + `factors/` `bazi/` `ziwei/`。

## 3. python 子进程内网直连引擎（LIKI_RPC_URL）

```bash
docker compose -f deploy/liki/docker-compose.yml exec web \
  sh -c 'echo "{\"fn\":\"make_factors\",\"args\":{\"pan\":{}}}" | LIKI_RPC_URL=http://engine:8082/jsonrpc python3 /web/skills/liki/bazi/tools/agent_cli.py'
```
期望：`{"ok": false, "error": "missing arg: 'fac'"}`（错误包装正常，说明 agent_cli 可执行）。

## 4. 引擎 RPC 可达（python 子进程角度）

```bash
docker compose -f deploy/liki/docker-compose.yml exec web \
  sh -c 'echo "{\"fn\":\"full_paipan\",\"args\":{\"time\":\"1990-06-01T12:00:00+08:00\",\"gender\":\"male\",\"correct\":false}}" | LIKI_RPC_URL=http://engine:8082/jsonrpc python3 /web/skills/liki/bazi/tools/agent_cli.py' | head -c 200
```
期望：`{"ok": true, "data": {"solar": ..., "chart": ...}}`（full_paipan 经引擎排盘成功）。

## 5. 真实 LLM 会话冒烟（手动，需 DeepSeek key 已配）

在 liki.hk 聊天页发一条命理问题（如"1990年6月1日中午出生男，问什么时候结婚"），观察：
1. 服务端日志出现 `registered skill tools`（启动时 5 工具注册成功）
2. 会话中 LLM 调用 `full_paipan` → `make_factors`/`make_liunian_factors` → `query`（代理日志/响应含规则引擎断语）
3. 应答含真值表断语（如"婚姻域 ymar_1xx 命中"），非"未走规则引擎"降级标注

## 回滚

若规则引擎异常：删除 main.go 中 skillTools.Register 段重新构建，或环境变量禁用（未实现则改代码）——agent 回落到无 5 工具状态（read_file + 引擎 RPC 仍可用）。
