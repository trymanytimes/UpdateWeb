package handler

import (
	"fmt"
	"net"
	"strings"

	restdb "github.com/zdnscloud/gorest/db"
	resterror "github.com/zdnscloud/gorest/error"
	restresource "github.com/zdnscloud/gorest/resource"

	"github.com/linkingthing/ddi-agent/pkg/dns/kafkaconsumer"
	pb "github.com/linkingthing/ddi-agent/pkg/proto"
	"github.com/trymanytimes/UpdateWeb/pkg/auth/authentification"
	"github.com/trymanytimes/UpdateWeb/pkg/db"
	"github.com/trymanytimes/UpdateWeb/pkg/dns/resource"
	"github.com/trymanytimes/UpdateWeb/pkg/util"
)

var (
	TableView    = restdb.ResourceDBType(&resource.View{})
	TableViewAcl = restdb.ResourceDBType(&resource.ViewAcl{})
	defaultView  = "default"
)

type ViewHandler struct {
}

func NewViewHandler() (*ViewHandler, error) {
	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		if exists, err := tx.Exists(TableView, map[string]interface{}{"id": defaultView}); err != nil {
			return err
		} else if exists == false {
			view := &resource.View{Name: defaultView, Priority: 1, Acls: []string{"any"}}
			view.SetID(defaultView)
			if _, err := tx.Insert(view); err != nil {
				return err
			}
		}

		if exists, err := tx.Exists(TableViewAcl, map[string]interface{}{"id": "1"}); err != nil {
			return err
		} else if exists == false {
			viewAcl := &resource.ViewAcl{View: defaultView, Acl: "any"}
			viewAcl.SetID("1")
			if _, err := tx.Insert(viewAcl); err != nil {
				return err
			}
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &ViewHandler{}, nil
}

func (h *ViewHandler) Create(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	view := ctx.Resource.(*resource.View)
	view.Name = strings.ToLower(view.Name)
	if err := view.GenerateKey(); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("generate view %s fail: %s", view.Name, err.Error()))
	}
	if err := h.checkView(view); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("check the view %s fail: %s", view.Name, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		viewPrioritys, err := adjustPriority(view, tx, false)
		if err != nil {
			return err
		}
		view.SetID(view.Name)
		if _, err = tx.Insert(view); err != nil {
			return err
		}
		if err = createACLForView(view, tx); err != nil {
			return err
		}

		return SendKafkaMessage(view.ID, kafkaconsumer.CreateView,
			&pb.CreateViewReq{
				Header:       &pb.DDIRequestHead{Method: "Create", Resource: view.GetType()},
				Id:           view.ID,
				Name:         view.Name,
				Priority:     uint32(view.Priority),
				Dns64:        view.Dns64,
				Acls:         view.Acls,
				Key:          view.Key,
				ViewPriority: viewPrioritys,
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("create view %s failed: %s", view.Name, err.Error()))
	}

	return view, nil
}

func adjustPriority(view *resource.View, tx restdb.Transaction, isDelete bool) ([]*pb.ViewPriority, error) {
	if view.GetID() == defaultView {
		return nil, nil
	}

	var views []*resource.View
	if err := db.GetResources(map[string]interface{}{"orderby": "priority"}, &views); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list views from db failed: %s", err.Error()))
	}

	if len(views) != 1 && int(view.Priority) >= len(views) {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("view.Priority update error:%d should < default.Priority(%d)",
				view.Priority, len(views)))
	} else if int(view.Priority) < 1 {
		view.Priority = 1
	}

	for i, v := range views {
		if v.GetID() == view.GetID() {
			views = append(views[:i], views[i+1:]...)
			break
		}
	}
	if !isDelete {
		views = append(views[:view.Priority-1], append([]*resource.View{view}, views[view.Priority-1:]...)...)
	}

	var viewPriority []*pb.ViewPriority
	for i, view := range views {
		if _, err := tx.Update(TableView, map[string]interface{}{
			"priority": i + 1,
		}, map[string]interface{}{restdb.IDField: view.GetID()}); err != nil {
			return nil, err
		}

		viewPriority = append(viewPriority, &pb.ViewPriority{Id: view.ID, Priority: uint32(i + 1)})
	}

	return viewPriority, nil
}

func createACLForView(view *resource.View, tx restdb.Transaction) error {
	for _, id := range view.Acls {
		if _, err := tx.Insert(&resource.ViewAcl{View: view.GetID(), Acl: id}); err != nil {
			return err
		}
	}
	return nil
}

func deleteACLForView(view *resource.View, tx restdb.Transaction) error {
	if _, err := tx.Delete(TableViewAcl, map[string]interface{}{
		"view": view.GetID(),
	}); err != nil {
		return err
	}
	return nil
}

func (h *ViewHandler) Delete(ctx *restresource.Context) *resterror.APIError {
	view := ctx.Resource.(*resource.View)
	if view.GetID() == defaultView {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete view default is fobidden"))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		viewPrioritys, err := adjustPriority(view, tx, true)
		if err != nil {
			return err
		}
		if _, err := tx.Delete(TableView, map[string]interface{}{
			restdb.IDField: view.GetID()}); err != nil {
			return err
		}

		return SendKafkaMessage(view.GetID(), kafkaconsumer.DeleteView,
			&pb.DeleteViewReq{
				Header:       &pb.DDIRequestHead{Method: "Delete", Resource: view.GetType()},
				Id:           view.GetID(),
				ViewPriority: viewPrioritys})
	}); err != nil {
		return resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("delete view %s fail", view.GetID()))
	}

	return nil
}

func checkAclChanged(acls1, acls2 []string) bool {
	if len(acls1) != len(acls2) {
		return true
	}
	for _, a1 := range acls1 {
		theSame := false
		for _, a2 := range acls2 {
			if a2 == a1 {
				theSame = true
			}
		}
		if !theSame {
			return true
		}
	}
	return false
}

func (h *ViewHandler) Update(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	view := ctx.Resource.(*resource.View)
	if err := checkDNS64Valid(&view.Dns64); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("check the view %s's dns64 fail: %s", view.Name, err.Error()))
	}

	if err := restdb.WithTx(db.GetDB(), func(tx restdb.Transaction) error {
		var oldViews []*resource.View
		if err := tx.Fill(map[string]interface{}{restdb.IDField: view.ID}, &oldViews); err != nil {
			return err
		}
		aclChanged := checkAclChanged(oldViews[0].Acls, view.Acls)
		viewPrioritys, err := adjustPriority(view, tx, false)
		if err != nil {
			return err
		}
		if _, err = tx.Update(TableView, map[string]interface{}{
			"priority": view.Priority,
			"acls":     view.Acls,
			"dns64":    view.Dns64,
			"comment":  view.Comment,
		}, map[string]interface{}{restdb.IDField: view.GetID()}); err != nil {
			return err
		}

		if aclChanged {
			if err = deleteACLForView(view, tx); err != nil {
				return err
			}
			if err = createACLForView(view, tx); err != nil {
				return err
			}
		}

		return SendKafkaMessage(view.ID, kafkaconsumer.UpdateView,
			&pb.UpdateViewReq{
				Header:       &pb.DDIRequestHead{Method: "Update", Resource: view.GetType()},
				Id:           view.ID,
				Priority:     uint32(view.Priority),
				Dns64:        view.Dns64,
				Acls:         view.Acls,
				ViewPriority: viewPrioritys,
			})
	}); err != nil {
		return nil, resterror.NewAPIError(resterror.InvalidFormat,
			fmt.Sprintf("update view %s fail: %s", view.ID, err.Error()))
	}

	return view, nil
}

func (h *ViewHandler) Get(ctx *restresource.Context) (restresource.Resource, *resterror.APIError) {
	viewId := ctx.Resource.(*resource.View).GetID()
	var views []*resource.View
	view, err := restdb.GetResourceWithID(db.GetDB(), viewId, &views)
	if err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get view %s from db failed: %s", viewId, err.Error()))
	}
	if err := GetCount(view.(*resource.View)); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("get view %s attriabute sizes from db failed: %s", viewId, err.Error()))
	}

	return view.(*resource.View), nil
}

func (h *ViewHandler) List(ctx *restresource.Context) (interface{}, *resterror.APIError) {
	var views []*resource.View
	if err := db.GetResources(map[string]interface{}{"orderby": "priority"}, &views); err != nil {
		return nil, resterror.NewAPIError(resterror.ServerError,
			fmt.Sprintf("list views from db failed: %s", err.Error()))
	}
	for i, _ := range views {
		if err := GetCount(views[i]); err != nil {
			return nil, resterror.NewAPIError(resterror.ServerError,
				fmt.Sprintf("get views attribute sizes from db failed: %s", err.Error()))
		}
	}

	var visible []*resource.View
	for _, view := range views {
		if authentification.ViewFilter(ctx, view.ID) {
			visible = append(visible, view)
		}
	}

	return visible, nil
}

func GetCount(v *resource.View) error {
	var masterZones []*resource.Zone
	var forwardZones []*resource.ForwardZone
	var redirections []*resource.Redirection
	var urlRedirects []*resource.UrlRedirect
	if err := db.GetResources(map[string]interface{}{"view": v.GetID()}, &masterZones); err != nil {
		return fmt.Errorf("get master zones from db failed: %s", err.Error())
	}
	if err := db.GetResources(map[string]interface{}{"view": v.GetID()}, &forwardZones); err != nil {
		return fmt.Errorf("get forward zones from db failed: %s", err.Error())
	}
	if err := db.GetResources(map[string]interface{}{"view": v.GetID()}, &redirections); err != nil {
		return fmt.Errorf("get redierctions from db failed: %s", err.Error())
	}
	if err := db.GetResources(map[string]interface{}{"view": v.GetID()}, &urlRedirects); err != nil {
		return fmt.Errorf("get url redirects from db failed: %s", err.Error())
	}
	var err error
	v.MasterZoneSize = len(masterZones)
	v.ForwardZoneSize = len(forwardZones)
	v.RRSize, err = GetRRCount(masterZones)
	if err != nil {
		return fmt.Errorf("list views's rr count from db failed: %s", err.Error())
	}
	for _, r := range redirections {
		if r.RedirectType == localZoneType {
			v.LocalZoneSize++
		}
		if r.RedirectType == nxDomainType {
			v.NxdomainSize++
		}
	}
	v.UrlRedirectSize = len(urlRedirects)

	return nil
}

func checkAclConflict(acls []string) bool {
	var haveNone, haveAny bool
	for _, acl := range acls {
		if acl == NoneACL {
			haveNone = true
		} else if acl == AnyACL {
			haveAny = true
		}
	}

	return haveNone && haveAny
}

func checkDNS64Valid(subnet *string) error {
	if *subnet == "" {
		return nil
	}
	ip, ipnet, err := net.ParseCIDR(*subnet)
	if err != nil {
		return fmt.Errorf("parse dns64 cidr err:%s", err.Error())
	}
	if ip.To4() != nil {
		return fmt.Errorf("dns64 invalid,it should be ipv6 network")
	}
	*subnet = ipnet.String()
	size, _ := ipnet.Mask.Size()

	switch size {
	case 32, 40, 48, 56, 64, 96:
		return nil
	default:
		return fmt.Errorf("bad prefix length %d, should be one of [32/40/48/56/64/96]", size)
	}
}

func (h *ViewHandler) checkView(view *resource.View) error {
	if err := util.CheckNameValid(view.Name); err != nil {
		return err
	}
	if err := checkDNS64Valid(&view.Dns64); err != nil {
		return err
	}

	if checkAclConflict(view.Acls) {
		return fmt.Errorf("acls should contain both any and none")
	}

	return nil
}
