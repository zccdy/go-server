// Copyright 2019 Axetroy. All rights reserved. MIT license.
package help

import (
	"errors"
	"github.com/asaskevich/govalidator"
	"github.com/axetroy/go-server/src/controller"
	"github.com/axetroy/go-server/src/exception"
	"github.com/axetroy/go-server/src/model"
	"github.com/axetroy/go-server/src/schema"
	"github.com/axetroy/go-server/src/service/database"
	"github.com/gin-gonic/gin"
	"github.com/jinzhu/gorm"
	"github.com/mitchellh/mapstructure"
	"net/http"
	"time"
)

type UpdateParams struct {
	Title    *string           `json:"title"`
	Content  *string           `json:"content"`
	Tags     *[]string         `json:"tags"`
	Status   *model.HelpStatus `json:"status"`
	Type     *model.HelpType   `json:"type"`
	ParentId *string           `json:"parent_id"`
}

func Update(context controller.Context, helpId string, input UpdateParams) (res schema.Response) {
	var (
		err          error
		data         schema.Help
		tx           *gorm.DB
		shouldUpdate bool
		isValidInput bool
	)

	defer func() {
		if r := recover(); r != nil {
			switch t := r.(type) {
			case string:
				err = errors.New(t)
			case error:
				err = t
			default:
				err = exception.Unknown
			}
		}

		if tx != nil {
			if err != nil || !shouldUpdate {
				_ = tx.Rollback().Error
			} else {
				err = tx.Commit().Error
			}
		}

		if err != nil {
			res.Message = err.Error()
			res.Data = nil
		} else {
			res.Data = data
			res.Status = schema.StatusSuccess
		}
	}()

	// 参数校验
	if isValidInput, err = govalidator.ValidateStruct(input); err != nil {
		return
	} else if isValidInput == false {
		err = exception.InvalidParams
		return
	}

	tx = database.Db.Begin()

	adminInfo := model.Admin{
		Id: context.Uid,
	}

	if err = tx.First(&adminInfo).Error; err != nil {
		// 没有找到管理员
		if err == gorm.ErrRecordNotFound {
			err = exception.AdminNotExist
		}
		return
	}

	helpInfo := model.Help{
		Id: helpId,
	}

	if err = tx.First(&helpInfo).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			err = exception.NoData
			return
		}
		return
	}

	updateModel := model.Help{}

	if input.Title != nil {
		shouldUpdate = true
		updateModel.Title = *input.Title
	}

	if input.Content != nil {
		shouldUpdate = true
		updateModel.Content = *input.Content
	}

	if input.Tags != nil {
		shouldUpdate = true
		updateModel.Tags = *input.Tags
	}

	if input.Status != nil {
		shouldUpdate = true
		updateModel.Status = *input.Status
	}

	if input.Type != nil {
		shouldUpdate = true
		updateModel.Type = *input.Type
	}

	if input.ParentId != nil {
		shouldUpdate = true
		updateModel.ParentId = input.ParentId
		// check parent id exist or not
		if er := tx.Model(&helpInfo).Where(model.Help{Id: *input.ParentId}).First(&model.Help{}).Error; er != nil {
			err = er
			return
		}
	}

	if shouldUpdate {
		if err = tx.Model(&helpInfo).Updates(&updateModel).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				err = exception.NoData
				return
			}
			return
		}
	}

	if err = mapstructure.Decode(helpInfo, &data.HelpPure); err != nil {
		return
	}

	data.CreatedAt = helpInfo.CreatedAt.Format(time.RFC3339Nano)
	data.UpdatedAt = helpInfo.UpdatedAt.Format(time.RFC3339Nano)

	return
}

func UpdateRouter(context *gin.Context) {
	var (
		err   error
		res   = schema.Response{}
		input UpdateParams
	)

	defer func() {
		if err != nil {
			res.Data = nil
			res.Message = err.Error()
		}
		context.JSON(http.StatusOK, res)
	}()

	id := context.Param("help_id")

	if err = context.ShouldBindJSON(&input); err != nil {
		err = exception.InvalidParams
		return
	}

	res = Update(controller.NewContext(context), id, input)
}
