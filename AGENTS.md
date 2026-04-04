# golang单测规范
- 不要侵入代码改造，使用"github.com/bytedance/mockey"去mock下游方法的调用。需要使用指令 -gcflags="all=-l -N" 来禁用内联优化。
- 修复单测的时候不要修改任何业务代码或者配置文件。

# 代码规范
- 函数长度不要超过100行，如果超出了，则考虑是否可以拆分成多个函数。
- 不要使用magic value，常量要定义在文件的开头部分
- 当函数入参超过3个的时候（不包括context），将函数入参封装成一个结构体，作为函数的入参。例如：
```go
type MyFuncParam struct {
	A int
	B string
	C bool
}
func MyFunc(ctx context.Context, input MyFuncParam) error {
	// 实现业务逻辑
	return nil
}
```
- biz/handler中处理http响应相关的处理，如入参解析、出参序列化bizErr错误码解析
- biz/service中处理业务逻辑相关的处理，如数据库操作、缓存操作等，不能处理与http相关的逻辑

# 日志等级规则
- 异常来自系统内部，例如数据库异常、服务逻辑异常，使用Errorf
- 异常来自请求，例如请求参数错误、请求超时，使用Noticef
- 其他情况，例如记录链路信息，使用Infof


# 任务顺序
- 生成完代码后先使用go build检查是否有语法错误
- 没有语法错误，再使用go test检查逻辑错误并进行修复
- 最后使用 `gofmt -w .` 优化文件排版

