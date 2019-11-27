package conf

import "xgit.xiaoniangao.cn/xngo/lib/xlog"

var (
	Logger      *xlog.XLogger
	TraceLogger *xlog.XLogger
)

func InitLogger() {
	if C.LogConfig == nil {
		panic("logConfig is required")
	}

	if C.TraceLogConfig == nil {
		panic("TraceLogConfig is required")
	}

	Logger = xlog.NewLogger(C.LogConfig)
	TraceLogger = xlog.NewLogger(C.TraceLogConfig)

}
