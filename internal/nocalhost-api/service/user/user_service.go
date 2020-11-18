/*
Copyright 2020 The Nocalhost Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package user

import (
	"context"
	"time"

	uuid "github.com/satori/go.uuid"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"

	"nocalhost/internal/nocalhost-api/model"
	"nocalhost/internal/nocalhost-api/repository/user"
	"nocalhost/pkg/nocalhost-api/pkg/auth"
	"nocalhost/pkg/nocalhost-api/pkg/token"
)

const (
	// MaxID 最大id
	MaxID = 0xffffffffffff
)

// 如果 userService 没有实现 UserService 报错
var _ UserService = (*userService)(nil)

// UserService 用户服务接口定义
// 使用大写对外暴露方法
type UserService interface {
	Create(ctx context.Context, email, password, name string, status uint64) error
	Delete(ctx context.Context, id uint64) error
	Register(ctx context.Context, email, password string) error
	EmailLogin(ctx context.Context, email, password string) (tokenStr string, err error)
	GetUserByID(ctx context.Context, id uint64) (*model.UserBaseModel, error)
	GetUserByPhone(ctx context.Context, phone int64) (*model.UserBaseModel, error)
	GetUserByEmail(ctx context.Context, email string) (*model.UserBaseModel, error)
	UpdateUser(ctx context.Context, id uint64, user *model.UserBaseModel) error
	GetUserList(ctx context.Context) ([]*model.UserList, error)
	Close()
}

type userService struct {
	userRepo user.BaseRepo
}

func NewUserService() UserService {
	db := model.GetDB()
	return &userService{
		userRepo: user.NewUserRepo(db),
	}
}

func (srv *userService) GetUserList(ctx context.Context) ([]*model.UserList, error) {
	return srv.userRepo.GetUserList(ctx)
}

// Delete 删除用户
func (srv *userService) Delete(ctx context.Context, id uint64) error {
	err := srv.userRepo.Delete(ctx, id)
	if err != nil {
		return errors.Wrapf(err, "delete user fail")
	}
	return nil
}

// Create 创建用户
func (srv *userService) Create(ctx context.Context, email, password, name string, status uint64) error {
	pwd, err := auth.Encrypt(password)
	if err != nil {
		return errors.Wrapf(err, "encrypt password err")
	}

	u := model.UserBaseModel{
		Password:  pwd,
		Email:     email,
		Name:      name,
		Status:    &status,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
		Uuid:      uuid.NewV4().String(),
	}
	_, err = srv.userRepo.Create(ctx, u)
	if err != nil {
		return errors.Wrapf(err, "create user")
	}
	return nil
}

// Register 注册用户
func (srv *userService) Register(ctx context.Context, email, password string) error {
	pwd, err := auth.Encrypt(password)
	if err != nil {
		return errors.Wrapf(err, "encrypt password err")
	}

	u := model.UserBaseModel{
		Password:  pwd,
		Email:     email,
		CreatedAt: time.Time{},
		UpdatedAt: time.Time{},
		Uuid:      uuid.NewV4().String(),
	}
	_, err = srv.userRepo.Create(ctx, u)
	if err != nil {
		return errors.Wrapf(err, "create user")
	}
	return nil
}

// EmailLogin 邮箱登录
func (srv *userService) EmailLogin(ctx context.Context, email, password string) (tokenStr string, err error) {
	u, err := srv.GetUserByEmail(ctx, email)
	if err != nil {
		return "", errors.Wrapf(err, "get user info err by email")
	}

	// Compare the login password with the user password.
	err = auth.Compare(u.Password, password)
	if err != nil {
		return "", errors.Wrapf(err, "password compare err")
	}

	if *u.Status == 0 {
		return "", errors.New("user not allow")
	}

	// 签发签名 Sign the json web token.
	tokenStr, err = token.Sign(ctx, token.Context{UserID: u.ID, Username: u.Username, Uuid: u.Uuid, Email: u.Email, IsAdmin: u.IsAdmin}, "")
	if err != nil {
		return "", errors.Wrapf(err, "gen token sign err")
	}

	return tokenStr, nil
}

// UpdateUser update user info
func (srv *userService) UpdateUser(ctx context.Context, id uint64, user *model.UserBaseModel) error {
	err := srv.userRepo.Update(ctx, id, user)

	if err != nil {
		return err
	}

	return nil
}

// GetUserByID 获取用户信息
func (srv *userService) GetUserByID(ctx context.Context, id uint64) (*model.UserBaseModel, error) {
	userModel, err := srv.userRepo.GetUserByID(ctx, id)
	if err != nil {
		return userModel, errors.Wrapf(err, "get user info err from db by id: %d", id)
	}

	return userModel, nil
}

func (srv *userService) GetUserByPhone(ctx context.Context, phone int64) (*model.UserBaseModel, error) {
	userModel, err := srv.userRepo.GetUserByPhone(ctx, phone)
	if err != nil || gorm.IsRecordNotFoundError(err) {
		return userModel, errors.Wrapf(err, "get user info err from db by phone: %d", phone)
	}

	return userModel, nil
}

func (srv *userService) GetUserByEmail(ctx context.Context, email string) (*model.UserBaseModel, error) {
	userModel, err := srv.userRepo.GetUserByEmail(ctx, email)
	if err != nil || gorm.IsRecordNotFoundError(err) {
		return userModel, errors.Wrapf(err, "get user info err from db by email: %s", email)
	}

	return userModel, nil
}

// Close close all user repo
func (srv *userService) Close() {
	srv.userRepo.Close()
}