package models

import (
	"fmt"
	// "log"

	orm "github.com/xdtest/project/database"
)

type User struct {
	Name     string `form:"name",json:"name",bingding:"required",gorm:"unique;not null"`
	Password string `form:"password",json:"password",bingding:"required"，gorm:"NOT NULL"`
	Id       int    `form:"id"，gorm:"PRIMARY_KEY"`
	Role     int    `gorm:"column:role_id"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) Adduser() (id int, err error) { //user对象的方法 可以直接user.Adduser方法来完成添加记录
	result := orm.Eloquent.Create(&u)
	id = u.Id
	if result.Error != nil {
		err = result.Error
		fmt.Println("这是错误", err, "这是错误")
		fmt.Printf("sdf%d", id)
		return
	}
	return

}

func (u *User) Listusers() (users []User, err error) {

	if err = orm.Eloquent.Find(&users).Error; err != nil {
		return
	}
	return

}

func (u *User) Login() (user1 User, err error) {
	obj := orm.Eloquent.Where("name=? and password=?", u.Name, u.Password).First(&user1)
	if err = obj.Error; err != nil {
		fmt.Printf("这是登陆错误  %v 和 %T", err, err)
		return
	}
	fmt.Println(user1)
	return

}

func (user *User) Deleteuser(id int) (Result User, err error) {
	if err = orm.Eloquent.Select([]string{"id"}).First(&user, id).Error; err != nil {
		return
	}
	if err = orm.Eloquent.Delete(&user).Error; err != nil {
		return
	}
	Result = *user
	return

}

func (user *User) Updatauser(id int) (updatauser User, err error) {
	if err = orm.Eloquent.Select([]string{"id"}).First(&updatauser, id).Error; err != nil {
		return
	}
	if err = orm.Eloquent.Model(&updatauser).Update(&user).Error; err != nil {
		return
	}
	return

}
