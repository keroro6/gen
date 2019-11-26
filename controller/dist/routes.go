package dist

import "github.com/gin-gonic/gin"

func InitRouters(app *gin.Engine) {
	group := app.Group("traffic")
	group.POST("dist", TrafficDist)
	group.POST("get", TrafficGetIds)
	group.POST("list", TrafficList)
	group.GET(":id", IsDisting)
	group.POST("dist_content", TrafficDistContent)
}
