// Current user service
// Run as part of service test suite
package services_test

import (
    "net/http"

    "github.com/hashtock/hashtock-go/gaetestsuite"
    "github.com/hashtock/hashtock-go/models"
)

// TODO(security): It would be good to expand this to test ALL api urls
func (s *ServicesTestSuite) TestUserHasToBeLoggedIn() {
    expectedStatus := http.StatusForbidden
    expected := gaetestsuite.Json{
        "code":  expectedStatus,
        "error": http.StatusText(expectedStatus),
    }

    rec := s.ExecuteJsonRequest("GET", "/api/user/", nil, s.NoUser)
    json_body := s.JsonResponceToStringMap(rec)

    s.Equal(expected, json_body)
    s.Equal(expectedStatus, rec.Code)
}

func (s *ServicesTestSuite) TestProfileExistForLoggedInUser() {
    rec := s.ExecuteJsonRequest("GET", "/api/user/", nil, s.User)
    json_body := s.JsonResponceToStringMap(rec)

    expected := gaetestsuite.Json{
        "id":     "user@here.prv",
        "founds": models.StartingFounds,
    }

    s.Equal(http.StatusOK, rec.Code)
    s.Equal(expected, json_body)
}

func (s *ServicesTestSuite) TestGetUsersTags() {
    req := s.NewJsonRequest("GET", "/api/portfolio/", nil, s.User)

    t1 := models.TagShare{HashTag: "Tag1", Quantity: 10.5, UserID: s.User.Email}
    t2 := models.TagShare{HashTag: "bTag", Quantity: 0.20, UserID: s.User.Email}
    t3 := models.TagShare{HashTag: "Tag3", Quantity: 1.20, UserID: s.User.Email}
    t4 := models.TagShare{HashTag: "aTag", Quantity: 0.20, UserID: s.User.Email}
    t5 := models.TagShare{HashTag: "Tag1", Quantity: 1.00, UserID: "OtherID"}
    t1.Put(req)
    t2.Put(req)
    t3.Put(req)
    t4.Put(req)
    t5.Put(req)

    rec := s.Do(req)
    json_body := s.JsonResponceToListOfStringMap(rec)

    // Order matters
    expected := gaetestsuite.JsonList{
        gaetestsuite.Json{
            "hashtag":  "Tag1",
            "quantity": 10.5,
        },
        gaetestsuite.Json{
            "hashtag":  "Tag3",
            "quantity": 1.2,
        },
        gaetestsuite.Json{
            "hashtag":  "aTag",
            "quantity": 0.2,
        },
        gaetestsuite.Json{
            "hashtag":  "bTag",
            "quantity": 0.2,
        },
    }

    s.Equal(http.StatusOK, rec.Code)
    s.JsonListEqual(expected, json_body)
}
