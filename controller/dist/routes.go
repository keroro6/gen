package dist

import "github.com/gin-gonic/gin"

func InitRouters(app *gin.Engine) {
	group := app.Group("traffic")
	group.POST("dist", TrafficDist)  //@c
	group.POST("get", TrafficGetIds) //@r
	group.POST("list", TrafficList)  //@r
	group.GET(":id", IsDisting)
	group.POST("dist_content", TrafficDistContent) //@
}
