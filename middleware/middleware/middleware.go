// package middleware

// import (
// 	"fmt"

// 	"github.com/gin-gonic/gin"
// )

// func JwtMiddleWare() gin.HandlerFunc {
// 	return func(c *gin.Context) {
// 		jwt := c.PostForm("jwt")
// 		fmt.Println("before middleware", jwt)
// 		c.Set("jwt", jwt)               //中间件进行的操作
// 		c.Next()                        //使用中间件的api进行对应的操作
// 		fmt.Println("after middleware") //返回时经过这里，类似于django中间件
// 	}
// }
