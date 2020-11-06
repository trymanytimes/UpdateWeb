package resource

import (
	"github.com/zdnscloud/gorest/resource"
)

type UploadLog struct {
	resource.ResourceBase `json:",inline"`
	UserName              string          `json:"userName" rest:"required=true"`
	Password              string          `json:"password" rest:"required=true"`
	Address               string          `json:"address" rest:"required=true"`
	Status                UploadLogStatus `json:"status"`
	Comment               string          `json:"comment"`
	FileName              string          `json:"fileName"`
	FinishTime            string          `json:"finishTime"`
}

const ActionUploadLog = "uploadLog"

type UploadLogStatus string

const (
	FtpStatusConnecting         UploadLogStatus = "connecting"
	FtpStatusConnFailed         UploadLogStatus = "connectFailed"
	FtpStatusTransporting       UploadLogStatus = "transporting"
	FtpStatusTransportFailed    UploadLogStatus = "transportFailed"
	FtpStatusTransportCompleted UploadLogStatus = "completed"
)

type UploadLogInput struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
	Address  string `json:"address"`
}

type UploadLogOutput struct {
	Status UploadLogStatus `json:"status"`
}

func (uploadLog UploadLog) GetActions() []resource.Action {
	return []resource.Action{
		resource.Action{
			Name:   ActionUploadLog,
			Input:  &UploadLogInput{},
			Output: &UploadLogOutput{},
		},
	}
}
