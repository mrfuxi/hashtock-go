package main

import (
    "net/http"

    "github.com/gorilla/mux"

    "github.com/hashtock/hashtock-go/api"
    "github.com/hashtock/hashtock-go/http_utils"

    "github.com/hashtock/hashtock-go/models"
)

type HashTagService struct{}

func (h *HashTagService) Name() string {
    return "tag"
}

func (h *HashTagService) EndPoints() (endpoints []*api.EndPoint) {
    tags := api.NewEndPoint("/", "GET", "tags", ListOfAllHashTags)
    tag_info := api.NewEndPoint("/{tag}/", "GET", "tag_info", TagInfo)
    update_tag := api.NewEndPoint("/{tag}/", "POST", "update_tag", nil) // Update info about the tag (admin)

    endpoints = []*api.EndPoint{
        tags,
        tag_info,
        update_tag,
    }
    return
}

// List of all tags with bank values
func ListOfAllHashTags(rw http.ResponseWriter, req *http.Request) {
    tags, err := models.GetAllHashTags(req)

    if err != nil {
        http.Error(rw, err.Error(), http.StatusInternalServerError)
        return
    }

    http_utils.SerializeResponse(rw, req, tags, http.StatusOK)
}

// Details about the hash tag
func TagInfo(rw http.ResponseWriter, req *http.Request) {
    vars := mux.Vars(req)
    hash_tag_name := vars["tag"]

    tag, err := models.GetHashTag(req, hash_tag_name)

    if err != nil {
        http_utils.SerializeErrorResponse(rw, req, err)

        return
    }

    http_utils.SerializeResponse(rw, req, tag, http.StatusOK)
}