# useany

A golangci-lint plugin that detects `interface{}` usage and suggests replacing it with `any`.

## 功能

- 🔍 **全面检测**: 检测所有使用场景下的 `interface{}`，包括：
  - 变量声明
  - 函数参数和返回值
  - 结构体字段
  - 复合类型（数组、切片、映射、通道）
  - 类型定义
  - 泛型约束
  
- 🛠️ **自动修复**: 支持 `golangci-lint --fix` 自动将 `interface{}` 替换为 `any`

- 🎯 **精确识别**: 只对空接口 `interface{}` 进行检测，不会误报有方法的接口

## 安装和使用

这个linter使用了 golangci-lint 的新插件系统（v2.1.5+）。

### 方式一：作为模块使用

1. **将此仓库作为依赖添加**：
```bash
go mod edit -require github.com/qor5/linter@latest
go mod tidy
```

2. **创建 .custom-gcl.yml 配置文件**：
```yaml
version: v2.1.5
plugins:
  - module: 'github.com/qor5/linter'
    import: 'github.com/qor5/linter/useany'
  # 可以添加更多插件...
```

3. **构建自定义 golangci-lint**：
```bash
golangci-lint custom
```

4. **配置 .golangci.yml**：
```yaml
linters-settings:
  custom:
    useany:
      description: 'Check for interface{} that can be replaced with any'
      # 暂无额外配置项

linters:
  enable:
    - useany
```

5. **运行检查**：
```bash
# 只检查
./custom-gcl run

# 检查并自动修复
./custom-gcl run --fix
```

### 方式二：本地开发

如果您想修改或扩展此linter：

```bash
git clone https://github.com/qor5/linter.git
cd linter
go test ./useany -v  # 运行测试
```

## 示例

**检测前:**
```go
var data interface{}

func process(input interface{}) interface{} {
    return input
}

type Config struct {
    Settings map[string]interface{}
    Values   []interface{}
}
```

**检测后 (使用 --fix):**
```go
var data any

func process(input any) any {
    return input
}

type Config struct {
    Settings map[string]any
    Values   []any
}
```

## 为什么要用 any？

- `any` 是 Go 1.18 引入的 `interface{}` 的类型别名
- 更简洁、易读
- 表达意图更明确
- 符合现代 Go 代码风格

## 不会误报的情况

```go
// 这些有方法的接口不会被检测
type Reader interface {
    Read([]byte) (int, error)
}

type Writer interface {
    Write([]byte) (int, error)
}
``` 
