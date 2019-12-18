package database

import (
	"fmt"

	_ "github.com/go-sql-driver/mysql" //加载mysql,用他的 init配置  所以前边加的 _
	"github.com/jinzhu/gorm"           //使用gorm来链接和操作数据库
)

var Eloquent *gorm.DB

func init() {
	var err error
	Eloquent, err = gorm.Open("mysql", "root:root1234@tcp(127.0.0.1:3306)/go_test?parseTime=true")

	if err != nil {
		fmt.Printf("mysql connect error %v", err)
	}

	if Eloquent.Error != nil {
		fmt.Printf("database error %v", Eloquent.Error)
	}
}
