// Package delivery 提供投递接口与类型。
package delivery

import "diting/internal/models"

// DeliverOptions 投递时的可选参数（如渠道类型、消息摘要）。
// 与 ConfirmationObject 解耦，可从 ConfirmationObject 中提取所需字段传入。
type DeliverOptions struct {
	ConfirmerIDs []string
	Summary      string
	ChannelType  string // 如 feishu / cli
}

// DeliverInput 供 Deliver 使用的入参，包含待确认对象与选项。
type DeliverInput struct {
	Object *models.ConfirmationObject
	Options *DeliverOptions
}
