//gen

package dist

import (
	"github.com/gin-gonic/gin"
	"github.com/keroro6/gen/api"
	"github.com/keroro6/gen/conf"
	"github.com/keroro6/gen/service/dist"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/lib"
	"xgit.xiaoniangao.cn/xngo/lib/sdk/xng"
)

func (req *api.TrafficDistReq)checkParam() (ok bool) {
	return
}

func TrafficDist(c *gin.Context) {
	xc := xng.NewXContext(c)
	var req *TrafficDistReq
	if !xc.GetReqObject(&req) {
		return
	}
	if !req.checkParam() {
		xc.ReplyFail(lib.CodePara)
		conf.Logger.Error("fail to check param", "req", req)
		return
	}

	//some operations


	if err := dist.TrafficDistService(req); err != nil {
		conf.Logger.Error("TrafficDistService", "req", req, "err", err)
		xc.ReplyFail(lib.CodeSrv)
		return
	}
	xc.ReplyOK(nil)
}
