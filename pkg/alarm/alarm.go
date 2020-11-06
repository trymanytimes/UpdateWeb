package alarm

import (
	"github.com/gin-gonic/gin"

	"github.com/zdnscloud/gorest"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-controller/pkg/alarm/handler"
	"github.com/linkingthing/ddi-controller/pkg/alarm/resource"
)

var (
	Version = restresource.APIVersion{
		Version: "v1",
		Group:   "linkingthing.com/alarm",
	}
)

func RegisterHandler(apiServer *gorest.Server, router gin.IRoutes) error {
	alarmManager, err := handler.NewAlarmHandler()
	if err != nil {
		return err
	}

	thresholdManager, err := handler.NewThresholdHandler()
	if err != nil {
		return err
	}

	apiServer.Schemas.MustImport(&Version, resource.Alarm{}, alarmManager)
	apiServer.Schemas.MustImport(&Version, resource.Threshold{}, thresholdManager)
	apiServer.Schemas.MustImport(&Version, resource.MailSender{}, handler.NewMailSenderHandler())
	apiServer.Schemas.MustImport(&Version, resource.MailReceiver{}, handler.NewMailReceiverHandler())
	alarmManager.RegisterWSHandler(router)
	return nil
}

func PersistentResources() []restresource.Resource {
	return []restresource.Resource{
		&resource.Alarm{},
		&resource.Threshold{},
		&resource.MailSender{},
		&resource.MailReceiver{},
	}
}

//for gen api doc
func RegistHandler(schemas restresource.SchemaManager) {
	schemas.MustImport(&Version, resource.Alarm{}, &handler.AlarmHandler{})
	schemas.MustImport(&Version, resource.Threshold{}, &handler.ThresholdHandler{})
	schemas.MustImport(&Version, resource.MailSender{}, handler.NewMailSenderHandler())
	schemas.MustImport(&Version, resource.MailReceiver{}, handler.NewMailReceiverHandler())
}
