package apis

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	jwtgo "github.com/dgrijalva/jwt-go" //需要安装 然后调用这个jwt-go包
	"github.com/gin-gonic/gin"
	"github.com/xdtest/project/middleware/jwt"
	. "github.com/xdtest/project/models"
)

func Getuserslist(c *gin.Context) {
	var user User
	users, err := user.Listusers()
	if err != nil {
		log.Fatalln(err)
	}
	c.JSON(http.StatusOK, gin.H{
		"msg": users,
	})
}

func Addnewuser(c *gin.Context) {
	var user User
	name := c.Request.FormValue("name")
	password := c.Request.FormValue("password")
	user.Name = name
	user.Password = password
	id, err := user.Adduser()
	if err != nil {
		if err.Error()[:10] == "Error 1062" {
			fmt.Println("ssssss")
			c.JSON(http.StatusOK, gin.H{
				"msg": "用户名已存在",
			})
		}

	} else {
		msg := fmt.Sprintf("创建新的用户成功 用户id为:%d", id)
		c.JSON(http.StatusOK, gin.H{
			"msg": msg,
		})
	}

}

type LoginReq struct {
	Name     string `json:name`
	Password string `json:password`
}

type LoginResult struct {
	User  interface{}
	Token string
}

// 生成令牌  创建jwt风格的token
func GenerateToken(c *gin.Context, user User) {
	j := &jwt.JWT{
		[]byte("newtrekWang"),
	}
	claims := jwt.CustomClaims{
		user.Id,
		user.Name,
		user.Password,
		jwtgo.StandardClaims{
			NotBefore: int64(time.Now().Unix() - 1000), // 签名生效时间
			ExpiresAt: int64(time.Now().Unix() + 3600), // 过期时间 一小时
			Issuer:    "newtrekWang",                   //签名的发行者
		},
	}

	token, err := j.CreateToken(claims)

	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": -1,
			"msg":    err.Error(),
		})
		return
	}

	log.Println(token)

	data := LoginResult{
		User:  user,
		Token: token,
	}
	c.JSON(http.StatusOK, gin.H{
		"status": 0,
		"msg":    "登录成功！",
		"data":   data,
	})
	return
}

// GetDataByTime 一个需要token认证的测试接口
func GetDataByTime(c *gin.Context) {
	claims := c.MustGet("claims").(*jwt.CustomClaims)
	if claims != nil {
		c.JSON(http.StatusOK, gin.H{
			"status": 0,
			"msg":    "token有效",
			"data":   claims,
		})
	}
}

func Userlogin(c *gin.Context) {
	var user User
	if c.Bind(&user) == nil { //把form格式传过来的数据绑定到结构体user中去
		msg, err := user.Login()
		if err != nil {
			if err.Error() == "record not found" {
				c.JSON(http.StatusOK, gin.H{
					"msg":  "用户不存在",
					"user": nil,
				})
			} else {
				c.JSON(http.StatusOK, gin.H{
					"msg":  "登陆错误",
					"user": nil,
				})

			}

		} else {
			GenerateToken(c, msg) //创建token
			// c.JSON(http.StatusOK, gin.H{
			// 	"msg":  "登陆成功",
			// 	"user": msg,
			// })
		}
	} else {
		c.JSON(400, gin.H{"JSON=== status": "binding JSON error!"})
	}
}

func Deleteuser(c *gin.Context) {
	ids := c.Query("id")
	id, _ := strconv.Atoi(ids)
	var u User
	u.Id = id
	user, err := u.Deleteuser(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "用户不存在",
			"user": nil,
		})
	} else {
		c.JSON(http.StatusOK, gin.H{
			"msg":  "删除陈工",
			"user": user,
		})
	}
}

func Updatauser(c *gin.Context) {
	ids := c.Query("id")
	name := c.DefaultPostForm("name", "")
	password := c.DefaultPostForm("password", "")

	id, _ := strconv.Atoi(ids)
	var user User
	if name != "" {
		user.Name = name
	}
	if password != "" {
		user.Password = password
	}
	result, err := user.Updatauser(id)
	if err != nil || result.Id == 0 {
		c.JSON(http.StatusOK, gin.H{
			"code":    -1,
			"message": "修改失败",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"code":    1,
		"message": "修改成功",
	})

}
