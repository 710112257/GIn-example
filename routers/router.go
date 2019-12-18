package routers

import (
	"io"
	"os"

	"github.com/gin-gonic/gin"
	. "github.com/xdtest/project/apis"
	"github.com/xdtest/project/middleware/jwt"
)

func InitRouter() *gin.Engine {
	f, _ := os.Create("logs/productions.log")
	gin.DefaultWriter = io.MultiWriter(f)

	router := gin.Default()
	v1 := router.Group("/v1")
	v1.Use(jwt.JWTAuth())                              //v1 使用jwt中间件进行前后验证
	router.GET("/user_list_new_handler", Getuserslist) //注意这里调用handler方法直接调用函数名
	router.POST("/register", Addnewuser)               //注意这里调用handler方法直接调用函数名
	router.POST("/login", Userlogin)                   //注意这里调用handler方法直接调用函数名
	router.DELETE("/deleteuser", Deleteuser)           //注意这里调用handler方法直接调用函数名
	v1.POST("/updatauser", Updatauser)                 //注意这里调用handler方法直接调用函数名
	v1.POST("/test", GetDataByTime)                    //使用中间件，验证token， 函数也是验证用户带的token
	return router
}
