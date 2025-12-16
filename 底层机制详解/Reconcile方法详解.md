# Reconcile 方法详细解析

## 1. 方法签名

```go
func (r *ApiserviceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)
```

### 方法结构图

```
┌─────────────────────────────────────────────────────────────┐
│                    Reconcile 方法                            │
├─────────────────────────────────────────────────────────────┤
│ 接收者: *ApiserviceReconciler                                │
│ 参数1:  ctx context.Context                                  │
│ 参数2:  req ctrl.Request                                     │
│ 返回值: (ctrl.Result, error)                                 │
└─────────────────────────────────────────────────────────────┘
```

---

## 2. 接收者 (Receiver): `*ApiserviceReconciler`

### 2.1 结构体定义

```go
type ApiserviceReconciler struct {
    client.Client        // 嵌入的 Kubernetes 客户端接口
    Scheme *runtime.Scheme  // Kubernetes 资源类型注册表
}
```

### 2.2 结构体属性详解

#### 属性1: `client.Client` (嵌入字段)
- **类型**: `client.Client` 接口
- **作用**: 提供与 Kubernetes API Server 交互的能力
- **来源**: `sigs.k8s.io/controller-runtime/pkg/client`
- **初始化**: 在 `main.go` 中通过 `mgr.GetClient()` 获取
  ```go
  // main.go 第181-184行
  &controller.ApiserviceReconciler{
      Client: mgr.GetClient(),  // ← 从这里获取
      Scheme: mgr.GetScheme(),
  }
  ```
- **常用方法**:
  - `Get(ctx, key, obj)` - 获取单个资源对象
    ```go
    // 示例: 第65行
    r.Get(ctx, req.NamespacedName, apiservice)
    ```
  - `List(ctx, list, opts...)` - 列出多个资源对象
    ```go
    // 示例用法（代码中未使用）
    var deployments appsv1.DeploymentList
    r.List(ctx, &deployments, client.InNamespace("default"))
    ```
  - `Create(ctx, obj, opts...)` - 创建资源对象
    ```go
    // 示例: 第85行
    r.Create(ctx, dep)
    ```
  - `Update(ctx, obj, opts...)` - 更新资源对象
    ```go
    // 示例: 第100行
    r.Update(ctx, deployment)
    ```
  - `Patch(ctx, obj, patch, opts...)` - 部分更新资源对象
    ```go
    // 示例用法（代码中未使用）
    patch := client.MergeFrom(original)
    r.Patch(ctx, obj, patch)
    ```
  - `Delete(ctx, obj, opts...)` - 删除资源对象
    ```go
    // 示例用法（代码中未使用）
    r.Delete(ctx, deployment)
    ```
  - `Status()` - 返回用于更新状态的子客户端
    ```go
    // 示例: 第248行
    r.Status().Update(ctx, latestApiservice)
    // Status() 返回 client.StatusWriter，专门用于更新资源的 status 子资源
    ```

#### 属性2: `Scheme *runtime.Scheme`
- **类型**: `*runtime.Scheme` 指针
- **作用**: Kubernetes 资源类型的注册表，用于：
  - 类型转换和序列化/反序列化
  - Owner Reference 的设置（通过 `controllerutil.SetControllerReference`）
  - 资源版本的识别和管理
- **初始化**: 在 `main.go` 中通过 `mgr.GetScheme()` 获取
  ```go
  // main.go 第181-184行
  &controller.ApiserviceReconciler{
      Client: mgr.GetClient(),
      Scheme: mgr.GetScheme(),  // ← 从这里获取
  }
  ```
- **Scheme 的创建**: 在 `main.go` 的 `init()` 函数中初始化
  ```go
  // main.go 第44行和第48-53行
  var scheme = runtime.NewScheme()
  
  func init() {
      utilruntime.Must(clientgoscheme.AddToScheme(scheme))  // 添加标准 K8s 资源
      utilruntime.Must(myservicev1.AddToScheme(scheme))   // 添加自定义资源
  }
  ```

### 2.3 接收者的方法

从代码中可以看到 `ApiserviceReconciler` 有以下方法：

1. **`Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error)`**
   - 核心协调方法，实现 `reconcile.Reconciler` 接口

2. **`deploymentForApiservice(apiservice *myservicev1.Apiservice) *appsv1.Deployment`**
   - 根据 Apiservice CR 创建 Deployment 对象

3. **`serviceForApiservice(apiservice *myservicev1.Apiservice) *corev1.Service`**
   - 根据 Apiservice CR 创建 Service 对象

4. **`updateStatus(ctx context.Context, apiservice *myservicev1.Apiservice, deployment *appsv1.Deployment) error`**
   - 更新 Apiservice CR 的状态

5. **`SetupWithManager(mgr ctrl.Manager) error`**
   - 将 Reconciler 注册到 Controller Manager

### 2.4 接收者关系图

```
┌─────────────────────────────────────┐
│   ApiserviceReconciler              │
├─────────────────────────────────────┤
│  + client.Client (嵌入接口)          │
│    ├─ Get()                         │
│    ├─ Create()                      │
│    ├─ Update()                      │
│    ├─ Delete()                      │
│    └─ Status()                      │
│                                     │
│  + Scheme *runtime.Scheme           │
│    └─ 资源类型注册表                 │
├─────────────────────────────────────┤
│  方法:                               │
│  + Reconcile()                      │
│  + deploymentForApiservice()        │
│  + serviceForApiservice()           │
│  + updateStatus()                   │
│  + SetupWithManager()               │
└─────────────────────────────────────┘
```

---

## 3. 参数1: `ctx context.Context`

### 3.1 类型说明
- **类型**: `context.Context`
- **来源**: `context` 标准库
- **作用**: 传递请求上下文信息，包括：
  - 请求的取消信号
  - 请求的超时时间
  - 请求的追踪信息
  - 日志记录器（通过 `logf.FromContext(ctx)` 获取）

### 3.2 Context 的来源

`ctx` 是由 **controller-runtime** 框架自动传入的，调用链如下：

```
Kubernetes API Server
    ↓ (资源变更事件)
Controller Manager
    ↓ (创建 context)
Controller
    ↓ (调用 Reconcile)
Reconcile(ctx, req)  ← ctx 在这里传入
```

### 3.3 Context 中包含的内容

在代码中可以看到 `ctx` 的使用：

```go
_ = logf.FromContext(ctx)  // 从 context 中提取 logger
logf.FromContext(ctx).Info("...")  // 使用 logger 记录信息
logf.FromContext(ctx).Error(err, "...")  // 记录错误
```

**Context 中通常包含**:
- **Logger**: 通过 `logf.FromContext(ctx)` 获取
- **Request ID**: 用于追踪请求
- **Timeout/Cancel**: 请求取消信号
- **Trace Context**: 分布式追踪信息

### 3.4 Context 传递图

```
┌─────────────────────────────────────────┐
│  Controller Runtime Framework           │
│                                         │
│  1. 监听 Kubernetes 资源变更            │
│  2. 创建 context.Context                │
│     ├─ 注入 logger                      │
│     ├─ 设置 timeout                     │
│     └─ 添加 trace context               │
│  3. 创建 ctrl.Request                   │
│  4. 调用 Reconcile(ctx, req)            │
└─────────────────────────────────────────┘
              ↓
┌─────────────────────────────────────────┐
│  Reconcile(ctx, req)                    │
│                                         │
│  ctx: context.Context                   │
│  ├─ logger (logf.FromContext)          │
│  ├─ timeout/cancel                      │
│  └─ trace info                          │
└─────────────────────────────────────────┘
```

---

## 4. 参数2: `req ctrl.Request`

### 4.1 类型说明
- **类型**: `ctrl.Request`
- **来源**: `sigs.k8s.io/controller-runtime`
- **作用**: 包含需要协调的资源对象标识信息

### 4.2 Request 结构体定义

根据 controller-runtime 的源码，`ctrl.Request` 的结构如下：

```go
type Request struct {
    NamespacedName types.NamespacedName
}

// types.NamespacedName 定义
type NamespacedName struct {
    Namespace string
    Name      string
}
```

### 4.3 Request 字段详解

#### `req.NamespacedName`
- **类型**: `types.NamespacedName`
- **作用**: 唯一标识一个 Kubernetes 资源对象
- **字段**:
  - `Namespace string`: 资源所在的命名空间
  - `Name string`: 资源的名称

### 4.4 Request 使用示例

在代码中的使用：

```go
// 第65行：使用 req.NamespacedName 获取 Apiservice 对象
if err := r.Get(ctx, req.NamespacedName, apiservice); err != nil {
    // ...
}
```

等价于：

```go
r.Get(ctx, types.NamespacedName{
    Namespace: req.NamespacedName.Namespace,
    Name:      req.NamespacedName.Name,
}, apiservice)
```

### 4.5 Request 结构图

```
┌─────────────────────────────────────┐
│      ctrl.Request                    │
├─────────────────────────────────────┤
│  NamespacedName: types.NamespacedName│
│    ├─ Namespace: string              │
│    │  例如: "default"                │
│    │                                 │
│    └─ Name: string                   │
│       例如: "my-apiservice"          │
└─────────────────────────────────────┘

示例值:
req.NamespacedName.Namespace = "default"
req.NamespacedName.Name = "my-apiservice"

实际使用示例:
r.Get(ctx, req.NamespacedName, apiservice)
等价于:
r.Get(ctx, types.NamespacedName{
    Namespace: "default",
    Name:      "my-apiservice",
}, apiservice)
```

### 4.6 Request 的来源

`req` 是由 controller-runtime 根据监听到的资源变更事件自动创建的：

```
资源变更事件 (Watch Event)
    ↓
Controller 识别到需要协调的资源
    ↓
创建 ctrl.Request{
    NamespacedName: {
        Namespace: "default",
        Name: "my-apiservice"
    }
}
    ↓
调用 Reconcile(ctx, req)
```

---

## 5. 返回值: `(ctrl.Result, error)`

### 5.1 返回值类型

#### `ctrl.Result`
- **类型**: `ctrl.Result` 结构体
- **来源**: `sigs.k8s.io/controller-runtime/pkg/reconcile`
- **作用**: 控制协调循环的行为，决定是否以及何时重新执行 Reconcile
- **字段**:
  ```go
  type Result struct {
      Requeue      bool          // 是否立即重新排队（优先级高于 RequeueAfter）
      RequeueAfter time.Duration // 延迟多久后重新排队（例如: 10*time.Second）
  }
  ```
- **字段说明**:
  - `Requeue bool`: 
    - `true`: 立即将请求重新加入队列，即使 `RequeueAfter` 有值也会立即执行
    - `false`: 不立即重新排队，但如果 `RequeueAfter > 0` 则延迟排队
  - `RequeueAfter time.Duration`:
    - `> 0`: 延迟指定时间后重新排队
    - `= 0`: 不延迟（如果 `Requeue = false`，则不再协调）
- **常用返回值模式**:
  ```go
  // 1. 成功完成，不需要重新协调
  return ctrl.Result{}, nil
  
  // 2. 立即重新协调（例如刚创建了资源，需要等待其就绪）
  return ctrl.Result{Requeue: true}, nil
  
  // 3. 延迟重新协调（例如等待资源就绪）
  return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
  
  // 4. 发生错误，框架会自动重新协调
  return ctrl.Result{}, err
  ```

#### `error`
- **类型**: `error` 接口
- **作用**: 表示协调过程中的错误

### 5.2 返回值的使用场景

代码中的实际返回值示例（来自 apiservice_controller.go）：

```go
// 第68行: Apiservice 资源不存在（已被删除）
return ctrl.Result{}, nil
// 说明: 资源已删除，不需要进一步协调

// 第71行: 获取 Apiservice 时出错
return ctrl.Result{}, err
// 说明: 发生错误，框架会自动重新协调

// 第90行: 刚创建了 Deployment，需要等待其就绪
return ctrl.Result{Requeue: true}, nil
// 说明: 立即重新协调，检查 Deployment 状态

// 第93行: 获取 Deployment 时出错
return ctrl.Result{}, err
// 说明: 发生错误，框架会自动重新协调

// 第102行: 更新 Deployment 时出错
return ctrl.Result{}, err
// 说明: 发生错误，框架会自动重新协调

// 第119行: 创建 Service 时出错
return ctrl.Result{}, err
// 说明: 发生错误，框架会自动重新协调

// 第123行: 获取 Service 时出错
return ctrl.Result{}, err
// 说明: 发生错误，框架会自动重新协调

// 第129行: 更新状态时出错
return ctrl.Result{}, err
// 说明: 发生错误，框架会自动重新协调

// 第131行: 所有操作成功完成
return ctrl.Result{}, nil
// 说明: 协调完成，不需要重新协调
```

**返回值决策逻辑**:
- ✅ **成功且资源已就绪**: `return ctrl.Result{}, nil`
- ⏩ **成功但需要等待**: `return ctrl.Result{Requeue: true}, nil`
- ⏰ **成功但需要延迟**: `return ctrl.Result{RequeueAfter: 10*time.Second}, nil`
- ❌ **发生错误**: `return ctrl.Result{}, err` (框架会自动重新协调)

### 5.3 返回值决策流程图

```
┌─────────────────────────────────────┐
│  Reconcile 执行完成                  │
└─────────────────────────────────────┘
              ↓
    ┌─────────┴─────────┐
    │                    │
  有错误?              无错误
    │                    │
    ↓                    ↓
返回错误           需要重新协调?
return (Result{}, err)    │
                    ┌─────┴─────┐
                    │           │
                  是           否
                    │           │
                    ↓           ↓
            Requeue: true   Requeue: false
            return (Result{Requeue: true}, nil)
            return (Result{}, nil)
```

---

## 6. Reconcile 方法执行流程

### 6.1 完整流程图

```
开始 Reconcile
    ↓
1. 从 context 获取 logger
    ↓
2. 使用 req.NamespacedName 获取 Apiservice CR
    ├─ 如果不存在 → 返回 (Result{}, nil)
    └─ 如果出错 → 返回 (Result{}, err)
    ↓
3. 检查 Deployment 是否存在
    ├─ 不存在 → 创建 Deployment → 返回 (Result{Requeue: true}, nil)
    ├─ 存在但出错 → 返回 (Result{}, err)
    └─ 存在 → 继续
    ↓
4. 确保 Deployment 副本数匹配
    ├─ 不匹配 → 更新 Deployment
    └─ 匹配 → 继续
    ↓
5. 检查 Service 是否存在
    ├─ 不存在 → 创建 Service
    └─ 存在 → 继续
    ↓
6. 更新 Apiservice CR 的状态
    ↓
7. 返回 (Result{}, nil)
```

### 6.2 方法调用关系图

```
Reconcile(ctx, req)
    │
    ├─→ r.Get(ctx, req.NamespacedName, apiservice)
    │      使用: ctx, req.NamespacedName
    │      作用: 获取 Apiservice CR 对象
    │
    ├─→ r.Get(ctx, types.NamespacedName{...}, deployment)
    │      使用: ctx, apiservice.Name, apiservice.Namespace
    │      作用: 检查 Deployment 是否存在
    │
    ├─→ r.deploymentForApiservice(apiservice)
    │      使用: apiservice, r.Scheme
    │      作用: 根据 CR 创建 Deployment 对象
    │      返回: *appsv1.Deployment
    │
    ├─→ r.Create(ctx, dep)
    │      使用: ctx, deployment对象
    │      作用: 在 K8s 中创建 Deployment
    │
    ├─→ r.Update(ctx, deployment)
    │      使用: ctx, deployment对象
    │      作用: 更新 Deployment 的副本数
    │
    ├─→ r.serviceForApiservice(apiservice)
    │      使用: apiservice, r.Scheme
    │      作用: 根据 CR 创建 Service 对象
    │      返回: *corev1.Service
    │
    ├─→ r.Create(ctx, svc)
    │      使用: ctx, service对象
    │      作用: 在 K8s 中创建 Service
    │
    └─→ r.updateStatus(ctx, apiservice, deployment)
            ├─→ r.Get(ctx, ...) 获取最新状态
            │    作用: 获取最新的 Apiservice CR
            └─→ r.Status().Update(ctx, latestApiservice)
                  作用: 更新 CR 的状态字段
```

### 6.3 数据流向图

```
输入:
  ctx: context.Context
  req: ctrl.Request {Namespace: "default", Name: "my-api"}
    ↓
步骤1: 获取 Apiservice CR
  r.Get(ctx, req.NamespacedName, apiservice)
    ↓
  apiservice: *myservicev1.Apiservice {
    Name: "my-api",
    Namespace: "default",
    Spec: {Replicas: 3, Image: "nginx", ...}
  }
    ↓
步骤2: 检查/创建 Deployment
  r.Get(ctx, ...) → 不存在
  r.deploymentForApiservice(apiservice) → dep
  r.Create(ctx, dep)
    ↓
步骤3: 更新 Deployment 副本数
  r.Get(ctx, ...) → deployment
  比较副本数 → 不一致
  r.Update(ctx, deployment)
    ↓
步骤4: 检查/创建 Service
  r.Get(ctx, ...) → 不存在
  r.serviceForApiservice(apiservice) → svc
  r.Create(ctx, svc)
    ↓
步骤5: 更新状态
  r.updateStatus(ctx, apiservice, deployment)
    ├─ r.Get(ctx, ...) → latestApiservice
    └─ r.Status().Update(ctx, latestApiservice)
    ↓
输出:
  return ctrl.Result{}, nil
```

---

## 7. 关键概念总结

### 7.1 接收者 `*ApiserviceReconciler`
- **作用**: 协调 Apiservice CR 的实际状态与期望状态
- **属性**: 
  - `client.Client`: 提供 Kubernetes API 操作能力
  - `Scheme`: 资源类型注册表
- **方法**: Reconcile, deploymentForApiservice, serviceForApiservice, updateStatus, SetupWithManager

### 7.2 参数 `ctx context.Context`
- **来源**: controller-runtime 框架自动创建并传入
- **内容**: logger, timeout, trace context
- **用途**: 传递上下文信息，支持取消、超时、日志记录

### 7.3 参数 `req ctrl.Request`
- **来源**: controller-runtime 根据资源变更事件创建
- **结构**: `{NamespacedName: {Namespace, Name}}`
- **用途**: 标识需要协调的资源对象

### 7.4 返回值 `(ctrl.Result, error)`
- **Result**: 控制是否重新协调及延迟时间
- **error**: 表示协调过程中的错误

---

## 8. 实际调用示例

假设有一个 Apiservice CR 被创建：

```yaml
apiVersion: myservice.cyk.io/v1
kind: Apiservice
metadata:
  name: my-api
  namespace: default
spec:
  replicas: 3
  image: nginx:latest
  port: 8080
```

**调用过程**:

1. **Controller Runtime 检测到资源创建**
   ```go
   // 框架内部创建
   ctx := context.WithTimeout(context.Background(), 10*time.Second)
   req := ctrl.Request{
       NamespacedName: types.NamespacedName{
           Namespace: "default",
           Name: "my-api",
       },
   }
   ```

2. **调用 Reconcile**
   ```go
   result, err := reconciler.Reconcile(ctx, req)
   ```

3. **Reconcile 内部执行**
   - 使用 `req.NamespacedName` 获取 Apiservice CR
   - 检查并创建 Deployment
   - 检查并创建 Service
   - 更新状态

4. **返回结果**
   ```go
   return ctrl.Result{}, nil  // 成功完成
   ```

---

## 9. 接收者的初始化过程

### 9.1 初始化流程图

```
main() 函数启动
    ↓
创建 Manager (mgr)
    ├─ 使用 ctrl.NewManager() 创建
    ├─ 传入 Scheme (包含所有资源类型)
    └─ Manager 内部创建 Client
    ↓
创建 ApiserviceReconciler
    ├─ Client: mgr.GetClient()  ← 从 Manager 获取
    └─ Scheme: mgr.GetScheme()  ← 从 Manager 获取
    ↓
调用 SetupWithManager(mgr)
    └─ 将 Reconciler 注册到 Manager
    ↓
Manager.Start()
    └─ 开始监听资源变更
        └─ 当资源变更时，调用 Reconcile(ctx, req)
```

### 9.2 初始化代码位置

**main.go 第181-184行**:
```go
if err := (&controller.ApiserviceReconciler{
    Client: mgr.GetClient(),    // Manager 提供的客户端
    Scheme: mgr.GetScheme(),     // Manager 提供的 Scheme
}).SetupWithManager(mgr); err != nil {
    setupLog.Error(err, "unable to create controller", "controller", "Apiservice")
    os.Exit(1)
}
```

**关键点**:
- `mgr.GetClient()`: Manager 会创建一个配置好的 Kubernetes 客户端
- `mgr.GetScheme()`: Manager 使用在 `init()` 中注册的 Scheme
- `SetupWithManager(mgr)`: 将 Reconciler 注册到 Manager，建立监听关系

---

## 10. 总结

`Reconcile` 方法是 Kubernetes Operator 的核心，它：

1. **接收者**: `*ApiserviceReconciler` 
   - 在 `main.go` 中初始化，通过 Manager 注入 `Client` 和 `Scheme`
   - 提供了操作 Kubernetes 资源的能力
   - 包含多个辅助方法用于创建和管理资源

2. **参数 ctx**: 
   - 由 controller-runtime 框架自动创建并传入
   - 包含日志记录器、超时控制、追踪信息等上下文

3. **参数 req**: 
   - 由框架根据监听到的资源变更事件自动创建
   - 包含资源的命名空间和名称，用于唯一标识资源

4. **返回值**: 
   - `ctrl.Result`: 控制是否重新协调及延迟时间
   - `error`: 表示协调过程中的错误

整个方法实现了声明式 API 的核心思想：**持续协调实际状态与期望状态，直到两者一致**。

### 10.1 完整调用链

```
Kubernetes API Server
    ↓ (资源变更: Create/Update/Delete)
Controller Manager (mgr)
    ↓ (创建 context 和 request)
Controller
    ↓ (调用 Reconcile)
Reconcile(ctx, req)
    ├─ 使用 ctx 进行日志记录和上下文传递
    ├─ 使用 req.NamespacedName 获取资源
    ├─ 使用 r.Client 操作 Kubernetes 资源
    ├─ 使用 r.Scheme 设置 Owner Reference
    └─ 返回 Result 和 error 控制协调循环
```

### 10.2 完整架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                       │
│                                                             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐    │
│  │ Apiservice CR│  │  Deployment  │  │   Service    │    │
│  │  (期望状态)   │  │  (实际状态)   │  │  (实际状态)   │    │
│  └──────────────┘  └──────────────┘  └──────────────┘    │
└─────────────────────────────────────────────────────────────┘
                    ↕ API 调用
┌─────────────────────────────────────────────────────────────┐
│              Controller Runtime Framework                    │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │         Controller Manager (mgr)                    │  │
│  │  ┌──────────────────────────────────────────────┐   │  │
│  │  │  Watch & Event Handler                      │   │  │
│  │  │  - 监听 Apiservice CR 变更                   │   │  │
│  │  │  - 创建 context.Context                      │   │  │
│  │  │  - 创建 ctrl.Request                         │   │  │
│  │  └──────────────────────────────────────────────┘   │  │
│  │                      ↓                               │  │
│  │  ┌──────────────────────────────────────────────┐   │  │
│  │  │    ApiserviceReconciler                      │   │  │
│  │  │  ┌────────────────────────────────────────┐ │   │  │
│  │  │  │  Client: client.Client                │ │   │  │
│  │  │  │  - Get()                              │ │   │  │
│  │  │  │  - Create()                           │ │   │  │
│  │  │  │  - Update()                           │ │   │  │
│  │  │  │  - Status().Update()                  │ │   │  │
│  │  │  └────────────────────────────────────────┘ │   │  │
│  │  │  ┌────────────────────────────────────────┐ │   │  │
│  │  │  │  Scheme: *runtime.Scheme               │ │   │  │
│  │  │  │  - 资源类型注册                         │ │   │  │
│  │  │  │  - Owner Reference 设置                 │ │   │  │
│  │  │  └────────────────────────────────────────┘ │   │  │
│  │  └──────────────────────────────────────────────┘   │  │
│  └──────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                    ↓ 调用
┌─────────────────────────────────────────────────────────────┐
│              Reconcile(ctx, req) 方法                        │
│                                                             │
│  输入:                                                       │
│    ctx: context.Context                                     │
│      ├─ logger (logf.FromContext)                          │
│      ├─ timeout/cancel                                      │
│      └─ trace context                                       │
│                                                             │
│    req: ctrl.Request                                        │
│      └─ NamespacedName: {Namespace, Name}                  │
│                                                             │
│  执行流程:                                                    │
│    1. 获取 Apiservice CR                                    │
│    2. 检查/创建 Deployment                                  │
│    3. 确保副本数一致                                         │
│    4. 检查/创建 Service                                     │
│    5. 更新 CR 状态                                          │
│                                                             │
│  输出:                                                       │
│    ctrl.Result: {Requeue, RequeueAfter}                     │
│    error: 错误信息                                           │
└─────────────────────────────────────────────────────────────┘
```

### 10.3 关键组件交互图

```
┌─────────────┐
│   main.go   │
└──────┬──────┘
       │ 创建
       ↓
┌──────────────────┐
│  Manager (mgr)   │
│  ┌────────────┐  │
│  │  Client    │──┼──→ 注入到
│  │  Scheme    │  │
│  └────────────┘  │
└──────┬───────────┘
       │ 创建并注册
       ↓
┌──────────────────────────┐
│ ApiserviceReconciler     │
│  Client: mgr.GetClient() │
│  Scheme: mgr.GetScheme() │
└──────┬───────────────────┘
       │ 实现
       ↓
┌──────────────────────────┐
│  Reconcile(ctx, req)     │
│                          │
│  使用 r.Client 操作资源   │
│  使用 r.Scheme 设置引用   │
│  使用 ctx 传递上下文      │
│  使用 req 标识资源        │
└──────────────────────────┘
```
