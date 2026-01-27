# golang单测规范
不要侵入代码改造，使用"github.com/bytedance/mockey"去mock下游方法的调用

# 日志等级规则
- 异常来自系统内部，例如数据库异常、服务逻辑异常，使用Errorf
- 异常来自系统外部，例如请求参数错误、请求超时，使用Noticef
- 其他情况，使用Infof

# 任务执行完后优化优化文件排版
```
gofmt -w .
```