package main

import (
	gorm "github.com/xdtest/project/database"
	model "github.com/xdtest/project/models"
	routers "github.com/xdtest/project/routers"
)

func main() {
	gorm.Eloquent.AutoMigrate(&model.User{}) //如果数据表结构发生变化自动更新mysql数据库结构
	defer gorm.Eloquent.Close()              //关闭数据库链接
	router := routers.InitRouter()           //指定路由
	router.Run(":8000")                      //在8000端口上运行
}
