package model_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/minodisk/qiitactl/api"
	"github.com/minodisk/qiitactl/info"
	"github.com/minodisk/qiitactl/model"
	"github.com/minodisk/qiitactl/testutil"
)

var (
	inf = info.Info{
		Version: "0.0.0",
		TaskSettings: info.TaskSettings{
			GitHub: info.GitHub{
				Name: "qiitactl",
			},
		},
	}
)

func TestNewPost(t *testing.T) {
	post := model.NewPost("Example Title", nil, nil)
	if post.Title != "Example Title" {
		t.Errorf("wrong Title: %s", post.Title)
	}
	if !post.CreatedAt.Equal(post.UpdatedAt.Time) {
		t.Errorf("wrong CreatedAt: %s", post.CreatedAt)
	}
}

func TestNewPostWithFile(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	err := os.MkdirAll("mine/2013/12", 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("mine/2013/12/10-main-title.md", []byte(`<!--
id: abcdefghijklmnopqrst
url: http://example.com/mypost
created_at: 2013-12-10T12:29:14+09:00
updated_at: 2015-02-25T09:26:30+09:00
private: true
coediting: false
tags:
- TypeScript
- Docker:
  - 1.9
- Go:
  - 1.4.3
  - 1.5.3
team: null
-->

# Main title

## Sub title
Paragraph
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	post, err := model.NewPostWithFile("mine/2013/12/10-main-title.md")
	if err != nil {
		t.Fatal(err)
	}
	if post.Meta.ID != "abcdefghijklmnopqrst" {
		t.Errorf("wrong Id")
	}
	if post.Meta.URL != "http://example.com/mypost" {
		t.Errorf("wrong Url")
	}
	if !post.Meta.CreatedAt.Equal(time.Date(2013, 12, 10, 3, 29, 14, 0, time.UTC)) {
		t.Errorf("wrong CreatedAt")
	}
	if !post.Meta.UpdatedAt.Equal(time.Date(2015, 02, 25, 0, 26, 30, 0, time.UTC)) {
		t.Errorf("wrong UpdatedAt")
	}
	if post.Meta.Private != true {
		t.Errorf("wrong Private")
	}
	if post.Meta.Coediting != false {
		t.Errorf("wrong Coediting")
	}
	if len(post.Meta.Tags) != 3 {
		t.Errorf("wrong Tags length: %d", len(post.Meta.Tags))
	} else {
		for _, tag := range post.Meta.Tags {
			switch tag.Name {
			case "TypeScript":
				if len(tag.Versions) != 0 {
					t.Errorf("wrong Tag with no version: %+v", tag)
				}
			case "Docker":
				if len(tag.Versions) != 1 || tag.Versions[0] != "1.9" {
					t.Errorf("wrong Tag with single version: %+v", tag)
				}
			case "Go":
				if len(tag.Versions) != 2 || tag.Versions[0] != "1.4.3" || tag.Versions[1] != "1.5.3" {
					t.Errorf("wrong Tag with multi versions: %+v", tag)
				}
			}
		}
	}
	if post.Title != "Main title" {
		t.Errorf("wrong Title")
	}
	if post.Body != "## Sub title\nParagraph" {
		t.Errorf("wrong Body: %s", post.Body)
	}
}

func TestNewPostWithFileWithNoFile(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	_, err := model.NewPostWithFile("mine/2013/12/10-main-title.md")
	if err == nil {
		t.Fatal("error should occur")
	}
}

func TestNewPostWithFileWithWrongFormattedFile(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	err := os.MkdirAll("mine/2013/12", 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("mine/2013/12/10-main-title.md", []byte(`<!--
id: abcdefghijklmnopqrst
url: http://example.com/mypost
created_at: 2013-12-10T12:29:14+09:00
updated_at: 2015-02-25T09:26:30+09:00
private: true
coediting: false
tags:
- TypeScript
- Docker:
  - 1.9
- Go:
  - 1.4.3
  - 1.5.3
team: null
-->

## Sub title
Paragraph
`), 0644)
	if err != nil {
		t.Fatal(err)
	}

	_, err = model.NewPostWithFile("mine/2013/12/10-main-title.md")
	if err == nil {
		t.Fatal("error should occur")
	}
}

func TestPostCreate(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			b, _ := json.Marshal(api.ResponseError{"method_not_allowed", "Method Not Allowed"})
			w.Write(b)
			return
		}

		defer r.Body.Close()

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
		if string(b) == "" {
			testutil.ResponseAPIError(w, 500, api.ResponseError{
				Type:    "fatal",
				Message: "empty body",
			})
			return
		}

		var post model.CreationPost
		err = json.Unmarshal(b, &post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		if post.Tweet || post.Gist {
			testutil.ResponseError(w, 500, errors.New("tweet and gist should be false"))
			return
		}

		post.CreatedAt = model.Time{Time: time.Date(2016, 2, 1, 12, 51, 42, 0, time.UTC)}
		post.UpdatedAt = post.CreatedAt
		b, err = json.Marshal(post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		_, err = w.Write(b)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)

	prevPath := post.Path
	if err != nil {
		t.Fatal(err)
	}

	err = post.Create(client, model.CreationOptions{})
	if err != nil {
		t.Fatal(err)
	}

	postPath := post.Path
	if err != nil {
		t.Fatal(err)
	}
	if postPath != prevPath {
		t.Errorf("wrong path: expected %s, but actual %s", prevPath, postPath)
	}

	if !post.CreatedAt.Equal(time.Date(2016, 2, 1, 12, 51, 42, 0, time.UTC)) {
		t.Errorf("wrong CreatedAt: %s", post.CreatedAt)
	}
	if !post.UpdatedAt.Equal(time.Date(2016, 2, 1, 12, 51, 42, 0, time.UTC)) {
		t.Errorf("wrong UpdatedAt: %s", post.UpdatedAt)
	}

	testutil.ShouldExistFile(t, 0)
}

func TestPostCreateWithWrongResponseBody(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Non JSON format")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	post := model.NewPost("Example Title", nil, nil)
	err = post.Create(client, model.CreationOptions{})
	if err == nil {
		t.Fatal("error should occur")
	}
}

func TestPostCreateInTeam(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			b, _ := json.Marshal(api.ResponseError{"method_not_allowed", "Method Not Allowed"})
			w.Write(b)
			return
		}

		defer r.Body.Close()

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
		if string(b) == "" {
			testutil.ResponseAPIError(w, 500, api.ResponseError{
				Type:    "fatal",
				Message: "empty body",
			})
			return
		}

		var post model.CreationPost
		err = json.Unmarshal(b, &post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		if post.Tweet || post.Gist {
			testutil.ResponseError(w, 500, errors.New("tweet and gist should be false"))
			return
		}

		post.CreatedAt = model.Time{Time: time.Date(2016, 2, 1, 12, 51, 42, 0, time.UTC)}
		post.UpdatedAt = post.CreatedAt
		b, err = json.Marshal(post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		_, err = w.Write(b)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		if subDomain != "increments" {
			t.Fatalf("wrong sub domain: %s", subDomain)
			return
		}
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, &model.Team{Active: true, ID: "increments", Name: "Increments Inc."})

	prevPath := post.Path
	if err != nil {
		t.Fatal(err)
	}

	err = post.Create(client, model.CreationOptions{})
	if err != nil {
		t.Fatal(err)
	}

	postPath := post.Path
	if err != nil {
		t.Fatal(err)
	}
	if postPath != prevPath {
		t.Errorf("wrong path: expected %s, but actual %s", prevPath, postPath)
	}

	if !post.CreatedAt.Equal(time.Date(2016, 2, 1, 12, 51, 42, 0, time.UTC)) {
		t.Errorf("wrong CreatedAt: %s", post.CreatedAt)
	}
	if !post.UpdatedAt.Equal(time.Date(2016, 2, 1, 12, 51, 42, 0, time.UTC)) {
		t.Errorf("wrong UpdatedAt: %s", post.UpdatedAt)
	}

	testutil.ShouldExistFile(t, 0)
}

func TestPostCreateWithNoServer(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", nil, nil)
	err = post.Create(client, model.CreationOptions{})
	if err == nil {
		t.Fatal("error should occur")
	}

	testutil.ShouldExistFile(t, 0)
}

func TestPostCreateWithTweetAndGist(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(405)
			b, _ := json.Marshal(api.ResponseError{"method_not_allowed", "Method Not Allowed"})
			w.Write(b)
			return
		}

		defer r.Body.Close()

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
		if string(b) == "" {
			testutil.ResponseAPIError(w, 500, api.ResponseError{
				Type:    "fatal",
				Message: "empty body",
			})
			return
		}

		var post model.CreationPost
		err = json.Unmarshal(b, &post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		if !post.Tweet || !post.Gist {
			testutil.ResponseError(w, 500, errors.New("tweet and gist should be true"))
			return
		}

		post.CreatedAt = model.Time{Time: time.Date(2016, 2, 1, 12, 51, 42, 0, time.UTC)}
		post.UpdatedAt = post.CreatedAt
		b, err = json.Marshal(post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		_, err = w.Write(b)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)

	prevPath := post.Path
	if err != nil {
		t.Fatal(err)
	}

	err = post.Create(client, model.CreationOptions{true, true})
	if err != nil {
		t.Fatal(err)
	}

	postPath := post.Path
	if err != nil {
		t.Fatal(err)
	}
	if postPath != prevPath {
		t.Errorf("wrong path: expected %s, but actual %s", prevPath, postPath)
	}

	if !post.CreatedAt.Equal(time.Date(2016, 2, 1, 12, 51, 42, 0, time.UTC)) {
		t.Errorf("wrong CreatedAt: %s", post.CreatedAt)
	}
	if !post.UpdatedAt.Equal(time.Date(2016, 2, 1, 12, 51, 42, 0, time.UTC)) {
		t.Errorf("wrong UpdatedAt: %s", post.UpdatedAt)
	}

	testutil.ShouldExistFile(t, 0)
}

func TestFetchPost(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items/4bd431809afb1bb99e4f", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(405)
			b, _ := json.Marshal(api.ResponseError{"method_not_allowed", "Method Not Allowed"})
			w.Write(b)
			return
		}

		body := `{
			"rendered_body": "<h2>Example body</h2>",
			"body": "## Example body",
			"coediting": false,
			"created_at": "2000-01-01T00:00:00+00:00",
			"id": "4bd431809afb1bb99e4f",
			"private": false,
			"tags": [
				{
					"name": "Ruby",
					"versions": [
						"0.0.1"
					]
				}
			],
			"title": "Example title",
			"updated_at": "2000-01-01T00:00:00+00:00",
			"url": "https://qiita.com/yaotti/items/4bd431809afb1bb99e4f",
			"user": {
				"description": "Hello, world.",
				"facebook_id": "yaotti",
				"followees_count": 100,
				"followers_count": 200,
				"github_login_name": "yaotti",
				"id": "yaotti",
				"items_count": 300,
				"linkedin_id": "yaotti",
				"location": "Tokyo, Japan",
				"name": "Hiroshige Umino",
				"organization": "Increments Inc",
				"permanent_id": 1,
				"profile_image_url": "https://si0.twimg.com/profile_images/2309761038/1ijg13pfs0dg84sk2y0h_normal.jpeg",
				"twitter_screen_name": "yaotti",
				"website_url": "http://yaotti.hatenablog.com"
			}
		}`
		w.Write([]byte(body))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	testutil.ShouldExistFile(t, 0)

	team := model.Team{
		Active: true,
		ID:     "increments",
		Name:   "Increments Inc",
	}
	post, err := model.FetchPost(client, &team, "4bd431809afb1bb99e4f")
	if err != nil {
		t.Fatal(err)
	}

	if post.RenderedBody != "<h2>Example body</h2>" {
		t.Errorf("wrong RenderedBody: %s", post.RenderedBody)
	}
	if post.Body != "## Example body" {
		t.Errorf("wrong Body: %s", post.Body)
	}
	if post.Coediting != false {
		t.Errorf("wrong Coediting: %b", post.Coediting)
	}
	if !post.CreatedAt.Equal(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("wrong CreatedAt: %s", post.CreatedAt)
	}
	if post.ID != "4bd431809afb1bb99e4f" {
		t.Errorf("wrong ID: %s", post.ID)
	}
	if post.Private != false {
		t.Errorf("wrong Private: %b", post.Private)
	}
	if post.Title != "Example title" {
		t.Errorf("wrong Title: %s", post.Title)
	}
	if !post.UpdatedAt.Equal(time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("wrong UpdatedAt: %s", post.UpdatedAt)
	}
	if post.URL != "https://qiita.com/yaotti/items/4bd431809afb1bb99e4f" {
		t.Errorf("wrong URL: %s", post.URL)
	}
	if post.User.Description != "Hello, world." {
		t.Errorf("wrong Description: %s", post.User.Description)
	}
	if post.User.FacebookID != "yaotti" {
		t.Errorf("wrong FacebookId: %s", post.User.FacebookID)
	}
	if post.User.FolloweesCount != 100 {
		t.Errorf("wrong FolloweesCount: %s", post.User.FolloweesCount)
	}
	if post.User.FollowersCount != 200 {
		t.Errorf("wrong FollowersCount: %s", post.User.FollowersCount)
	}
	if post.User.GithubLoginName != "yaotti" {
		t.Errorf("wrong GithubLoginName: %s", post.User.GithubLoginName)
	}
	if post.User.ID != "yaotti" {
		t.Errorf("wrong Id: %s", post.User.ID)
	}
	if post.User.ItemsCount != 300 {
		t.Errorf("wrong ItemsCount: %d", post.User.ItemsCount)
	}
	if post.User.LinkedinID != "yaotti" {
		t.Errorf("wrong LinkedinId: %s", post.User.LinkedinID)
	}
	if post.User.Location != "Tokyo, Japan" {
		t.Errorf("wrong Location: %s", post.User.Location)
	}
	if post.User.Name != "Hiroshige Umino" {
		t.Errorf("wrong Name: %s", post.User.Name)
	}
	if post.User.Organization != "Increments Inc" {
		t.Errorf("wrong Organization: %s", post.User.Organization)
	}
	if post.User.PermanentID != 1 {
		t.Errorf("wrong PermanentId: %d", post.User.PermanentID)
	}
	if post.User.ProfileImageURL != "https://si0.twimg.com/profile_images/2309761038/1ijg13pfs0dg84sk2y0h_normal.jpeg" {
		t.Errorf("wrong ProfileImageUrl: %s", post.User.ProfileImageURL)
	}
	if post.User.TwitterScreenName != "yaotti" {
		t.Errorf("wrong TwitterScreenName: %s", post.User.TwitterScreenName)
	}
	if post.User.WebsiteURL != "http://yaotti.hatenablog.com" {
		t.Errorf("wrong WebsiteUrl: %s", post.User.WebsiteURL)
	}
	if len(post.Tags) != 1 {
		t.Fatalf("wrong Tags length: %d", len(post.Tags))
	}
	if post.Tags[0].Name != "Ruby" {
		t.Errorf("wrong tag Name: %s", post.Tags[0].Name)
	}
	if len(post.Tags[0].Versions) != 1 {
		t.Fatalf("wrong tag Versions length: %d", len(post.Tags[0].Versions))
	}
	if post.Tags[0].Versions[0] != "0.0.1" {
		t.Errorf("wrong tag Versions: %s", post.Tags[0].Versions[0])
	}

	testutil.ShouldExistFile(t, 0)
}

func TestFetchPostWithResponseError(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items/4bd431809afb1bb99e4f", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{
  "message": "Not found",
  "type": "not_found"
}`))
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	testutil.ShouldExistFile(t, 0)

	_, err = model.FetchPost(client, nil, "4bd431809afb1bb99e4f")
	if err == nil {
		t.Fatal("error should occur")
	}
	_, ok := err.(api.ResponseError)
	if !ok {
		t.Fatalf("wrong type error: %s", reflect.TypeOf(err))
	}

	testutil.ShouldExistFile(t, 0)
}

func TestFetchPostWithResponseStatusError(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items/4bd431809afb1bb99e4f", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	testutil.ShouldExistFile(t, 0)

	_, err = model.FetchPost(client, nil, "4bd431809afb1bb99e4f")
	if err == nil {
		t.Fatal("error should occur")
	}
	_, ok := err.(api.StatusError)
	if !ok {
		t.Fatalf("wrong type error: %s", reflect.TypeOf(err))
	}

	testutil.ShouldExistFile(t, 0)
}

func TestFetchPostWithWrongResponseBody(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items/4bd431809afb1bb99e4f", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Non JSON format")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	_, err = model.FetchPost(client, nil, "4bd431809afb1bb99e4f")
	if err == nil {
		t.Fatal("error should occur")
	}
}

func TestPostUpdate(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items/abcdefghijklmnopqrst", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			w.WriteHeader(405)
			b, _ := json.Marshal(api.ResponseError{"method_not_allowed", "Method Not Allowed"})
			w.Write(b)
			return
		}

		defer r.Body.Close()

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
		if string(b) == "" {
			testutil.ResponseAPIError(w, 500, api.ResponseError{
				Type:    "fatal",
				Message: "empty body",
			})
			return
		}

		var post model.Post
		err = json.Unmarshal(b, &post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		post.UpdatedAt = model.Time{Time: time.Date(2016, 2, 1, 12, 51, 42, 0, time.UTC)}
		b, err = json.Marshal(post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		_, err = w.Write(b)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)
	post.ID = "abcdefghijklmnopqrst"

	prevPath := post.Path
	if err != nil {
		t.Fatal(err)
	}

	err = post.Update(client)
	if err != nil {
		t.Fatal(err)
	}

	postPath := post.Path
	if err != nil {
		t.Fatal(err)
	}
	if postPath != prevPath {
		t.Errorf("wrong path: expected %s, but actual %s", prevPath, postPath)
	}

	if !post.UpdatedAt.Equal(time.Date(2016, 2, 1, 12, 51, 42, 0, time.UTC)) {
		t.Errorf("wrong UpdatedAt: %s", post.UpdatedAt)
	}

	testutil.ShouldExistFile(t, 0)
}

func TestPostUpdateInTeam(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items/abcdefghijklmnopqrst", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "PATCH" {
			w.WriteHeader(405)
			b, _ := json.Marshal(api.ResponseError{"method_not_allowed", "Method Not Allowed"})
			w.Write(b)
			return
		}

		defer r.Body.Close()

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
		if string(b) == "" {
			testutil.ResponseAPIError(w, 500, api.ResponseError{
				Type:    "fatal",
				Message: "empty body",
			})
			return
		}

		var post model.Post
		err = json.Unmarshal(b, &post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		post.UpdatedAt = model.Time{Time: time.Date(2016, 2, 1, 12, 51, 42, 0, time.UTC)}
		b, err = json.Marshal(post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		_, err = w.Write(b)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		if subDomain != "increments" {
			t.Fatalf("wrong sub domain: %s", subDomain)
			return
		}
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, &model.Team{Active: true, ID: "increments", Name: "Increments Inc."})
	post.ID = "abcdefghijklmnopqrst"

	prevPath := post.Path
	if err != nil {
		t.Fatal(err)
	}

	err = post.Update(client)
	if err != nil {
		t.Fatal(err)
	}

	postPath := post.Path
	if err != nil {
		t.Fatal(err)
	}
	if postPath != prevPath {
		t.Errorf("wrong path: expected %s, but actual %s", prevPath, postPath)
	}

	if !post.UpdatedAt.Equal(time.Date(2016, 2, 1, 12, 51, 42, 0, time.UTC)) {
		t.Errorf("wrong UpdatedAt: %s", post.UpdatedAt)
	}

	testutil.ShouldExistFile(t, 0)
}

func TestPostUpdateWithEmptyID(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		log.Fatal(err)
	}
	client := api.NewClient(nil, inf)

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", nil, nil)
	err = post.Update(client)
	err, ok := err.(model.EmptyIDError)
	if !ok {
		t.Fatal("empty ID error should occur")
	}

	testutil.ShouldExistFile(t, 0)
}

func TestPostUpdateWithNoServer(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	post := model.NewPost("Example Title", nil, nil)
	post.ID = "abcdefghijklmnopqrst"
	err = post.Update(client)
	if err == nil {
		t.Fatal("error should occur")
	}
}

func TestPostUpdateWithWrongResponseBody(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items/abcdefghijklmnopqrst", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Non JSON format")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	post := model.NewPost("Example Title", nil, nil)
	post.ID = "abcdefghijklmnopqrst"
	err = post.Update(client)
	if err == nil {
		t.Fatal("error should occur")
	}
}

func TestPostDelete(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items/abcdefghijklmnopqrst", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			w.WriteHeader(405)
			b, _ := json.Marshal(api.ResponseError{"method_not_allowed", "Method Not Allowed"})
			w.Write(b)
			return
		}

		defer r.Body.Close()

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
		if string(b) == "" {
			testutil.ResponseAPIError(w, 500, api.ResponseError{
				Type:    "fatal",
				Message: "empty body",
			})
			return
		}

		var post model.Post
		err = json.Unmarshal(b, &post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		b, err = json.Marshal(post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		_, err = w.Write(b)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)
	post.ID = "abcdefghijklmnopqrst"

	prevPath := post.Path
	if err != nil {
		t.Fatal(err)
	}

	err = post.Delete(client)
	if err != nil {
		t.Fatal(err)
	}

	postPath := post.Path
	if err != nil {
		t.Fatal(err)
	}
	if postPath != prevPath {
		t.Errorf("wrong path: expected %s, but actual %s", prevPath, postPath)
	}

	testutil.ShouldExistFile(t, 0)
}

func TestPostDeleteInTeam(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items/abcdefghijklmnopqrst", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "DELETE" {
			w.WriteHeader(405)
			b, _ := json.Marshal(api.ResponseError{"method_not_allowed", "Method Not Allowed"})
			w.Write(b)
			return
		}

		defer r.Body.Close()

		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
		if string(b) == "" {
			testutil.ResponseAPIError(w, 500, api.ResponseError{
				Type:    "fatal",
				Message: "empty body",
			})
			return
		}

		var post model.Post
		err = json.Unmarshal(b, &post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		b, err = json.Marshal(post)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}

		_, err = w.Write(b)
		if err != nil {
			testutil.ResponseError(w, 500, err)
			return
		}
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		if subDomain != "increments" {
			t.Fatalf("wrong sub domain: %s", subDomain)
			return
		}
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, &model.Team{Active: true, ID: "increments", Name: "Increments Inc."})
	post.ID = "abcdefghijklmnopqrst"

	prevPath := post.Path
	if err != nil {
		t.Fatal(err)
	}

	err = post.Delete(client)
	if err != nil {
		t.Fatal(err)
	}

	postPath := post.Path
	if err != nil {
		t.Fatal(err)
	}
	if postPath != prevPath {
		t.Errorf("wrong path: expected %s, but actual %s", prevPath, postPath)
	}

	testutil.ShouldExistFile(t, 0)
}

func TestPostDeleteWithNoServer(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)
	post.ID = "abcdefghijklmnopqrst"
	err = post.Delete(client)
	if err == nil {
		t.Fatal("error should occur")
	}

	testutil.ShouldExistFile(t, 0)
}

func TestPostDeleteWithEmptyID(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		log.Fatal(err)
	}
	client := api.NewClient(nil, inf)

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)
	err = post.Delete(client)
	err, ok := err.(model.EmptyIDError)
	if !ok {
		t.Fatal("empty ID error should occur")
	}

	testutil.ShouldExistFile(t, 0)
}

func TestPostDeleteWithWrongResponseBody(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/items/abcdefghijklmnopqrst", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Non JSON format")
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	err := os.Setenv("QIITA_ACCESS_TOKEN", "XXXXXXXXXXXX")
	if err != nil {
		t.Fatal(err)
	}

	client := api.NewClient(func(subDomain, path string) (url string) {
		url = fmt.Sprintf("%s%s%s", server.URL, "/api/v2", path)
		return
	}, inf)

	post := model.NewPost("Example Title", nil, nil)
	post.ID = "abcdefghijklmnopqrst"
	err = post.Delete(client)
	// Don't parse response body from DELETE method
	if err != nil {
		t.Fatal(err)
	}
}

func TestPostSave(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", &model.Time{time.Date(2015, 11, 28, 13, 2, 37, 0, time.UTC)}, nil)
	post.ID = "abcdefghijklmnopqrst"
	err := post.Save(nil)
	if err != nil {
		t.Fatal(err)
	}

	testutil.ShouldExistFile(t, 1)

	func() {
		a, err := ioutil.ReadFile("mine/2015/11/28/Example Title.md")
		if err != nil {
			t.Fatal(err)
		}
		actual := string(a)
		expected := `<!--
id: abcdefghijklmnopqrst
url: ""
created_at: 2015-11-28T22:02:37+09:00
updated_at: 2015-11-28T22:02:37+09:00
private: false
coediting: false
tags: []
team: null
-->

# Example Title

`
		if actual != expected {
			t.Errorf("wrong content:\n%s", testutil.Diff(expected, actual))
		}
	}()

	post.Title = "Example Edited Title"
	post.CreatedAt = model.Time{time.Date(2015, 12, 28, 13, 2, 37, 0, time.UTC)}
	post.UpdatedAt = post.CreatedAt
	err = post.Save(nil)
	if err != nil {
		t.Fatal(err)
	}

	testutil.ShouldExistFile(t, 1)

	func() {
		_, err := os.Stat("mine/2015/12/28/Example Edited Title.md")
		if err == nil {
			t.Errorf("filename based on edited post shouldn't exist: %s", "mine/2015/12/28/Example Edited Title.md")
		}
	}()

	func() {
		a, err := ioutil.ReadFile("mine/2015/11/28/Example Title.md")
		if err != nil {
			t.Fatal(err)
		}
		actual := string(a)
		expected := `<!--
id: abcdefghijklmnopqrst
url: ""
created_at: 2015-12-28T22:02:37+09:00
updated_at: 2015-12-28T22:02:37+09:00
private: false
coediting: false
tags: []
team: null
-->

# Example Edited Title

`
		if actual != expected {
			t.Errorf("wrong content:\n%s", testutil.Diff(expected, actual))
		}
	}()
}

func TestPostSaveWithTeam(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", &model.Time{time.Date(2015, 11, 28, 13, 2, 37, 0, time.UTC)}, &model.Team{Active: true, ID: "increments", Name: "Increments Inc."})
	post.ID = "abcdefghijklmnopqrst"
	err := post.Save(nil)
	if err != nil {
		t.Fatal(err)
	}

	testutil.ShouldExistFile(t, 1)

	func() {
		a, err := ioutil.ReadFile("increments/2015/11/28/Example Title.md")
		if err != nil {
			t.Fatal(err)
		}
		actual := string(a)
		expected := `<!--
id: abcdefghijklmnopqrst
url: ""
created_at: 2015-11-28T22:02:37+09:00
updated_at: 2015-11-28T22:02:37+09:00
private: false
coediting: false
tags: []
team:
  active: true
  id: increments
  name: Increments Inc.
-->

# Example Title

`
		if actual != expected {
			t.Errorf("wrong content:\n%s", testutil.Diff(expected, actual))
		}
	}()

	post.Title = "Example Edited Title"
	post.CreatedAt = model.Time{time.Date(2015, 12, 28, 13, 2, 37, 0, time.UTC)}
	post.UpdatedAt = post.CreatedAt
	err = post.Save(nil)
	if err != nil {
		t.Fatal(err)
	}

	testutil.ShouldExistFile(t, 1)

	func() {
		_, err := os.Stat("increments/2015/12/28/Example Edited Title.md")
		if err == nil {
			t.Errorf("filename based on edited post shouldn't exist: %s", "mine/2015/12/28/Example Edited Title.md")
		}
	}()

	func() {
		a, err := ioutil.ReadFile("increments/2015/11/28/Example Title.md")
		if err != nil {
			t.Fatal(err)
		}
		actual := string(a)
		expected := `<!--
id: abcdefghijklmnopqrst
url: ""
created_at: 2015-12-28T22:02:37+09:00
updated_at: 2015-12-28T22:02:37+09:00
private: false
coediting: false
tags: []
team:
  active: true
  id: increments
  name: Increments Inc.
-->

# Example Edited Title

`
		if actual != expected {
			t.Errorf("wrong content:\n%s", testutil.Diff(expected, actual))
		}
	}()
}

func TestPostSaveWithPath(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	testutil.ShouldExistFile(t, 0)

	post := model.NewPost("Example Title", &model.Time{time.Date(2015, 11, 28, 13, 2, 37, 0, time.UTC)}, nil)
	post.ID = "abcdefghijklmnopqrst"
	post.Path = "foo/bar.md"
	err := post.Save(nil)
	if err != nil {
		t.Fatal(err)
	}

	testutil.ShouldExistFile(t, 1)

	func() {
		a, err := ioutil.ReadFile("foo/bar.md")
		if err != nil {
			t.Fatal(err)
		}
		actual := string(a)
		expected := `<!--
id: abcdefghijklmnopqrst
url: ""
created_at: 2015-11-28T22:02:37+09:00
updated_at: 2015-11-28T22:02:37+09:00
private: false
coediting: false
tags: []
team: null
-->

# Example Title

`
		if actual != expected {
			t.Errorf("wrong content:\n%s", testutil.Diff(expected, actual))
		}
	}()
}

func TestPostSaveWithInvalidMarkdown(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	err := os.MkdirAll("foo", 0755)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("foo/wrong-format.md", []byte{}, 0644)
	if err != nil {
		t.Fatal(err)
	}
	err = ioutil.WriteFile("foo/no-id.md", []byte(`<!--
id: ""
url: ""
created_at: 2015-11-28T22:02:37+09:00
updated_at: 2015-11-28T22:02:37+09:00
private: false
coediting: false
tags: []
team: null
-->

# Example Title

`), 0644)
	if err != nil {
		t.Fatal(err)
	}
	if err != nil {
		t.Fatal(err)
	}

	testutil.ShouldExistFile(t, 2)

	func() {
		post := model.NewPost("Example Title", &model.Time{time.Date(2015, 11, 28, 13, 2, 37, 0, time.UTC)}, nil)
		post.ID = "abcdefghijklmnopqrst"
		post.Path = "foo/bar.md"
		err := post.Save(nil)
		if err != nil {
			t.Fatal(err)
		}
	}()

	testutil.ShouldExistFile(t, 3)

	func() {
		a, err := ioutil.ReadFile("foo/bar.md")
		if err != nil {
			t.Fatal(err)
		}
		actual := string(a)
		expected := `<!--
id: abcdefghijklmnopqrst
url: ""
created_at: 2015-11-28T22:02:37+09:00
updated_at: 2015-11-28T22:02:37+09:00
private: false
coediting: false
tags: []
team: null
-->

# Example Title

`
		if actual != expected {
			t.Errorf("wrong content:\n%s", testutil.Diff(expected, actual))
		}
	}()
}

func TestPostSaveDuplicationWithID(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	testutil.ShouldExistFile(t, 0)

	func() {
		post := model.NewPost("Example Title", &model.Time{time.Date(2015, 11, 28, 13, 2, 37, 0, time.UTC)}, nil)
		post.ID = "abcdefghijklmnopqrst"
		err := post.Save(nil)
		if err != nil {
			t.Fatal(err)
		}
	}()

	func() {
		post := model.NewPost("Example Title", &model.Time{time.Date(2015, 11, 28, 13, 2, 37, 0, time.UTC)}, nil)
		post.ID = "abcdefghijklmnopqrst"
		err := post.Save(nil)
		if err != nil {
			t.Fatal(err)
		}
	}()

	testutil.ShouldExistFile(t, 1)
}

func TestPostSaveDuplicationWithoutID(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	testutil.ShouldExistFile(t, 0)

	func() {
		post := model.NewPost("Example Title", &model.Time{time.Date(2015, 11, 28, 13, 2, 37, 0, time.UTC)}, nil)
		err := post.Save(nil)
		if err != nil {
			t.Fatal(err)
		}
	}()

	func() {
		post := model.NewPost("Example Title", &model.Time{time.Date(2015, 11, 28, 13, 2, 37, 0, time.UTC)}, nil)
		err := post.Save(nil)
		if err != nil {
			t.Fatal(err)
		}
	}()

	testutil.ShouldExistFile(t, 2)
}

func TestPostEncodeWithNewPost(t *testing.T) {
	post := model.NewPost("Example title", &model.Time{time.Date(2016, 2, 2, 6, 30, 46, 0, time.UTC)}, nil)
	post.ID = "4bd431809afb1bb99e4f"
	post.URL = "https://qiita.com/yaotti/items/4bd431809afb1bb99e4f"
	buf := bytes.NewBuffer([]byte{})
	err := post.Encode(buf)
	if err != nil {
		t.Fatal(err)
	}
	actual := string(buf.Bytes())
	expected := `<!--
id: 4bd431809afb1bb99e4f
url: https://qiita.com/yaotti/items/4bd431809afb1bb99e4f
created_at: 2016-02-02T15:30:46+09:00
updated_at: 2016-02-02T15:30:46+09:00
private: false
coediting: false
tags: []
team: null
-->

# Example title

`
	if expected != actual {
		t.Errorf("wrong content:\n%s", testutil.Diff(expected, actual))
	}
}

func TestPostDecodeWithWrongMeta(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	var post model.Post
	err := post.Decode([]byte(`XXXXXXXX
<!--
id: abcdefghijklmnopqrst
url: http://example.com/mypost
created_at: 2013-12-10T12:29:14+09:00
updated_at: 2015-02-25T09:26:30+09:00
private: true
coediting: false
tags:
- TypeScript
- Docker:
  - 1.9
- Go:
  - 1.4.3
  - 1.5.3
team: null
-->

# Main title

## Sub title
Paragraph
`))
	if err == nil {
		t.Errorf("start without meta comment should return error")
	}
}

func TestPostDecodeWithMultiComment(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	var post model.Post
	err := post.Decode([]byte(`<!--
id: ""
url: ""
created_at: 2016-02-17T13:08:30+09:00
updated_at: 2016-02-17T13:08:30+09:00
private: false
coediting: false
tags:
- Qiita
- Go
team: null
-->

# Example Title

<!--
This is not meta
-->

# This is not title of this post
`))
	if err != nil {
		t.Fatal(err)
	}
}

func TestPostDecodeWithWrongTag(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	var post model.Post
	err := post.Decode([]byte(`<!--
id: abcdefghijklmnopqrst
url: http://example.com/mypost
created_at: 2013-12-10T12:29:14+09:00
updated_at: 2015-02-25T09:26:30+09:00
private: true
coediting: false
tags:
  TypeScript: []
  Docker:
    - 1.9
  Go:
    - 1.4.3
    - 1.5.3
team: null
-->

# Main title

## Sub title
Paragraph
`))
	if err == nil {
		t.Errorf("should return error objective tags")
	}
}

func TestPostDecodeWithWrongTitle(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	var post model.Post
	err := post.Decode([]byte(`<!--
id: abcdefghijklmnopqrst
url: http://example.com/mypost
created_at: 2013-12-10T12:29:14+09:00
updated_at: 2015-02-25T09:26:30+09:00
private: true
coediting: false
tags:
- TypeScript
- Docker:
  - 1.9
- Go:
  - 1.4.3
  - 1.5.3
team: null
-->

## Sub title
# Main title

Paragraph
`))
	if err == nil {
		t.Errorf("should return error with non-object element in tags")
	}
}

func TestPostDecodeWithCorrectMarkdown(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	var post model.Post
	err := post.Decode([]byte(`<!--
id: abcdefghijklmnopqrst
url: http://example.com/mypost
created_at: 2013-12-10T12:29:14+09:00
updated_at: 2015-02-25T09:26:30+09:00
private: true
coediting: false
tags:
- TypeScript
- Docker:
  - 1.9
- Go:
  - 1.4.3
  - 1.5.3
team: null
-->

# Main title

## Sub title
Paragraph
`))
	if err != nil {
		t.Fatal(err)
	}

	if post.Meta.ID != "abcdefghijklmnopqrst" {
		t.Errorf("wrong Id")
	}
	if post.Meta.URL != "http://example.com/mypost" {
		t.Errorf("wrong Url")
	}
	if !post.Meta.CreatedAt.Equal(time.Date(2013, 12, 10, 3, 29, 14, 0, time.UTC)) {
		t.Errorf("wrong CreatedAt")
	}
	if !post.Meta.UpdatedAt.Equal(time.Date(2015, 02, 25, 0, 26, 30, 0, time.UTC)) {
		t.Errorf("wrong UpdatedAt")
	}
	if post.Meta.Private != true {
		t.Errorf("wrong Private")
	}
	if post.Meta.Coediting != false {
		t.Errorf("wrong Coediting")
	}
	if len(post.Meta.Tags) != 3 {
		t.Errorf("wrong Tags length: %d", len(post.Meta.Tags))
	} else {
		for _, tag := range post.Meta.Tags {
			switch tag.Name {
			case "TypeScript":
				if len(tag.Versions) != 0 {
					t.Errorf("wrong Tag with no version: %+v", tag)
				}
			case "Docker":
				if len(tag.Versions) != 1 || tag.Versions[0] != "1.9" {
					t.Errorf("wrong Tag with single version: %+v", tag)
				}
			case "Go":
				if len(tag.Versions) != 2 || tag.Versions[0] != "1.4.3" || tag.Versions[1] != "1.5.3" {
					t.Errorf("wrong Tag with multi versions: %+v", tag)
				}
			}
		}
	}
	if post.Title != "Main title" {
		t.Errorf("wrong Title")
	}
	if post.Body != "## Sub title\nParagraph" {
		t.Errorf("wrong Body: %s", post.Body)
	}
}

func TestPostValidateWithoutTeam(t *testing.T) {
	func() {
		post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)
		post.Body = "Example Body"
		post.Tags = model.Tags{
			model.Tag{
				Name: "Go",
			},
		}
		err := post.Validate()
		if err != nil {
			t.Error("should be valid")
		}
	}()

	func() {
		post := model.NewPost("", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)
		post.Body = "Example Body"
		post.Tags = model.Tags{
			model.Tag{
				Name: "Go",
			},
		}
		err := post.Validate()
		if err == nil {
			t.Error("should be invalid")
		}
		for name, status := range err {
			switch name {
			case "title":
				if !status.Required {
					t.Errorf("%s should be required", name)
				}
			default:
				t.Errorf("%s should be valid: %s", name, status)
			}
		}
	}()

	func() {
		post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)
		post.Tags = model.Tags{
			model.Tag{
				Name: "Go",
			},
		}
		err := post.Validate()
		if err == nil {
			t.Error("should be invalid")
		}
		for name, status := range err {
			switch name {
			case "body":
				if !status.Required {
					t.Errorf("%s should be required", name)
				}
			default:
				t.Errorf("%s should be valid: %s", name, status)
			}
		}
	}()

	func() {
		post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)
		post.Body = "Example Body"
		err := post.Validate()
		if err == nil {
			t.Error("should be invalid")
		}
		for name, status := range err {
			switch name {
			case "tags":
				if !status.Required {
					t.Errorf("%s should be required", name)
				}
			default:
				t.Errorf("%s should be valid: %s", name, status)
			}
		}
	}()

	func() {
		post := model.NewPost("", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)
		err := post.Validate()
		if err == nil {
			t.Error("should be invalid")
		}
		for name, status := range err {
			switch name {
			case "title":
				if !status.Required {
					t.Errorf("%s should be required", name)
				}
			case "body":
				if !status.Required {
					t.Errorf("%s should be required", name)
				}
			case "tags":
				if !status.Required {
					t.Errorf("%s should be required", name)
				}
			default:
				t.Errorf("%s should be valid: %s", name, status)
			}
		}
	}()
}

func TestPostValidateWithTeam(t *testing.T) {
	func() {
		post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)
		post.Body = "Example Body"
		post.Team = &model.Team{
			Active: true,
			ID:     "increments",
			Name:   "Increments Inc.",
		}
		err := post.Validate()
		if err != nil {
			t.Error("should be valid")
		}
	}()

	func() {
		post := model.NewPost("", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)
		post.Body = "Example Body"
		post.Team = &model.Team{
			Active: true,
			ID:     "increments",
			Name:   "Increments Inc.",
		}
		err := post.Validate()
		if err == nil {
			t.Error("should be invalid")
		}
		for name, status := range err {
			switch name {
			case "title":
				if !status.Required {
					t.Errorf("%s should be required", name)
				}
			default:
				t.Errorf("%s should be valid: %s", name, status)
			}
		}
	}()

	func() {
		post := model.NewPost("Example Title", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)
		post.Team = &model.Team{
			Active: true,
			ID:     "increments",
			Name:   "Increments Inc.",
		}
		err := post.Validate()
		if err == nil {
			t.Error("should be invalid")
		}
		for name, status := range err {
			switch name {
			case "body":
				if !status.Required {
					t.Errorf("%s should be required", name)
				}
			default:
				t.Errorf("%s should be valid: %s", name, status)
			}
		}
	}()

	func() {
		post := model.NewPost("", &model.Time{time.Date(2000, 1, 1, 9, 0, 0, 0, time.UTC)}, nil)
		post.Team = &model.Team{
			Active: true,
			ID:     "increments",
			Name:   "Increments Inc.",
		}
		err := post.Validate()
		if err == nil {
			t.Error("should be invalid")
		}
		for name, status := range err {
			switch name {
			case "title":
				if !status.Required {
					t.Errorf("%s should be required", name)
				}
			case "body":
				if !status.Required {
					t.Errorf("%s should be required", name)
				}
			default:
				t.Errorf("%s should be valid: %s", name, status)
			}
		}
	}()
}

func TestEmptyIDError(t *testing.T) {
	err := model.EmptyIDError{}
	if !strings.HasPrefix(err.Error(), "empty ID") {
		t.Errorf("wrong error: %s", err.Error())
	}
}

func TestInvalidError(t *testing.T) {
	func() {
		err := model.InvalidError{}
		if err.Error() != "Valid" {
			t.Errorf("wrong message: %s", err)
		}
	}()

	func() {
		err := model.InvalidError{}
		err["foo"] = model.InvalidStatus{
			Name:     "foo",
			Required: true,
		}
		actual := err.Error()
		expected := `A field is invalid:
- foo
  - shouldn't be empty`
		if actual != expected {
			t.Errorf("wrong message:\n%s", testutil.Diff(expected, actual))
		}
	}()
}
