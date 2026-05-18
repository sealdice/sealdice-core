# 数据拉取最佳实践 (TanStack Query + heyapi)

本文档总结了项目中使用 `@tanstack/vue-query` + `@hey-api/openapi-ts` 进行数据拉取的最佳实践和常见模式。

> **技术栈版本：**
> - `@tanstack/vue-query`: ^5.x
> - `@hey-api/openapi-ts`: ^0.91.x（内置 Fetch client，无需单独安装）

---

## 🚨 核心原则（必读）

### 生成函数命名（了解一下）

Hey API 会根据 OpenAPI `operationId` 生成以下函数（以下为示例）：

- **GET 查询**：`getUserInfoOptions()` / `getUserInfoQueryKey()`
- **Mutation**：`postUserInfoUpdateMutation()`
- **直接请求**：`getUserInfo()` / `postUserInfoUpdate()`

> 本文中的 `getUserDetailOptions` / `getUserSearchOptions` 等为示例命名，实际请以生成器输出的 `*Options` / `*QueryKey` 为准。

### 原则 1：必须使用生成的 API 函数，禁止使用 Store

所有 API 请求必须直接使用 heyapi 生成的函数，**禁止**通过 Store 包装调用。

```typescript
import { getUserInfoOptions, postUserInfoUpdateMutation } from '@/api'

// ✅ 正确：使用生成的函数
const { data } = useQuery(getUserInfoOptions())

// ✅ 正确：mutation 使用生成的函数
const { mutate } = useMutation(postUserInfoUpdateMutation())
// mutate({ body: data })

// ❌ 错误：不要使用 Store
import { useUserStore } from '@/stores/user'
const userStore = useUserStore()
const result = await userStore.updateUser(data)  // 禁止！
```

**原因：**
- Store 应只用于存储应用状态，不应包含 API 请求方法
- 直接使用生成的函数可以获得完整的类型推导
- TanStack Query 自动管理缓存，无需 Store 介入
- 统一的 API 调用规范，避免混乱

### 原则 2：Query 用于读取，Mutation 用于写入

严格区分查询（GET）和变更（POST/PUT/DELETE）操作。

```typescript
// ✅ 正确：GET 请求使用 useQuery
const { data, isLoading } = useQuery(getUserInfoOptions())

// ✅ 正确：POST/PUT/DELETE 使用 useMutation
const { mutate, isPending } = useMutation(postUserSigninMutation())
// mutate({ body: data })

// ❌ 错误：不要用 useMutation 包装 GET 请求
const { mutate } = useMutation({
  mutationFn: () => getUserInfo()  // 这是 GET 请求，应该用 useQuery
})
```

### 原则 3：使用 queryKey 管理缓存

TanStack Query 通过 `queryKey` 识别和管理缓存，正确使用 queryKey 是关键。

```typescript
// ✅ 正确：heyapi 自动生成的 options 已包含正确的 queryKey
const { data } = useQuery(getUserInfoOptions())
const { data } = useQuery(getUserDetailOptions({ path: { id: userId } }))

// ✅ 正确：mutation 后使缓存失效
const queryClient = useQueryClient()
const { mutate } = useMutation({
  ...postUserInfoUpdateMutation(),
  onSuccess: () => {
    // 使用生成的 queryKey 让缓存精确失效
    queryClient.invalidateQueries({ queryKey: getUserInfoQueryKey() })
  }
})

// ❌ 错误：手动构造不一致的 queryKey
const { data } = useQuery({
  queryKey: ['my-user-info'],  // 与生成的 key 不一致，缓存无法共享
  queryFn: () => getUserInfo()
})
```

### 原则 4：业务逻辑在回调中处理

`useQuery` 和 `useMutation` 的核心参数只负责请求，业务逻辑在回调中处理。

```typescript
// ✅ 正确：业务逻辑在回调中
const { mutate } = useMutation({
  ...postUserInfoUpdateMutation(),
  onSuccess: (result) => {
    message.success('更新成功')
    queryClient.invalidateQueries({ queryKey: getUserInfoQueryKey() })
    router.push('/dashboard')
  },
  onError: (error) => {
    message.error(error.message || '更新失败')
  }
})

const handleSubmit = () => {
  // 验证放在调用前
  if (!form.value.name) {
    message.error('名称不能为空')
    return
  }
  mutate(form.value)
}

// ❌ 错误：在 mutationFn 中处理业务逻辑
const { mutate } = useMutation({
  mutationFn: async (data) => {
    if (!data.name) throw new Error('名称不能为空')  // ❌
    const result = await postUserInfoUpdate({ body: data })
    message.success('更新成功')  // ❌
    router.push('/dashboard')   // ❌
    return result
  }
})
```

---

## ✅ 实际页面示例（HomeView）

以下示例来自 `src/views/HomeView.vue`，展示了查询 + 手动刷新缓存的真实用法：

```vue
<script setup lang="ts">
import { computed } from 'vue'
import { useQuery, useQueryClient } from '@tanstack/vue-query'
import { getHealthOptions, getHealthQueryKey } from '@/api'

const queryClient = useQueryClient()
const { data, isLoading, isError, error, isFetching } = useQuery(getHealthOptions())

const message = computed(() => data.value?.message ?? '')

const refresh = () => {
  queryClient.invalidateQueries({ queryKey: getHealthQueryKey() })
}
</script>
```

---

## 核心 Hooks 概览

### useQuery
用于 GET 请求，自动管理加载状态、缓存、后台刷新、错误重试。

### useMutation
用于 POST/PUT/DELETE 请求，提供 `mutate` 函数手动触发，支持乐观更新。

### useQueryClient
获取 QueryClient 实例，用于手动操作缓存（失效、预取、更新）。

### useInfiniteQuery
用于分页/无限滚动场景，自动管理分页状态。

---

## 1. useQuery - 数据查询

### 基础用法

```typescript
import { useQuery } from '@tanstack/vue-query'
import { getUserInfoOptions } from '@/api'

// 最简单的用法 - heyapi 生成的 options 包含一切
const { data, isLoading, error, refetch } = useQuery(getUserInfoOptions())
```

### 返回值说明

```typescript
const {
  data,           // 响应数据（Ref）
  isLoading,      // 首次加载中
  isFetching,     // 任何请求进行中（包括后台刷新）
  isPending,      // 无数据且加载中
  isError,        // 请求失败
  error,          // 错误对象
  isSuccess,      // 请求成功
  refetch,        // 手动重新请求
  isStale,        // 数据是否过期
} = useQuery(getUserInfoOptions())
```

### 带参数的查询

```typescript
import { getUserDetailOptions } from '@/api'

// 静态参数
const { data } = useQuery(getUserDetailOptions({
  path: { id: '123' }
}))

// 响应式参数 - 使用 computed
const userId = ref('123')
const { data } = useQuery(
  computed(() => getUserDetailOptions({
    path: { id: userId.value }
  }))
)
// userId 变化时自动重新请求
```

### 条件查询

```typescript
const userId = ref<string | null>(null)

const { data } = useQuery({
  ...getUserDetailOptions({ path: { id: userId.value! } }),
  enabled: computed(() => !!userId.value)  // 只有 userId 存在时才请求
})
```

### 查询配置

```typescript
const { data } = useQuery({
  ...getUserListOptions(),

  // 缓存配置
  staleTime: 1000 * 60 * 5,    // 5分钟内数据视为新鲜
  gcTime: 1000 * 60 * 30,      // 30分钟后清理未使用的缓存

  // 刷新配置
  refetchOnWindowFocus: true,   // 窗口聚焦时刷新
  refetchOnMount: true,         // 组件挂载时刷新（如果数据过期）
  refetchOnReconnect: true,     // 网络恢复时刷新
  refetchInterval: 1000 * 30,   // 每30秒自动刷新

  // 重试配置
  retry: 3,                     // 失败后重试3次
  retryDelay: (attemptIndex) => Math.min(1000 * 2 ** attemptIndex, 30000),

  // 数据转换
  select: (data) => data.items.filter(item => item.active),
})
```

### 初始数据 / 占位数据

```typescript
// 使用缓存中的数据作为初始值
const { data } = useQuery({
  ...getUserDetailOptions({ path: { id } }),
  initialData: () => {
    // 从列表缓存中查找
    const users = queryClient.getQueryData<User[]>(getUserListQueryKey())
    return users?.find(u => u.id === id)
  },
  initialDataUpdatedAt: () => {
    // 返回初始数据的更新时间，用于判断是否需要后台刷新
    return queryClient.getQueryState(getUserListQueryKey())?.dataUpdatedAt
  }
})

// 占位数据（不会触发后台刷新判断）
const { data } = useQuery({
  ...getUserDetailOptions({ path: { id } }),
  placeholderData: { id, name: '加载中...', email: '' }
})
```

---

## 2. useMutation - 数据变更

### 基础用法

```typescript
import { useMutation, useQueryClient } from '@tanstack/vue-query'
import { getUserInfoQueryKey, postUserInfoUpdateMutation } from '@/api'

const queryClient = useQueryClient()

const { mutate, isPending, error } = useMutation({
  ...postUserInfoUpdateMutation(),
  onSuccess: () => {
    message.success('更新成功')
    queryClient.invalidateQueries({ queryKey: getUserInfoQueryKey() })
  },
  onError: (error) => {
    message.error(error.message)
  }
})

// 调用
const handleSave = () => {
  mutate({ body: formData.value })
}
```

### 返回值说明

```typescript
const {
  mutate,         // 触发变更（不返回 Promise）
  mutateAsync,    // 触发变更（返回 Promise，可 await）
  isPending,      // 请求进行中
  isError,        // 请求失败
  error,          // 错误对象
  isSuccess,      // 请求成功
  data,           // 响应数据
  reset,          // 重置状态
} = useMutation({ ... })
```

### mutate vs mutateAsync

```typescript
// mutate - 回调方式处理结果
mutate(data, {
  onSuccess: (result) => { ... },
  onError: (error) => { ... }
})

// mutateAsync - Promise 方式处理结果
const handleSubmit = async () => {
  try {
    const result = await mutateAsync(data)
    // 处理成功
  } catch (error) {
    // 处理失败
  }
}
```

### 乐观更新

```typescript
const { mutate } = useMutation({
  mutationFn: (newTodo) => todoUpdate({ body: newTodo }),

  onMutate: async (newTodo) => {
    // 取消正在进行的查询
    await queryClient.cancelQueries({ queryKey: ['todos'] })

    // 保存当前数据快照
    const previousTodos = queryClient.getQueryData(['todos'])

    // 乐观更新缓存
    queryClient.setQueryData(['todos'], (old) =>
      old.map(todo => todo.id === newTodo.id ? newTodo : todo)
    )

    // 返回快照用于回滚
    return { previousTodos }
  },

  onError: (err, newTodo, context) => {
    // 发生错误时回滚
    queryClient.setQueryData(['todos'], context.previousTodos)
  },

  onSettled: () => {
    // 无论成功失败，都重新获取最新数据
    queryClient.invalidateQueries({ queryKey: ['todos'] })
  }
})
```

---

## 3. 缓存管理

### 使缓存失效

```typescript
const queryClient = useQueryClient()

// 使单个查询失效
queryClient.invalidateQueries({ queryKey: getUserInfoQueryKey() })

// 使列表相关缓存失效
queryClient.invalidateQueries({ queryKey: getUserListQueryKey() })

// 使所有查询失效
queryClient.invalidateQueries()

// 精确匹配
queryClient.invalidateQueries({
  queryKey: getUserListQueryKey({ query: { page: 1, pageSize: 20 } }),
  exact: true
})
```

### 手动更新缓存

```typescript
// 直接设置缓存数据
queryClient.setQueryData(getUserInfoQueryKey(), newUserData)

// 基于旧数据更新
queryClient.setQueryData(getUserListQueryKey(), (oldData) => {
  return oldData?.map(user =>
    user.id === updatedUser.id ? updatedUser : user
  )
})
```

### 预取数据

```typescript
// 预取（用于悬停预加载等场景）
queryClient.prefetchQuery(getUserInfoOptions())

// 确保数据存在（如果缓存中有新鲜数据则不请求）
queryClient.ensureQueryData(getUserListOptions())
```

### 获取缓存数据

```typescript
// 获取缓存的数据
const userData = queryClient.getQueryData(getUserInfoQueryKey())

// 获取查询状态
const queryState = queryClient.getQueryState(getUserInfoQueryKey())
console.log(queryState?.dataUpdatedAt)  // 最后更新时间
console.log(queryState?.status)         // 'pending' | 'error' | 'success'
```

---

## 4. 组合使用模式

### 模式 1：列表 + 详情联动

```typescript
// 列表页
const { data: userList } = useQuery(getUserListOptions())

// 详情页 - 从列表缓存初始化
const { data: userDetail } = useQuery({
  ...getUserDetailOptions({ path: { id } }),
  initialData: () => {
    return queryClient.getQueryData<User[]>(getUserListQueryKey())
      ?.find(user => user.id === id)
  }
})

// 更新后使两者都刷新
const { mutate } = useMutation({
  ...postUserInfoUpdateMutation(),
  onSuccess: () => {
    queryClient.invalidateQueries({ queryKey: getUserListQueryKey() })
  }
})
```

### 模式 2：级联选择器

```typescript
const orgId = ref<string>('')
const deptId = ref<string>('')

// 组织列表 - 始终加载
const { data: orgList } = useQuery(getOrgListOptions())

// 部门列表 - 依赖组织
const { data: deptList } = useQuery({
  ...getDeptListOptions({ query: { orgId: orgId.value } }),
  enabled: computed(() => !!orgId.value)
})

// 用户列表 - 依赖部门
const { data: userList } = useQuery({
  ...getUserListOptions({ query: { deptId: deptId.value } }),
  enabled: computed(() => !!deptId.value)
})

// 组织变化时清空下级选择
watch(orgId, () => {
  deptId.value = ''
})
watch(deptId, () => {
  // 用户选择自动清空（因为 enabled 变为 false）
})
```

### 模式 3：搜索防抖

```typescript
import { refDebounced } from '@vueuse/core'

const keyword = ref('')
const debouncedKeyword = refDebounced(keyword, 300)

const { data, isFetching } = useQuery({
  ...getUserSearchOptions({ query: { keyword: debouncedKeyword.value } }),
  enabled: computed(() => debouncedKeyword.value.length >= 2)
})
```

### 模式 4：分页列表

```typescript
const page = ref(1)
const pageSize = ref(20)

const { data, isFetching, isPlaceholderData } = useQuery({
  ...getUserListOptions({
    query: {
      page: page.value,
      pageSize: pageSize.value
    }
  }),
  placeholderData: keepPreviousData,  // 切换页面时保持旧数据
})

// 预取下一页
watchEffect(() => {
  if (!isPlaceholderData.value && data.value?.hasMore) {
    queryClient.prefetchQuery(getUserListOptions({
      query: {
        page: page.value + 1,
        pageSize: pageSize.value
      }
    }))
  }
})
```

### 模式 5：无限滚动

```typescript
import { useInfiniteQuery } from '@tanstack/vue-query'

const {
  data,
  fetchNextPage,
  hasNextPage,
  isFetchingNextPage
} = useInfiniteQuery({
  queryKey: ['userList', 'infinite'],
  queryFn: ({ pageParam = 1 }) => userList({
    query: { page: pageParam, pageSize: 20 }
  }),
  getNextPageParam: (lastPage, pages) => {
    return lastPage.hasMore ? pages.length + 1 : undefined
  },
  initialPageParam: 1
})

// 所有数据展平
const allUsers = computed(() =>
  data.value?.pages.flatMap(page => page.items) ?? []
)
```

---

## 5. Loading 状态管理

### 区分不同的加载状态

```typescript
const { isLoading, isFetching, isPending } = useQuery(...)

// isLoading: 首次加载（无缓存数据时）
// isFetching: 任何请求进行中（包括后台刷新）
// isPending: 无数据且正在加载
```

```vue
<template>
  <!-- 首次加载显示骨架屏 -->
  <n-skeleton v-if="isLoading" />

  <!-- 有数据时显示内容，右上角显示刷新指示器 -->
  <div v-else>
    <n-spin :show="isFetching" size="small" class="float-right" />
    <UserList :data="data" />
  </div>
</template>
```

### Mutation 加载状态

```typescript
const { mutate, isPending } = useMutation({ ... })

<n-button :loading="isPending" @click="handleSubmit">
  保存
</n-button>
```

### 合并多个加载状态

```typescript
const { isFetching: fetchingUser } = useQuery(getUserInfoOptions())
const { isFetching: fetchingOrg } = useQuery(getOrgListOptions())
const { isPending: saving } = useMutation({ ... })

const isLoading = computed(() =>
  fetchingUser.value || fetchingOrg.value || saving.value
)
```

---

## 6. 错误处理

### 查询错误处理

```typescript
const { data, error, isError } = useQuery({
  ...getUserInfoOptions(),
  retry: 2,  // 失败后重试2次
  retryDelay: 1000,  // 重试间隔
})

// 模板中处理错误
<template>
  <div v-if="isError" class="error">
    {{ error.message }}
    <n-button @click="refetch">重试</n-button>
  </div>
</template>
```

### 全局错误处理

```typescript
// main.ts
app.use(VueQueryPlugin, {
  queryClientConfig: {
    defaultOptions: {
      queries: {
        retry: 1,
        onError: (error) => {
          // 全局查询错误处理
          console.error('Query error:', error)
        }
      },
      mutations: {
        onError: (error) => {
          // 全局 mutation 错误处理
          message.error(error.message || '操作失败')
        }
      }
    }
  }
})
```

### Mutation 错误处理

```typescript
const { mutate } = useMutation({
  ...postUserInfoUpdateMutation(),
  onError: (error, variables, context) => {
    // error: 错误对象
    // variables: 传入的参数
    // context: onMutate 返回的上下文
    message.error(error.message)
  }
})

// 或者使用 mutateAsync
try {
  await mutateAsync(data)
} catch (error) {
  message.error(error.message)
}
```

---

## 7. 与 Alova 的对比

| 特性 | Alova (useReq) | TanStack Query |
|------|----------------|----------------|
| **查询** | `useReq(() => Apis.user.info())` | `useQuery(getUserInfoOptions())` |
| **变更** | `useReq(() => Apis.user.update({ data }))` | `useMutation(postUserInfoUpdateMutation())` |
| **加载状态** | `loading` | `isLoading`, `isFetching`, `isPending` |
| **强制刷新** | `send(FORCE)` | `refetch()` 或 `invalidateQueries()` |
| **缓存时间** | `cacheFor: 5000` | `staleTime: 5000` |
| **缓存失效** | 手动管理 | `invalidateQueries()` |
| **响应式参数** | 通过 `send(params)` | 通过 `computed` 包装 options |
| **错误处理** | `skipShowError` + try/catch | `onError` 回调 |
| **后台刷新** | 需手动实现 | 自动（窗口聚焦、网络恢复等） |
| **DevTools** | 无 | 有 |

### 迁移示例

**Alova 方式：**
```typescript
const userList = ref([])
const { loading, send } = useReq(
  Apis.user.list
).onDataRefresh((data) => {
  userList.value = data.value?.items || []
})

useInit(() => send())
```

**TanStack Query 方式：**
```typescript
const { data, isLoading } = useQuery(getUserListOptions())
const userList = computed(() => data.value?.items ?? [])
// 无需 useInit，自动加载
```

---

## 8. 最佳实践清单

### 🚨 核心原则（必须遵守）

- ✅ **使用生成的 API**：使用 heyapi 生成的函数和 options
- ✅ **Query vs Mutation**：GET 用 useQuery，POST/PUT/DELETE 用 useMutation
- ✅ **正确使用 queryKey**：使用生成的 options，确保缓存一致性
- ✅ **业务逻辑在回调中**：验证、提示、副作用在 onSuccess/onError 中处理
- ✅ **使用返回的状态**：使用 isLoading/isPending，不手动管理

### 缓存策略

- ✅ **静态数据**：`staleTime: Infinity`（字典、配置等）
- ✅ **准静态数据**：`staleTime: 1000 * 60 * 5`（5分钟，用户信息等）
- ✅ **动态数据**：`staleTime: 0`（默认，列表等）
- ✅ **gcTime >= staleTime**：确保过期数据在清理前仍可用

### 代码组织

- ✅ 将查询定义放在组件顶部
- ✅ 使用 `computed` 包装响应式参数
- ✅ Mutation 成功后使用 `invalidateQueries` 刷新相关数据
- ✅ 使用 `select` 在查询层面转换数据

### 性能优化

- ✅ 使用 `enabled` 条件控制请求时机
- ✅ 使用 `placeholderData: keepPreviousData` 优化分页体验
- ✅ 使用 `prefetchQuery` 预加载数据
- ✅ 搜索场景使用 `refDebounced` 防抖

### 错误处理

- ✅ 设置合理的 `retry` 次数
- ✅ Mutation 使用 `onError` 回调
- ✅ 关键操作使用 `mutateAsync` + try/catch

---

## 9. 扩展阅读

- [TanStack Query Vue 文档](https://tanstack.com/query/latest/docs/vue/overview)
- [Hey API 文档](https://heyapi.dev/)
- [TanStack Query DevTools](https://tanstack.com/query/latest/docs/vue/devtools)
- [项目 API 配置](../../src/api/client.ts)

