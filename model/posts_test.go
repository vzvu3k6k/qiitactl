package model_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/minodisk/qiitactl/api"
	"github.com/minodisk/qiitactl/model"
	"github.com/minodisk/qiitactl/testutil"
)

func TestFetchPostsWithTotalCount(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/authenticated_user/items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(405)
			b, _ := json.Marshal(api.ResponseError{"method_not_allowed", "Method Not Allowed"})
			w.Write(b)
			return
		}
		total := 1422
		q := r.URL.Query()
		perPage, err := strconv.Atoi(q.Get("per_page"))
		if err != nil {
			perPage = 10
		}
		page, err := strconv.Atoi(q.Get("page"))
		if err != nil {
			page = 1
		}

		var posts []model.Post

		from := perPage*(page-1) + 1
		if from <= total {
			to := perPage * page
			if to > total {
				to = total
			}
			for i := from; i <= to; i++ {
				post := model.Post{Meta: model.Meta{ID: fmt.Sprint(i)}}
				posts = append(posts, post)
			}
		} else {
			testutil.ResponseError(w, 500, fmt.Errorf("shouldn't access over total count"))
			return
		}

		b, _ := json.Marshal(posts)
		w.Header().Set("Total-Count", fmt.Sprint(total))
		w.Write(b)
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

	team := model.Team{
		Active: true,
		ID:     "increments",
		Name:   "Increments Inc",
	}
	posts, err := model.FetchPosts(client, &team)
	if err != nil {
		t.Fatal(err)
	}

	if len(posts) != 1422 {
		t.Errorf("wrong posts length: %d", len(posts))
	}
}

func TestFetchPostsWithWrongTotalCount(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/authenticated_user/items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(405)
			b, _ := json.Marshal(api.ResponseError{"method_not_allowed", "Method Not Allowed"})
			w.Write(b)
			return
		}

		var posts []model.Post
		b, _ := json.Marshal(posts)
		w.Header().Set("Total-Count", "dummy")
		w.Write(b)
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

	_, err = model.FetchPosts(client, nil)
	if err == nil {
		t.Fatal("error should occur")
	}
}

func TestFetchPosts(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/authenticated_user/items", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(405)
			b, _ := json.Marshal(api.ResponseError{"method_not_allowed", "Method Not Allowed"})
			w.Write(b)
			return
		}
		var body string
		if r.URL.Query().Get("page") == "1" {
			body = `[
				{
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
				}
			]`
		} else {
			testutil.ResponseError(w, 500, fmt.Errorf("shouldn't access over total count"))
			return
		}
		w.Header().Set("Total-Count", fmt.Sprint(1))
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

	team := model.Team{
		Active: true,
		ID:     "increments",
		Name:   "Increments Inc",
	}
	posts, err := model.FetchPosts(client, &team)
	if err != nil {
		t.Fatal(err)
	}

	if len(posts) != 1 {
		t.Fatalf("wrong length: expected %d, but actual %d", 1, len(posts))
	}

	post := posts[0]
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
}

func TestFetchPostsResponseError(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/authenticated_user/items", func(w http.ResponseWriter, r *http.Request) {
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

	_, err = model.FetchPosts(client, nil)
	if err == nil {
		t.Fatal("error should occur")
	}
	_, ok := err.(api.ResponseError)
	if !ok {
		t.Fatalf("wrong type error: %s", reflect.TypeOf(err))
	}
}

func TestFetchPostsWithResponseStatusError(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/authenticated_user/items", func(w http.ResponseWriter, r *http.Request) {
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

	_, err = model.FetchPosts(client, nil)
	if err == nil {
		t.Fatal("error should occur")
	}
	_, ok := err.(api.StatusError)
	if !ok {
		t.Fatalf("wrong type error: %s", reflect.TypeOf(err))
	}
}

func TestFetchPostsWitWrongResponseBody(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v2/authenticated_user/items", func(w http.ResponseWriter, r *http.Request) {
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

	_, err = model.FetchPosts(client, nil)
	if err == nil {
		t.Fatal("error should occur")
	}
}

func TestPostsSave(t *testing.T) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	post0 := model.NewPost("Example Title 0", &model.Time{time.Date(2015, 11, 28, 13, 2, 37, 0, time.UTC)}, nil)
	post1 := model.NewPost("Example Title 1", &model.Time{time.Date(2016, 2, 1, 5, 21, 49, 0, time.UTC)}, nil)

	posts := model.Posts{post0, post1}
	err := posts.Save()
	if err != nil {
		t.Fatal(err)
	}

	func() {
		a, err := ioutil.ReadFile("mine/2015/11/28/Example Title 0.md")
		if err != nil {
			t.Fatal(err)
		}
		actual := string(a)
		expected := `<!--
id: ""
url: ""
created_at: 2015-11-28T22:02:37+09:00
updated_at: 2015-11-28T22:02:37+09:00
private: false
coediting: false
tags: []
team: null
-->

# Example Title 0

`
		if actual != expected {
			t.Errorf("wrong content:\n%s", testutil.Diff(expected, actual))
		}
	}()

	func() {
		a, err := ioutil.ReadFile("mine/2016/02/01/Example Title 1.md")
		if err != nil {
			t.Fatal(err)
		}
		actual := string(a)
		expected := `<!--
id: ""
url: ""
created_at: 2016-02-01T14:21:49+09:00
updated_at: 2016-02-01T14:21:49+09:00
private: false
coediting: false
tags: []
team: null
-->

# Example Title 1

`
		if actual != expected {
			t.Errorf("wrong content:\n%s", testutil.Diff(expected, actual))
		}
	}()
}

func BenchmarkPostsSave(b *testing.B) {
	testutil.CleanUp()
	defer testutil.CleanUp()

	var posts model.Posts
	for i := 0; i < b.N; i++ {
		posts = append(posts, model.NewPost(fmt.Sprintf("Example Title %d", i), nil, nil))
	}

	b.ResetTimer()

	err := posts.Save()
	if err != nil {
		b.Fatal(err)
	}
}
