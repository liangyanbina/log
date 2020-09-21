通用日志模块
import github.com/liangyanbina/common/logs

logs.MustRollingLog("./logs", 5, 100000000, "debug")