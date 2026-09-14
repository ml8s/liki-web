# 前端页面设计流程

新增或重构一个页面时，按以下顺序工作，避免 UI、状态和业务规则混在页面里。

## 1. 定义用户任务

先回答：

- 用户来这个页面完成什么？
- 关键输入是什么？
- 成功路径是什么？
- 失败、无权限、无数据时怎么办？

## 2. 定义路由与权限

写入 `docs/frontend/route-matrix.md`：

```text
route
auth 要求
entitlement / owner 要求
primary APIs
状态
```

前端 guard 只改善体验；真正的 owner 和 entitlement 校验必须由后端 API 执行。

## 3. 定义状态矩阵

每个页面至少列：

```text
loading
empty
success
forbidden
unauthenticated
network error
server error
action pending
action failed
```

## 4. 定义 API contract

明确：

- endpoint；
- request/response 类型；
- 错误 code；
- 分页/limit；
- 是否是 SSE；
- 哪些字段前端只读。

前端类型写在 `src/api/*.ts`，不要在页面里手写散落对象。

## 5. 拆分组件

页面只组合：

```text
AppPageHeader
AppAlert
AppEmpty
AppSkeleton
AppErrorState
业务组件
```

业务组件通过 props/emits 交互，不直接绕过 store 调用后端。

## 6. 实现状态流

按状态逐个实现，不先只做 happy path。

## 7. 测试

- API client：envelope、错误、SSE；
- Store：状态转换；
- Component：展示和 emits；
- E2E：完整用户路径。

## 8. 文档

更新：

- route matrix；
- page spec；
- state/API mapping；
- 组件目录说明（如新增通用组件）。
