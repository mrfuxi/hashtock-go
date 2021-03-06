package models

import (
    "errors"
    "fmt"
    "net/http"
    "time"

    "appengine"
    "appengine/datastore"
    "code.google.com/p/go-uuid/uuid"

    "github.com/hashtock/hashtock-go/core"
)

func orderKey(ctx appengine.Context, order_uuid string) (key *datastore.Key) {
    return datastore.NewKey(ctx, orderKind, order_uuid, 0, nil)
}

func newOrderSystem(req *http.Request, base_order OrderBase) (order OrderSystem, err error) {
    var (
        profile *Profile
        hashTag *HashTag
    )

    if profile, err = GetProfile(req); err != nil {
        return
    }

    if hashTag, err = GetHashTag(req, base_order.HashTag); err != nil {
        return
    }

    order = OrderSystem{
        UUID:       uuid.New(),
        UserID:     profile.UserID,
        Complete:   false,
        CreatedAt:  time.Now(),
        Resolution: PENDING,
        Value:      base_order.Quantity * hashTag.Value,
    }

    return
}

func PlaceOrder(req *http.Request, base_order OrderBase) (order *Order, err error) {
    if err = base_order.IsValid(req); err != nil {
        return
    }

    var system_order OrderSystem
    if system_order, err = newOrderSystem(req, base_order); err != nil {
        return
    }

    order = &Order{
        OrderBase:   base_order,
        OrderSystem: system_order,
    }

    order.Put(req)

    return
}

func GetOrder(req *http.Request, order_uuid string) (order *Order, err error) {
    ctx := appengine.NewContext(req)

    order = new(Order)
    key := orderKey(ctx, order_uuid)
    err = datastore.Get(ctx, key, order)

    not_found_msg := fmt.Sprintf("Order %#v not found", order_uuid)

    if err == datastore.ErrNoSuchEntity {
        err = core.NewNotFoundError(not_found_msg)
    } else if err != nil {
        err = core.NewInternalServerError(err.Error())
    }

    var ok bool
    ok, err = order.canAccess(req)
    if err != nil {
        err = core.NewInternalServerError(err.Error())
    }

    if !ok {
        err = core.NewNotFoundError(not_found_msg)
        order = nil
    }

    return
}

func CancelOrder(req *http.Request, order_uuid string) (err error) {
    order, err := GetOrder(req, order_uuid)
    if err != nil {
        return
    }

    if !order.isCancellable() {
        err = core.NewBadRequestError("Order can not be cancelled any more")
        return
    }

    order.Delete(req)

    return
}

func allUserOrdersQuery(req *http.Request) (query *datastore.Query, err error) {
    var profile *Profile

    if profile, err = GetProfile(req); err != nil {
        return
    }

    query = allOrdersQuery().Filter("UserID =", profile.UserID)

    return
}

func allOrdersQuery() (query *datastore.Query) {
    return datastore.NewQuery(orderKind)
}

func orderByCompletnessQuery(query *datastore.Query, complete bool) *datastore.Query {
    return query.Filter("Complete =", complete).Order("-CreatedAt")
}

func filterOrderQuery(query *datastore.Query, tag string, resolution string) *datastore.Query {
    if tag != "" {
        query = query.Filter("HashTag =", tag)
    }

    if resolution != "" {
        query = query.Filter("Resolution =", resolution)
    }
    return query
}

func executeOrderQuery(req *http.Request, query *datastore.Query) (orders []Order, err error) {
    ctx := appengine.NewContext(req)
    _, err = query.GetAll(ctx, &orders)

    return
}

func GetActiveUserOrders(req *http.Request) (orders []Order, err error) {
    var query *datastore.Query
    query, err = allUserOrdersQuery(req)
    if err != nil {
        return
    }

    query = orderByCompletnessQuery(query, false)

    return executeOrderQuery(req, query)
}

func GetCompletedUserOrders(req *http.Request, tag string, resolution string) (orders []Order, err error) {
    var query *datastore.Query
    query, err = allUserOrdersQuery(req)
    if err != nil {
        return
    }

    query = orderByCompletnessQuery(query, true)
    query = filterOrderQuery(query, tag, resolution)

    return executeOrderQuery(req, query)
}

func GetAllActiveBankOrders(req *http.Request) (orders []Order, err error) {
    query := allOrdersQuery()
    query = query.Filter("Complete =", false)
    query = query.Filter("BankOrder =", true)

    return executeOrderQuery(req, query)
}

func ExecuteBankOrder(req *http.Request, order Order) (err error) {
    var (
        profile  *Profile
        hashTag  *HashTag
        tagShare *TagShare
    )

    // It's time to blow up if asked to execute non bank order here
    if !order.BankOrder {
        panic(errors.New("execution of non bank order"))
    }

    if hashTag, err = GetHashTag(req, order.HashTag); err != nil {
        order.markAsComplete(req, ERROR, "")
        return
    }

    if profile, err = getProfileForUserId(req, order.UserID); err != nil {
        order.markAsComplete(req, ERROR, "")
        return
    }

    if tagShare, err = getOrCreateTagShare(req, profile, order.HashTag); err != nil {
        order.markAsComplete(req, ERROR, "")
        return
    }

    if order.isBuy() {
        if profile.Founds < order.Value {
            order.markAsComplete(req, FAILURE, "Not enough founds")
            msg := fmt.Sprintf("User %v does not have enough founds to complete %v", profile, order)
            return core.NewBadRequestError(msg)
        }

        if hashTag.InBank < order.Quantity {
            order.markAsComplete(req, FAILURE, "Not enough shares in bank")
            msg := fmt.Sprintf("Bank does not have enough shares to complete %v", order)
            return core.NewBadRequestError(msg)
        }

        profile.Founds -= order.Value
        tagShare.Quantity += order.Quantity
        hashTag.InBank -= order.Quantity

        profile.Put(req)
        tagShare.Put(req)
        hashTag.Put(req)
    } else {
        if tagShare.Quantity < order.Quantity {
            order.markAsComplete(req, FAILURE, "Not enough shares in users possession")
            msg := fmt.Sprintf("User %v does not have enough shares to complete %v", profile, order)
            return core.NewBadRequestError(msg)
        }

        profile.Founds += order.Value
        tagShare.Quantity -= order.Quantity
        hashTag.InBank += order.Quantity

        profile.Put(req)
        hashTag.Put(req)
        if tagShare.Quantity < minShareStep/100.0 {
            tagShare.Delete(req)
        } else {
            tagShare.Put(req)
        }
    }

    order.markAsComplete(req, SUCCESS, "")

    return
}
