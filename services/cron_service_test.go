// Cron service
// Run as part of service test suite
package services_test

import (
    "net/http"

    "github.com/hashtock/hashtock-go/models"
)

// TODO(tests): Those tests need to be simplified somehow :(
// And we need to test handling errors
func (s *ServicesTestSuite) TestExecuteBankBuyOrders() {
    user_req := s.DummyRequest(s.User)
    tag := models.HashTag{HashTag: "Tag1", Value: 2.00, InBank: 100}
    tag.Put(user_req)
    order := models.OrderBase{
        Action:    "buy",
        BankOrder: true,
        HashTag:   "Tag1",
        Quantity:  3,
    }
    models.PlaceOrder(user_req, order)

    // Initial state check
    profile, _ := models.GetProfile(user_req)
    activeOrders, _ := models.GetActiveUserOrders(user_req)
    completedOrders, _ := models.GetCompletedUserOrders(user_req, "", "")
    shares, _ := models.GetProfileShares(user_req, profile)
    s.Len(activeOrders, 1)
    s.Len(completedOrders, 0)
    s.Equal(profile.Founds, models.StartingFounds)
    s.Len(shares, 0)

    // Execute bank orders
    cron_req := s.NewUserRequest("GET", "/_cron/bank-orders/", nil, nil)
    cron_req.Header.Set("X-AppEngine-Cron", "true")
    rec := s.Do(cron_req)

    // Request went well
    s.Equal(http.StatusOK, rec.Code, rec.Body.String())

    // New request to get real, not cached data
    user_req = s.DummyRequest(s.User)

    expected_share := models.TagShare{
        HashTag:  "Tag1",
        UserID:   s.User.Email,
        Quantity: 3,
    }

    // State after
    profileAfter, _ := models.GetProfile(user_req)
    activeOrdersAfter, _ := models.GetActiveUserOrders(user_req)
    completedOrdersAfter, _ := models.GetCompletedUserOrders(user_req, "", "")
    sharesAfter, _ := models.GetProfileShares(user_req, profile)
    s.Len(activeOrdersAfter, 0)
    s.Len(completedOrdersAfter, 1)
    s.Equal(profileAfter.Founds, models.StartingFounds-2*3)
    s.Equal(sharesAfter, []models.TagShare{expected_share})
}

func (s *ServicesTestSuite) TestExecuteBankSellOrders() {
    user_req := s.DummyRequest(s.User)
    tag := models.HashTag{HashTag: "Tag1", Value: 2.00, InBank: 10}
    tag.Put(user_req)
    order := models.OrderBase{
        Action:    "sell",
        BankOrder: true,
        HashTag:   "Tag1",
        Quantity:  3,
    }
    models.PlaceOrder(user_req, order)
    user_share := models.TagShare{
        HashTag:  "Tag1",
        UserID:   s.User.Email,
        Quantity: 4,
    }
    user_share.Put(user_req)

    // Initial state check
    profile, _ := models.GetProfile(user_req)
    activeOrders, _ := models.GetActiveUserOrders(user_req)
    completedOrders, _ := models.GetCompletedUserOrders(user_req, "", "")
    shares, _ := models.GetProfileShares(user_req, profile)
    s.Len(activeOrders, 1)
    s.Len(completedOrders, 0)
    s.Equal(models.StartingFounds, profile.Founds)
    s.Equal([]models.TagShare{user_share}, shares)

    // Execute bank orders
    cron_req := s.NewUserRequest("GET", "/_cron/bank-orders/", nil, nil)
    cron_req.Header.Set("X-AppEngine-Cron", "true")
    rec := s.Do(cron_req)

    // Request went well
    s.Equal(http.StatusOK, rec.Code, rec.Body.String())

    // New request to get real, not cached data
    user_req = s.DummyRequest(s.User)

    expected_share := models.TagShare{
        HashTag:  "Tag1",
        UserID:   s.User.Email,
        Quantity: 1,
    }
    expectedHashTagAfter := &models.HashTag{HashTag: "Tag1", Value: 2.00, InBank: 13}

    // State after
    profileAfter, _ := models.GetProfile(user_req)
    activeOrdersAfter, _ := models.GetActiveUserOrders(user_req)
    completedOrdersAfter, _ := models.GetCompletedUserOrders(user_req, "", "")
    sharesAfter, _ := models.GetProfileShares(user_req, profile)
    hashTagAfter, _ := models.GetHashTag(user_req, "Tag1")
    s.Len(activeOrdersAfter, 0)
    s.Len(completedOrdersAfter, 1)
    s.Equal(models.StartingFounds+2*3, profileAfter.Founds)
    s.Equal([]models.TagShare{expected_share}, sharesAfter)
    s.Equal(expectedHashTagAfter, hashTagAfter)
}
